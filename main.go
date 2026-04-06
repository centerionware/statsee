package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

//go:embed static/*
var staticFS embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var db *bolt.DB

type WSMessage struct {
	CPU        float64            `json:"cpu"`
	Load       *load.AvgStat      `json:"load"`
	RAMUsed    uint64             `json:"ram_used"`
	RAMFree    uint64             `json:"ram_free"`
	Net        map[string]NetStat `json:"net"`
	Disk       map[string]DiskStat`json:"disk"`
	Timestamp  int64              `json:"ts"`
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

type SpeedResult struct {
	Server   string  `json:"server"`
	Latency  float64 `json:"latency"`
	Download float64 `json:"download"`
	Upload   float64 `json:"upload"`
}

func main() {
	var err error
	db, err = bolt.Open("stats.db", 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	http.Handle("/", http.FileServer(http.FS(staticFS)))

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/speedtest/servers", serversHandler)
	http.HandleFunc("/api/speedtest/start", speedTestHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()

	prevNet := make(map[string]net.IOCountersStat)

	for {
		msg := WSMessage{
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

		nets, _ := net.IOCounters(true)
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

func serversHandler(w http.ResponseWriter, r *http.Request) {
	servers := discoverServers()
	json.NewEncoder(w).Encode(servers)
}

func discoverServers() []SpeedServer {
	// Lightweight public endpoints (auto discovery placeholder)
	return []SpeedServer{
		{"Cloudflare", "https://speed.cloudflare.com/__down?bytes=10000000", "speed.cloudflare.com"},
		{"Google", "https://storage.googleapis.com/gcp-public-data-landsat/index.csv.gz", "google.com"},
	}
}

func speedTestHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Servers []SpeedServer `json:"servers"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	results := []SpeedResult{}
	for _, s := range req.Servers {
		res := runSpeedTest(s)
		results = append(results, res)
	}

	json.NewEncoder(w).Encode(results)
}

func runSpeedTest(s SpeedServer) SpeedResult {
	lat := measureLatency(s.URL)
	down := measureDownload(s.URL)
	up := measureUpload("https://" + s.Host)

	return SpeedResult{
		Server:   s.Name,
		Latency:  lat,
		Download: down,
		Upload:   up,
	}
}

func measureLatency(url string) float64 {
	start := time.Now()
	http.Get(url)
	return float64(time.Since(start).Milliseconds())
}

func measureDownload(url string) float64 {
	start := time.Now()
	resp, err := http.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	n, _ := io.Copy(io.Discard, resp.Body)
	sec := time.Since(start).Seconds()

	return float64(n) / sec / 1024 / 1024
}

func measureUpload(host string) float64 {
	data := strings.NewReader(strings.Repeat("A", 5*1024*1024))
	start := time.Now()
	http.Post(host, "application/octet-stream", data)
	sec := time.Since(start).Seconds()

	return float64(5) / sec
}