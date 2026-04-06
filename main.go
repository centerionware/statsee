package main

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
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
	BytesSent uint64  `json:"sent"`
	BytesRecv uint64  `json:"recv"`
	RateSent  float64 `json:"rate_sent"` // bytes/sec
	RateRecv  float64 `json:"rate_recv"`
}

type DiskStat struct {
	Used  uint64 `json:"used"`
	Free  uint64 `json:"free"`
	Total uint64 `json:"total"`
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
        if r.URL.Path == "/" {
            f, _ := staticFS.Open("static/index.html")
            defer f.Close()
            io.Copy(w, f)
            return
        }
        http.FileServer(http.FS(staticFS)).ServeHTTP(w, r)
    })
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })
	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/bandwidth/daily", dailyHandler)
	http.HandleFunc("/api/bandwidth/monthly", monthlyHandler)

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// ===== WS =====
func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, _ := upgrader.Upgrade(w, r, nil)
	defer conn.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go streamStats(ctx, conn)
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil { return }
		var cmd map[string]string
		json.Unmarshal(msg, &cmd)
		if cmd["type"] == "speedtest" {
			go runSpeedTestWS(conn)
		}
	}
}

// ===== STATS =====
func streamStats(ctx context.Context, conn *websocket.Conn) {
	prevNet := make(map[string]gnet.IOCountersStat)
	prevTime := time.Now()
	for {
		msg := WSMessage{
			Type:      "stats",
			Timestamp: time.Now().Unix(),
			Net:       make(map[string]NetStat),
			Disk:      make(map[string]DiskStat),
		}

		c, _ := cpu.Percent(0, false)
		if len(c) > 0 { msg.CPU = c[0] }

		msg.Load, _ = load.Avg()
		vm, _ := mem.VirtualMemory()
		msg.RAMUsed = vm.Used
		msg.RAMFree = vm.Free

		nets, _ := gnet.IOCounters(true)
		now := time.Now()
		elapsed := now.Sub(prevTime).Seconds()
		for _, n := range nets {
			prev := prevNet[n.Name]
			sentRate := float64(n.BytesSent-prev.BytesSent) / elapsed
			recvRate := float64(n.BytesRecv-prev.BytesRecv) / elapsed
			msg.Net[n.Name] = NetStat{
				BytesSent: n.BytesSent,
				BytesRecv: n.BytesRecv,
				RateSent:  sentRate,
				RateRecv:  recvRate,
			}
			prevNet[n.Name] = n
		}
		prevTime = now

		parts, _ := disk.Partitions(false)
		for _, p := range parts {
			u, err := disk.Usage(p.Mountpoint)
			if err == nil {
				msg.Disk[p.Mountpoint] = DiskStat{Used: u.Used, Free: u.Free, Total: u.Total}
			}
		}

		conn.WriteJSON(msg)
		time.Sleep(1 * time.Second)
	}
}

// ===== SPEEDTEST =====
func runSpeedTestWS(conn *websocket.Conn) {
	url := "https://speed.cloudflare.com/__down?bytes=20000000"
	host := "speed.cloudflare.com"

	// latency
	start := time.Now()
	http.Get(url)
	conn.WriteJSON(SpeedUpdate{"latency", float64(time.Since(start).Milliseconds())})

	// download
	var total int64
	var wg sync.WaitGroup
	start = time.Now()
	done := make(chan struct{})

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, _ := http.Get(url)
			defer resp.Body.Close()
			n, _ := io.Copy(io.Discard, resp.Body)
			atomic.AddInt64(&total, n)
		}()
	}

	go func() {
		for {
			select {
			case <-done: return
			default:
				elapsed := time.Since(start).Seconds()
				if elapsed>0 {
					mb := float64(atomic.LoadInt64(&total))/elapsed/1024/1024
					conn.WriteJSON(SpeedUpdate{"download", mb})
				}
				time.Sleep(500*time.Millisecond)
			}
		}
	}()

	wg.Wait()
	close(done)

	// upload
	total = 0
	start = time.Now()
	done = make(chan struct{})
	for i:=0;i<4;i++ {
		wg.Add(1)
		go func(){
			defer wg.Done()
			data := strings.NewReader(strings.Repeat("A",5*1024*1024))
			http.Post("https://"+host,"application/octet-stream",data)
			atomic.AddInt64(&total,5*1024*1024)
		}()
	}

	go func(){
		for{
			select {
			case <-done: return
			default:
				elapsed := time.Since(start).Seconds()
				if elapsed>0{
					mb := float64(atomic.LoadInt64(&total))/elapsed/1024/1024
					conn.WriteJSON(SpeedUpdate{"upload", mb})
				}
				time.Sleep(500*time.Millisecond)
			}
		}
	}()
	wg.Wait()
	close(done)
	conn.WriteJSON(SpeedUpdate{"done",0})
}

