package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
    "io/fs"

	bolt "go.etcd.io/bbolt"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

//go:embed static/*
var staticFS embed.FS

var db *bolt.DB
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type WSMessage struct {
	Type      string              `json:"type"`
	Timestamp int64               `json:"ts"`
	CPU       float64             `json:"cpu"`
	RAM       RAMStat             `json:"ram"`
	Disk      map[string]DiskStat `json:"disk"`
	Net       map[string]NetStat  `json:"net"`
}

type RAMStat struct {
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
}

type DiskStat struct {
	ReadIO  uint64 `json:"read_io"`
	WriteIO uint64 `json:"write_io"`
}

type NetStat struct {
	BytesRecv   uint64  `json:"bytes_recv"`
	BytesSent   uint64  `json:"bytes_sent"`
	PacketsRecv uint64  `json:"packets_recv"`
	PacketsSent uint64  `json:"packets_sent"`
	RateRecv    float64 `json:"rate_recv"` // MB/s
	RateSent    float64 `json:"rate_sent"` // MB/s
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

    // Wrap the embedded FS to remove the "static/" prefix
    subFS, err := fs.Sub(staticFS, "static")
    if err != nil {
    	log.Fatal(err)
    }
    
    // Serve index.html at /
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
    	data, err := subFS.ReadFile("index.html") // now we can just read "index.html"
    	if err != nil {
    		http.Error(w, "index.html not found", 500)
    		return
    	}
    	w.Header().Set("Content-Type", "text/html")
    	w.Write(data)
    })
    
    // Serve static files at /static/
    http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(subFS))))
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/bandwidth/daily", dailyHandler)
	http.HandleFunc("/api/bandwidth/monthly", monthlyHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// =================== WebSocket ===================
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
			go runSpeedTestWS(conn)
		}
	}
}

func streamStats(conn *websocket.Conn, done chan struct{}) {
	prevNet := make(map[string]NetStat)
	prevDisk := make(map[string]DiskStat)

	// initialize previous values to avoid 0
	netIO, _ := net.IOCounters(true)
	for _, n := range netIO {
		prevNet[n.Name] = NetStat{BytesRecv: n.BytesRecv, BytesSent: n.BytesSent}
	}
	diskIO, _ := disk.IOCounters()
	for dev, d := range diskIO {
		prevDisk[dev] = DiskStat{ReadIO: d.ReadBytes, WriteIO: d.WriteBytes}
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			cpuPerc, _ := cpu.Percent(0, false)
			memStat, _ := mem.VirtualMemory()
			netIO, _ := net.IOCounters(true)
			diskIO, _ := disk.IOCounters()

			netStats := make(map[string]NetStat)
			for _, n := range netIO {
				prev := prevNet[n.Name]
				rateRecv := float64(n.BytesRecv-prev.BytesRecv) / 1024 / 1024
				rateSent := float64(n.BytesSent-prev.BytesSent) / 1024 / 1024
				netStats[n.Name] = NetStat{
					BytesRecv:   n.BytesRecv,
					BytesSent:   n.BytesSent,
					PacketsRecv: n.PacketsRecv,
					PacketsSent: n.PacketsSent,
					RateRecv:    rateRecv,
					RateSent:    rateSent,
				}
				prevNet[n.Name] = NetStat{BytesRecv: n.BytesRecv, BytesSent: n.BytesSent}
			}

			diskStats := make(map[string]DiskStat)
			for dev, d := range diskIO {
				prev := prevDisk[dev]
				diskStats[dev] = DiskStat{
					ReadIO:  d.ReadBytes - prev.ReadIO,
					WriteIO: d.WriteBytes - prev.WriteIO,
				}
				prevDisk[dev] = DiskStat{ReadIO: d.ReadBytes, WriteIO: d.WriteBytes}
			}

			msg := WSMessage{
				Type:      "stats",
				Timestamp: time.Now().Unix(),
				CPU:       cpuPerc[0],
				RAM: RAMStat{
					Used: memStat.Used / (1024 * 1024),
					Free: memStat.Available / (1024 * 1024),
				},
				Disk: diskStats,
				Net:  netStats,
			}
			conn.WriteJSON(msg)
		}
	}
}

// =================== Speed Test ===================
func runSpeedTestWS(conn *websocket.Conn) {
	if conn == nil {
		return
	}

	start := time.Now()
	resp, _ := http.Get("https://speed.cloudflare.com/__down?bytes=20000000")
	if resp != nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	latency := float64(time.Since(start).Milliseconds())
	conn.WriteJSON(SpeedUpdate{"latency", latency})

	payload := strings.Repeat("0", 1024*1024*5)
	start = time.Now()
	http.Post("https://speed.cloudflare.com/__up", "text/plain", strings.NewReader(payload))
	duration := float64(time.Since(start).Milliseconds())
	conn.WriteJSON(SpeedUpdate{"upload", 5 * 1024 / duration})   // MB/s
	conn.WriteJSON(SpeedUpdate{"download", 20 * 1024 / duration}) // MB/s
}

// =================== BoltDB Bandwidth ===================
func initDB() {
	db.Update(func(tx *bolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("bandwidth"))
		return nil
	})
}

func startBandwidthCollector() {
	prevNet := make(map[string]NetStat)
	netIO, _ := net.IOCounters(true)
	for _, n := range netIO {
		prevNet[n.Name] = NetStat{BytesRecv: n.BytesRecv, BytesSent: n.BytesSent}
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C
		netIO, _ := net.IOCounters(true)
		now := time.Now()
		for _, n := range netIO {
			prev := prevNet[n.Name]
			inBytes := n.BytesRecv - prev.BytesRecv
			outBytes := n.BytesSent - prev.BytesSent
			storeBandwidth(n.Name, "in", inBytes, now)
			storeBandwidth(n.Name, "out", outBytes, now)
			prevNet[n.Name] = NetStat{BytesRecv: n.BytesRecv, BytesSent: n.BytesSent}
		}
	}
}

func storeBandwidth(iface, direction string, bytes uint64, t time.Time) {
	key := fmt.Sprintf("%s-%s-%s", t.Format("2006-01-02"), iface, direction)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("bandwidth"))
		prev := b.Get([]byte(key))
		var prevVal uint64
		fmt.Sscanf(string(prev), "%d", &prevVal)
		prevVal += bytes
		b.Put([]byte(key), []byte(fmt.Sprintf("%d", prevVal)))
		return nil
	})
}

// =================== API ===================
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