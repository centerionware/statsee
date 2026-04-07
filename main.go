package main

import (
	"embed"
	"encoding/json"
	"fmt"
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
)

type NetTotals struct {
	DailyIn    float64 `json:"daily_in"`
	DailyOut   float64 `json:"daily_out"`
	MonthlyIn  float64 `json:"monthly_in"`
	MonthlyOut float64 `json:"monthly_out"`
}

var prevNet = make(map[string]net.IOCountersStat)
var prevTime = time.Now()

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

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(subFS, "index.html ")
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

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
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
	})

	go startCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func startCollector() {
	ticker := time.NewTicker(5 * time.Second)
	for range ticker.C {
		collectStats()
	}
}

func collectStats() {
	ts := time.Now().Unix()
	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()
	diskIO, _ := disk.IOCounters()
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
		rxRate := float64(nic.BytesRecv-prev.BytesRecv) / 1024 / 1024 / elapsed
		txRate := float64(nic.BytesSent-prev.BytesSent) / 1024 / 1024 / elapsed

		netRates[nic.Name] = map[string]float64{
			"rate_recv": rxRate,
			"rate_sent": txRate,
		}

		prevNet[nic.Name] = nic
	}
	prevTime = time.Now()

	db.Update(func(tx *bolt.Tx) error {
		b, _ := tx.CreateBucketIfNotExists([]byte("net"))
		for _, nic := range netIO {
			key := []byte(nic.Name)
			v := map[string]float64{
				"in":  float64(nic.BytesRecv),
				"out": float64(nic.BytesSent),
			}
			data, _ := json.Marshal(v)
			b.Put(key, data)
		}
		return nil
	})

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
			dailyGB := data["in"] / 1024 / 1024 / 1024
			monthlyGB := data["in"] / 1024 / 1024 / 1024
			totals[name] = NetTotals{
				DailyIn:    dailyGB,
				DailyOut:   data["out"] / 1024 / 1024 / 1024,
				MonthlyIn:  monthlyGB,
				MonthlyOut: data["out"] / 1024 / 1024 / 1024,
			}
			return nil
		})
		return nil
	})
	return totals
}

func runSpeedTest(conn *websocket.Conn) {
	for i := 0; i < 20; i++ {
		download := 50 + rand.Float64()*200
		upload := 20 + rand.Float64()*100
		latency := 5 + rand.Float64()*50
		conn.WriteJSON(map[string]interface{}{"type": "download", "value": download})
		conn.WriteJSON(map[string]interface{}{"type": "upload", "value": upload})
		conn.WriteJSON(map[string]interface{}{"type": "latency", "value": latency})
		time.Sleep(1 * time.Second)
	}
}