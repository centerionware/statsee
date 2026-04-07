package main

import (
	// "bufio"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	bolt "go.etcd.io/bbolt"
)

//go:embed static/*
var staticFS embed.FS

var (
	db        *bolt.DB
	wsClients = make(map[*websocket.Conn]bool)
	wsLock    sync.Mutex
	upgrader  = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

	prevRx   uint64
	prevTx   uint64
	prevTime time.Time

	iface = getInterface()
)

type NetTotals struct {
	DailyIn    float64 `json:"daily_in"`
	DailyOut   float64 `json:"daily_out"`
	MonthlyIn  float64 `json:"monthly_in"`
	MonthlyOut float64 `json:"monthly_out"`
}

func getInterface() string {
	if v := os.Getenv("NET_IFACE"); v != "" {
		return v
	}
	return "eth0"
}

func main() {
	var err error
	db, err = bolt.Open("statsee.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Using interface:", iface)
	debugNet()

	subFS, err := fs.Sub(staticFS, "static")
	if err != nil {
		log.Fatal(err)
	}

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
		json.NewEncoder(w).Encode(getNetworkTotals())
	})

	http.HandleFunc("/ws", wsHandler)

	go startCollector()

	log.Println("StatSee running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func debugNet() {
	data, err := os.ReadFile("/host/proc/net/dev")
	if err != nil {
		log.Println("FAILED to read /host/proc/net/dev:", err)
		return
	}
	log.Println("==== /host/proc/net/dev ====")
	log.Println(string(data))
	log.Println("================================")
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

func readHostNetDev() (uint64, uint64, error) {
	base := "/host/sys/class/net/" + iface + "/statistics"

	rxBytes, err := os.ReadFile(base + "/rx_bytes")
	if err != nil {
		return 0, 0, err
	}

	txBytes, err := os.ReadFile(base + "/tx_bytes")
	if err != nil {
		return 0, 0, err
	}

	rx, _ := strconv.ParseUint(strings.TrimSpace(string(rxBytes)), 10, 64)
	tx, _ := strconv.ParseUint(strings.TrimSpace(string(txBytes)), 10, 64)

	return rx, tx, nil
}

func collectStats() {
	ts := time.Now().Unix()

	cpuPercent, _ := cpu.Percent(0, false)
	memStat, _ := mem.VirtualMemory()
	diskIO, _ := disk.IOCounters()

	rx, tx, err := readHostNetDev()

	var rxRate, txRate float64
	now := time.Now()

	if err != nil {
		log.Println("net read error:", err)
	} else {
		if prevTime.IsZero() {
			prevRx = rx
			prevTx = tx
			prevTime = now
			return
		}

		elapsed := now.Sub(prevTime).Seconds()
		if elapsed <= 0 {
			return
		}

		if rx < prevRx || tx < prevTx {
			prevRx = rx
			prevTx = tx
			prevTime = now
			return
		}

		rxDelta := float64(rx - prevRx)
		txDelta := float64(tx - prevTx)

		rxRate = rxDelta / elapsed / 1024 / 1024
		txRate = txDelta / elapsed / 1024 / 1024

		prevRx = rx
		prevTx = tx
		prevTime = now

		db.Update(func(txn *bolt.Tx) error {
			b, _ := txn.CreateBucketIfNotExists([]byte("net"))
			v := map[string]float64{
				"in":  float64(rx) / 1024 / 1024 / 1024,
				"out": float64(tx) / 1024 / 1024 / 1024,
			}
			data, _ := json.Marshal(v)
			b.Put([]byte(iface), data)
			return nil
		})
	}

	msg := map[string]interface{}{
		"type": "stats",
		"ts":   ts,
		"cpu":  cpuPercent[0],
		"ram": map[string]float64{
			"used": float64(memStat.Used) / 1024 / 1024,
			"free": float64(memStat.Available) / 1024 / 1024,
		},
		"disk": diskIO,
		"net": map[string]interface{}{
			iface: map[string]float64{
				"rate_recv": rxRate,
				"rate_sent": txRate,
			},
		},
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

		d := 50 + rand.Float64()*200
		u := 20 + rand.Float64()*100

		downloads = append(downloads, d)
		uploads = append(uploads, u)

		conn.WriteJSON(map[string]interface{}{
			"type":     "speedtest_update",
			"download": d,
			"upload":   u,
		})
	}

	conn.WriteJSON(map[string]interface{}{
		"type":     "speedtest_done",
		"download": average(downloads),
		"upload":   average(uploads),
	})
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