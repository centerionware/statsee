package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/gorilla/websocket"
)

//go:embed static/*
var staticFS embed.FS

var db *bolt.DB
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
	Type      string             `json:"type"`
	Timestamp int64              `json:"ts"`
	Net       map[string]NetStat `json:"net"`
	Disk      map[string]DiskStat `json:"disk"`
}

type NetStat struct {
	BytesRecv   uint64  `json:"bytes_recv"`
	BytesSent   uint64  `json:"bytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
	RateRecv    float64 `json:"rate_recv"`
	RateSent    float64 `json:"rate_sent"`
}

type DiskStat struct {
	ReadIO  uint64 `json:"read_io"`
	WriteIO uint64 `json:"write_io"`
}

type SpeedUpdate struct {
	Type  string  `json:"type"`
	Value float64 `json:"value"`
}

func main() {
	var err error
	db, err = bolt.Open("stats.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDB()
	go startBandwidthCollector()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		f, err := staticFS.Open("static/index.html")
		if err != nil {
			w.WriteHeader(404)
			return
		}
		defer f.Close()
		io.Copy(w, f)
	})
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/bandwidth/daily", dailyHandler)
	http.HandleFunc("/api/bandwidth/monthly", monthlyHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ===== SAFE WS WRITE =====
func safeWSWrite(conn *websocket.Conn, msg interface{}) bool {
	if conn == nil {
		return false
	}
	if err := conn.WriteJSON(msg); err != nil {
		return false
	}
	return true
}

// ===== WS HANDLER =====
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	done := make(chan struct{})
	go streamStats(conn, done)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			close(done)
			return
		}
		var cmd map[string]string
		json.Unmarshal(msg, &cmd)
		if cmd["type"] == "speedtest" {
			go runSpeedTestWS(conn, done)
		}
	}
}

// ===== STREAM STATS =====
func streamStats(conn *websocket.Conn, done chan struct{}) {
	prevNet := make(map[string]NetStat)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			netStats, _ := readNetworkStats()
			diskStats, _ := readDiskStats()

			now := time.Now()
			for iface, curr := range netStats {
				prev := prevNet[iface]
				curr.RateRecv = float64(curr.BytesRecv-prev.BytesRecv) / 1024 / 1024
				curr.RateSent = float64(curr.BytesSent-prev.BytesSent) / 1024 / 1024
				prevNet[iface] = curr
				netStats[iface] = curr
			}

			msg := WSMessage{
				Type:      "stats",
				Timestamp: now.Unix(),
				Net:       netStats,
				Disk:      diskStats,
			}
			safeWSWrite(conn, msg)
		}
	}
}

// ===== SPEEDTEST =====
func runSpeedTestWS(conn *websocket.Conn, done chan struct{}) {
	if conn == nil {
		return
	}

	// Download test
	downloadURL := "https://speed.cloudflare.com/__down?bytes=20000000"
	start := time.Now()
	resp, _ := http.Get(downloadURL)
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	latency := float64(time.Since(start).Milliseconds())
	safeWSWrite(conn, SpeedUpdate{"latency", latency})

	// Upload test (dummy POST to Cloudflare echo)
	var total int64
	var wg sync.WaitGroup
	start = time.Now()
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			payload := strings.Repeat("0", 1024*1024*5)
			http.Post("https://speed.cloudflare.com/__up", "text/plain", strings.NewReader(payload))
			atomic.AddInt64(&total, 5)
		}()
	}
	wg.Wait()
	safeWSWrite(conn, SpeedUpdate{"download", float64(total)})
	safeWSWrite(conn, SpeedUpdate{"upload", float64(total)})
}

// ===== NETWORK & DISK =====
func readNetworkStats() (map[string]NetStat, error) {
	stats := make(map[string]NetStat)
	data, err := os.ReadFile("/proc/net/dev")
	if err != nil {
		return stats, err
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines[2:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		iface := strings.TrimSuffix(parts[0], ":")
		recvBytes, _ := strconv.ParseUint(parts[1], 10, 64)
		recvPackets, _ := strconv.ParseUint(parts[2], 10, 64)
		sentBytes, _ := strconv.ParseUint(parts[9], 10, 64)
		sentPackets, _ := strconv.ParseUint(parts[10], 10, 64)
		stats[iface] = NetStat{
			BytesRecv:   recvBytes,
			BytesSent:   sentBytes,
			PacketsRecv: recvPackets,
			PacketsSent: sentPackets,
		}
	}
	return stats, nil
}

func readDiskStats() (map[string]DiskStat, error) {
	stats := make(map[string]DiskStat)
	data, err := os.ReadFile("/proc/diskstats")
	if err != nil {
		return stats, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 14 {
			continue
		}
		dev := parts[2]
		reads, _ := strconv.ParseUint(parts[5], 10, 64)
		writes, _ := strconv.ParseUint(parts[9], 10, 64)
		stats[dev] = DiskStat{ReadIO: reads, WriteIO: writes}
	}
	return stats, nil
}

// ===== BOLTDB BANDWIDTH =====
func initDB() {
	db.Update(func(tx *bolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("bandwidth"))
		return nil
	})
}

func startBandwidthCollector() {
	prevNet := make(map[string]NetStat)
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		curr, _ := readNetworkStats()
		now := time.Now()
		for iface, val := range curr {
			prev := prevNet[iface]
			inBytes := val.BytesRecv - prev.BytesRecv
			outBytes := val.BytesSent - prev.BytesSent
			storeBandwidth(iface, "in", inBytes, now)
			storeBandwidth(iface, "out", outBytes, now)
			prevNet[iface] = val
		}
	}
}

func storeBandwidth(iface, direction string, bytes uint64, t time.Time) {
	key := fmt.Sprintf("%s-%s-%s", t.Format("2006-01-02"), iface, direction)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("bandwidth"))
		var prev uint64
		val := b.Get([]byte(key))
		if val != nil {
			fmt.Sscanf(string(val), "%d", &prev)
		}
		prev += bytes
		b.Put([]byte(key), []byte(fmt.Sprintf("%d", prev)))
		return nil
	})
}

// ===== DAILY / MONTHLY API =====
func dailyHandler(w http.ResponseWriter, r *http.Request)   { outputUsage(w, "2006-01-02") }
func monthlyHandler(w http.ResponseWriter, r *http.Request) { outputUsage(w, "2006-01") }

func outputUsage(w http.ResponseWriter, format string) {
	out := map[string]map[string]float64{}
	prefix := time.Now().Format(format)
	db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("bandwidth")).Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if strings.HasPrefix(string(k), prefix) {
				parts := strings.Split(string(k), "-")
				if len(parts) != 3 {
					continue
				}
				iface, direction := parts[1], parts[2]
				var val uint64
				fmt.Sscanf(string(v), "%d", &val)
				if out[iface] == nil {
					out[iface] = map[string]float64{}
				}
				out[iface][direction] += float64(val) / 1024 / 1024 / 1024
			}
		}
		return nil
	})
	json.NewEncoder(w).Encode(out)
}