package main

import (
	"embed"
	"encoding/json"
	// "fmt"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	bolt "go.etcd.io/bbolt"
)

//go:embed static/*
var staticFS embed.FS

var (
	db        *bolt.DB
	wsClients = make(map[*websocket.Conn]bool)
	wsLock    sync.Mutex
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	prevNet   = make(map[string]net.IOCountersStat)
	prevTime  = time.Now()
)

type NetTotals struct {
	DailyIn    float64 `json:"daily_in"`
	DailyOut   float64 `json:"daily_out"`
	MonthlyIn  float64 `json:"monthly_in"`
	MonthlyOut float64 `json:"monthly_out"`
}

func main() {
	var err error
	db, err = bolt.Open("statsee.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

	// Serve index.html at /
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(subFS, "index.html")
		if err != nil {
			http.Error(w, "index.html not found", 500)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	http.HandleFunc("/api/network-totals", func(w http.ResponseWriter, r *http.Request) {
		totals := getNetworkTotals()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(totals)
	})

	http.HandleFunc("/ws", wsHandler)

	// Start background stats collector
	go startCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	wsLock.Lock()
	wsClients[conn] = true
	wsLock.Unlock()

	defer func() {
		wsLock.Lock()
		delete(wsClients, conn)
		wsLock.Unlock()
		conn.Close()
	}()

	for {
		var msg map[string]string
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}
		if msg["type"] == "speedtest" {
			go runSpeedTest(conn)
		}
	}
}

func startCollector() {
	ticker := time.NewTicker(1 * time.Second)
	for range ticker.C {
		collectStats()
	}
}

func collectStats() {
	ts := time.Now().Unix()

	// CPU
	cpuPercent, _ := cpu.Percent(0, false)

	// RAM
	memStat, _ := mem.VirtualMemory()

	// Disk
	diskIO, _ := disk.IOCounters()

	// Network
	netIO, _ := net.IOCounters(true)
	elapsed := time.Since(prevTime).Seconds()
	netRates := make(map[string]map[string]float64)

	for _, nic := range netIO {
		if nic.Name != "eth0" {
			continue
		}
		prev, ok := prevNet[nic.Name]
		if !ok {
			prev = nic
		}
		rxRate := float64(nic.BytesRecv-prev.BytesRecv) / elapsed / 1024 / 1024
		txRate := float64(nic.BytesSent-prev.BytesSent) / elapsed / 1024 / 1024
		netRates[nic.Name] = map[string]float64{
			"rate_recv": rxRate,
			"rate_sent": txRate,
		}
		prevNet[nic.Name] = nic
	}
	prevTime = time.Now()

	// Store totals in DB
	db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("net"))
		for _, nic := range netIO {
			key := []byte(nic.Name)
			v := map[string]float64{
				"in":  float64(nic.BytesRecv) / 1024 / 1024 / 1024,
				"out": float64(nic.BytesSent) / 1024 / 1024 / 1024,
			}
			data, _ := json.Marshal(v)
			b.Put(key, data)
		}
		return nil
	})

	// Broadcast to websocket clients
	msg := map[string]interface{}{
		"type": "stats",
		"ts":   ts,
		"cpu":  cpuPercent[0],
		"ram": map[string]float64{
			"used": float64(memStat.Used) / 1024 / 1024,
			"free": float64(memStat.Available) / 1024 / 1024,
		},
		"disk": diskIO,
		"net":  netRates,
	}

	wsLock.Lock()
	for c := range wsClients {
		c.WriteJSON(msg)
	}
	wsLock.Unlock()
}

func getNetworkTotals() map[string]NetTotals {
	totals := make(map[string]NetTotals)
	db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("net"))
		if b == nil {
			return nil
		}
		b.ForEach(func(k, v []byte) error {
			var data map[string]float64
			json.Unmarshal(v, &data)
			name := string(k)
			totals[name] = NetTotals{
				DailyIn:    data["in"],
				DailyOut:   data["out"],
				MonthlyIn:  data["in"],
				MonthlyOut: data["out"],
			}
			return nil
		})
		return nil
	})
	return totals
}

func runSpeedTest(conn *websocket.Conn) {
	const duration = 10 * time.Second
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	start := time.Now()
	var downloads, uploads []float64

	for range ticker.C {
		if time.Since(start) > duration {
			break
		}
		// Simulated speeds (replace with real network test if needed)
		d := 50 + rand.Float64()*200
		u := 20 + rand.Float64()*100
		downloads = append(downloads, d)
		uploads = append(uploads, u)
		conn.WriteJSON(map[string]interface{}{"type": "speedtest_update", "download": d, "upload": u})
	}

	avgDownload := average(downloads)
	avgUpload := average(uploads)
	conn.WriteJSON(map[string]interface{}{"type": "speedtest_done", "download": avgDownload, "upload": avgUpload})
}

func average(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}