// ===== BANDWIDTH =====
func initDB() {
	db.Update(func(tx *bolt.Tx) error {
		_, _ = tx.CreateBucketIfNotExists([]byte("bandwidth"))
		return nil
	})
}

// Track per-interface ingress/egress separately
func startBandwidthCollector() {
	prevPerIface := make(map[string]gnet.IOCountersStat)
	for {
		nets, _ := gnet.IOCounters(true)
		now := time.Now()
		for _, n := range nets {
			prev := prevPerIface[n.Name]
			inBytes := n.BytesRecv - prev.BytesRecv
			outBytes := n.BytesSent - prev.BytesSent
			if inBytes>0 { storeBandwidth(n.Name,"in",inBytes, now) }
			if outBytes>0 { storeBandwidth(n.Name,"out",outBytes, now) }
			prevPerIface[n.Name] = n
		}
		time.Sleep(1*time.Minute)
	}
}

func storeBandwidth(iface,direction string,bytes uint64,t time.Time){
	key := fmt.Sprintf("%s-%s-%s", t.Format("2006-01-02"), iface, direction)
	db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("bandwidth"))
		var prev uint64
		val := b.Get([]byte(key))
		if val!=nil { fmt.Sscanf(string(val),"%d",&prev) }
		prev += bytes
		b.Put([]byte(key),[]byte(fmt.Sprintf("%d",prev)))
		return nil
	})
}

// ===== DAILY / MONTHLY HANDLERS =====
func dailyHandler(w http.ResponseWriter, r *http.Request){
	out := map[string]map[string]float64{} // iface -> direction -> GB
	db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("bandwidth")).Cursor()
		prefix := time.Now().Format("2006-01-02")
		for k,v := c.First();k!=nil;k,v=c.Next(){
			if strings.HasPrefix(string(k),prefix){
				parts := strings.Split(string(k),"-")
				if len(parts)!=3 { continue }
				iface,direction := parts[1],parts[2]
				var val uint64
				fmt.Sscanf(string(v),"%d",&val)
				if out[iface]==nil { out[iface]=map[string]float64{} }
				out[iface][direction] += float64(val)/1024/1024/1024
			}
		}
		return nil
	})
	json.NewEncoder(w).Encode(out)
}

func monthlyHandler(w http.ResponseWriter, r *http.Request){
	out := map[string]map[string]float64{}
	db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("bandwidth")).Cursor()
		prefix := time.Now().Format("2006-01")
		for k,v := c.First();k!=nil;k,v=c.Next(){
			if strings.HasPrefix(string(k),prefix){
				parts := strings.Split(string(k),"-")
				if len(parts)!=3 { continue }
				iface,direction := parts[1],parts[2]
				var val uint64
				fmt.Sscanf(string(v),"%d",&val)
				if out[iface]==nil { out[iface]=map[string]float64{} }
				out[iface][direction] += float64(val)/1024/1024/1024
			}
		}
		return nil
	})
	json.NewEncoder(w).Encode(out)
}