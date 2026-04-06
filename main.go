package main

import (
	"embed"
	"encoding/json"
	// "fmt"
	"io/fs"
	"log"
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
	DailyIn   float64 `json:"daily_in"`
	DailyOut  float64 `json:"daily_out"`
	MonthlyIn float64 `json:"monthly_in"`
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

	// Serve static files under /static/
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))

	// API endpoint for network totals
	http.HandleFunc("/api/network-totals", func(w http.ResponseWriter, r *http.Request) {
		totals := getNetworkTotals()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(totals)
	})

	// WebSocket for live stats
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

	// Start background collector
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

var prevNet = make(map[string]net.IOCountersStat)
var prevTime = time.Now()

func collectStats() {
	ts := time.Now().Unix()
	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()
	diskIO, _ := disk.IOCounters()
	netIO, _ := net.IOCounters(true)

	elapsed := time.Since(prevTime).Seconds()
	netRates := make(map[string]map[string]float64) // MB/s

	for _, nic := range netIO {
		// Only track eth0
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

	// Broadcast stats
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

// Returns daily/monthly network totals per interface
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
			dailyGB := data["in"]/1024/1024/1024
			monthlyGB := data["in"]/1024/1024/1024 // for simplicity, same as daily
			totals[name] = NetTotals{
				DailyIn:   dailyGB,
				DailyOut:  data["out"]/1024/1024/1024,
				MonthlyIn: monthlyGB,
				MonthlyOut: data["out"]/1024/1024/1024,
			}
			return nil
		})
		return nil
	})
	return totals
}

func runSpeedTest(conn *websocket.Conn) {
	// Placeholder: simple ping-based test
	for _, typ := range []string{"latency", "download", "upload"} {
		val := float64(0)
		if typ == "latency" {
			val = float64(10 + time.Now().UnixNano()%50) // dummy latency
		} else {
			val = float64(50 + time.Now().UnixNano()%100) // dummy MB/s
		}
		conn.WriteJSON(map[string]interface{}{
			"type":  typ,
			"value": val,
		})
		time.Sleep(1 * time.Second)
	}
}