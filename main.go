package main

import (
	"context"
	"embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	gnet "github.com/shirou/gopsutil/v3/net"
)

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var db *bolt.DB

// ===== TYPES =====

type WSMessage struct {
	Type      string              `json:"type"`
	CPU       float64             `json:"cpu"`
	Load      *load.AvgStat       `json:"load"`
	RAMUsed   uint64              `json:"ram_used"`
	RAMFree   uint64              `json:"ram_free"`
	Net       map[string]NetStat  `json:"net"`
	Disk      map[string]DiskStat `json:"disk"`
	Timestamp int64               `json:"ts"`
}

type NetStat struct {
	BytesSent uint64 `json:"sent"`
	BytesRecv uint64 `json:"recv"`
}

type DiskStat struct {
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
}

type SpeedServer struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Host string `json:"host"`
}

type SpeedUpdate struct {
	Type    string  `json:"type"`
	Value   float64 `json:"value"`
	Server  string  `json:"server,omitempty"`
}

// ===== MAIN =====

func main() {
	var err error
	db, err = bolt.Open("stats.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	http.Handle("/", http.FileServer(http.FS(staticFS)))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/speedtest/servers", serversHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ===== WS HANDLER =====

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go streamStats(ctx, conn)

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var cmd map[string]string
		json.Unmarshal(msg, &cmd)

		if cmd["type"] == "speedtest" {
			go runSpeedTestWS(conn)
		}
	}
}

// ===== SYSTEM STATS =====

func streamStats(ctx context.Context, conn *websocket.Conn) {
	prevNet := make(map[string]gnet.IOCountersStat)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		msg := WSMessage{
			Type:      "stats",
			Timestamp: time.Now().Unix(),
			Net:       make(map[string]NetStat),
			Disk:      make(map[string]DiskStat),
		}

		c, _ := cpu.Percent(0, false)
		if len(c) > 0 {
			msg.CPU = c[0]
		}

		msg.Load, _ = load.Avg()

		vm, _ := mem.VirtualMemory()
		msg.RAMUsed = vm.Used
		msg.RAMFree = vm.Free

		nets, _ := gnet.IOCounters(true)
		for _, n := range nets {
			prev := prevNet[n.Name]
			msg.Net[n.Name] = NetStat{
				BytesSent: n.BytesSent - prev.BytesSent,
				BytesRecv: n.BytesRecv - prev.BytesRecv,
			}
			prevNet[n.Name] = n
		}

		parts, _ := disk.Partitions(false)
		for _, p := range parts {
			u, err := disk.Usage(p.Mountpoint)
			if err == nil {
				msg.Disk[p.Mountpoint] = DiskStat{
					Used:  u.Used,
					Free:  u.Free,
					Total: u.Total,
				}
			}
		}

		conn.WriteJSON(msg)
		time.Sleep(1 * time.Second)
	}
}

// ===== SPEEDTEST =====

func serversHandler(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(discoverServers())
}

func discoverServers() []SpeedServer {
	return []SpeedServer{
		{"Cloudflare", "https://speed.cloudflare.com/__down?bytes=20000000", "speed.cloudflare.com"},
		{"Google", "https://storage.googleapis.com/gcp-public-data-landsat/index.csv.gz", "google.com"},
	}
}

func runSpeedTestWS(conn *websocket.Conn) {
	server := discoverServers()[0]

	// LATENCY
	start := time.Now()
	http.Get(server.URL)
	lat := float64(time.Since(start).Milliseconds())

	conn.WriteJSON(SpeedUpdate{
		Type:   "latency",
		Value:  lat,
		Server: server.Name,
	})

	// DOWNLOAD
	var totalBytes int64
	var wg sync.WaitGroup
	workers := 4
	start = time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(server.URL)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			buf := make([]byte, 32*1024)
			for {
				n, err := resp.Body.Read(buf)
				if n > 0 {
					atomic.AddInt64(&totalBytes, int64(n))
				}
				if err != nil {
					break
				}
			}
		}()
	}

	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				elapsed := time.Since(start).Seconds()
				if elapsed == 0 {
					continue
				}
				mbps := float64(atomic.LoadInt64(&totalBytes)) / elapsed / 1024 / 1024
				conn.WriteJSON(SpeedUpdate{Type: "download", Value: mbps})
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(done)

	// UPLOAD
	totalBytes = 0
	start = time.Now()
	done = make(chan struct{})

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			data := strings.NewReader(strings.Repeat("A", 5*1024*1024))
			http.Post("https://"+server.Host, "application/octet-stream", data)
			atomic.AddInt64(&totalBytes, 5*1024*1024)
		}()
	}

	go func() {
		for {
			select {
			case <-done:
				return
			default:
				elapsed := time.Since(start).Seconds()
				if elapsed == 0 {
					continue
				}
				mbps := float64(atomic.LoadInt64(&totalBytes)) / elapsed / 1024 / 1024
				conn.WriteJSON(SpeedUpdate{Type: "upload", Value: mbps})
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(done)

	conn.WriteJSON(SpeedUpdate{Type: "done"})
}