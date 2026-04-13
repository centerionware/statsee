package src

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB

const dbPath = "/db/statsee.db"

type NetSnapshot struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
	Ts int64  `json:"ts"`
}

var lastPersist time.Time

// --------------------
// INIT
// --------------------

func InitDB() {
	if err := os.MkdirAll("/db", 0755); err != nil {
		log.Fatal(err)
	}

	var err error
	db, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("DB initialized at", dbPath)
}

func CloseDB() {
	db.Close()
}

// --------------------
// WRITE LOGIC (UTC + FIXED RETENTION)
// --------------------

func UpdateNetworkTotals(rx, tx uint64) {
	now := time.Now().UTC()

	if time.Since(lastPersist) < 5*time.Second {
		return
	}
	lastPersist = now

	err := db.Update(func(txn *bolt.Tx) error {
		b, err := txn.CreateBucketIfNotExists([]byte("net"))
		if err != nil {
			return err
		}

		current := NetSnapshot{
			Rx: rx,
			Tx: tx,
			Ts: now.Unix(),
		}

		data, _ := json.Marshal(current)

		prev := b.Get([]byte("meta:current"))

		// Store current
		if err := b.Put([]byte("meta:current"), data); err != nil {
			return err
		}

		dayKey := "daily:" + now.Format("2006-01-02")
		monthKey := "monthly:" + now.Format("2006-01")

		// Snapshot previous value at boundary
		if b.Get([]byte(dayKey)) == nil && prev != nil {
			b.Put([]byte(dayKey), prev)
			log.Println("Created daily snapshot:", dayKey)
		}

		if b.Get([]byte(monthKey)) == nil && prev != nil {
			b.Put([]byte(monthKey), prev)
			log.Println("Created monthly snapshot:", monthKey)
		}

		// ✅ RETENTION: keep ONLY current month daily data
		currentMonth := now.Format("2006-01")

		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			key := string(k)

			if strings.HasPrefix(key, "daily:") {
				dateStr := strings.TrimPrefix(key, "daily:")
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					continue
				}

				if t.Format("2006-01") != currentMonth {
					b.Delete(k)
				}
			}
		}

		return nil
	})

	if err != nil {
		log.Println("DB write error:", err)
	}
}

//
// --------------------
// LIVE API
// --------------------

func HandleNetworkLive(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]NetTotals)

	db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte("net"))
		if b == nil {
			return nil
		}

		now := time.Now().UTC()

		todayKey := "daily:" + now.Format("2006-01-02")
		monthKey := "monthly:" + now.Format("2006-01")

		var todaySnap, monthSnap, current NetSnapshot

		if v := b.Get([]byte(todayKey)); v != nil {
			json.Unmarshal(v, &todaySnap)
		}
		if v := b.Get([]byte(monthKey)); v != nil {
			json.Unmarshal(v, &monthSnap)
		}
		if v := b.Get([]byte("meta:current")); v != nil {
			json.Unmarshal(v, &current)
		}

		result[Iface] = NetTotals{
			DailyIn:    safeDelta(current.Rx, todaySnap.Rx),
			DailyOut:   safeDelta(current.Tx, todaySnap.Tx),
			MonthlyIn:  safeDelta(current.Rx, monthSnap.Rx),
			MonthlyOut: safeDelta(current.Tx, monthSnap.Tx),
		}

		return nil
	})

	json.NewEncoder(w).Encode(result)
}

//
// --------------------
// HISTORY API
// --------------------

func HandleNetworkHistory(w http.ResponseWriter, r *http.Request) {
	var daily []DailyStat
	var monthly []MonthlyStat

	db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte("net"))
		if b == nil {
			return nil
		}

		type kv struct {
			key string
			val NetSnapshot
		}

		var dailySnaps []kv
		var monthlySnaps []kv

		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)

			var snap NetSnapshot
			json.Unmarshal(v, &snap)

			if strings.HasPrefix(key, "daily:") {
				dailySnaps = append(dailySnaps, kv{key, snap})
			}
			if strings.HasPrefix(key, "monthly:") {
				monthlySnaps = append(monthlySnaps, kv{key, snap})
			}
		}

		sort.Slice(dailySnaps, func(i, j int) bool {
			return dailySnaps[i].key < dailySnaps[j].key
		})

		sort.Slice(monthlySnaps, func(i, j int) bool {
			return monthlySnaps[i].key < monthlySnaps[j].key
		})

		// Daily deltas
		for i := 1; i < len(dailySnaps); i++ {
			prev := dailySnaps[i-1]
			curr := dailySnaps[i]

			daily = append(daily, DailyStat{
				Date: strings.TrimPrefix(curr.key, "daily:"),
				In:   safeDelta(curr.val.Rx, prev.val.Rx),
				Out:  safeDelta(curr.val.Tx, prev.val.Tx),
			})
		}

		// Monthly deltas
		for i := 1; i < len(monthlySnaps); i++ {
			prev := monthlySnaps[i-1]
			curr := monthlySnaps[i]

			monthly = append(monthly, MonthlyStat{
				Month: strings.TrimPrefix(curr.key, "monthly:"),
				In:    safeDelta(curr.val.Rx, prev.val.Rx),
				Out:   safeDelta(curr.val.Tx, prev.val.Tx),
			})
		}

		// ✅ Include current month live
		now := time.Now().UTC()
		monthKey := "monthly:" + now.Format("2006-01")

		var monthSnap, current NetSnapshot

		if v := b.Get([]byte(monthKey)); v != nil {
			json.Unmarshal(v, &monthSnap)
		}
		if v := b.Get([]byte("meta:current")); v != nil {
			json.Unmarshal(v, &current)
		}

		monthly = append(monthly, MonthlyStat{
			Month: now.Format("2006-01"),
			In:    safeDelta(current.Rx, monthSnap.Rx),
			Out:   safeDelta(current.Tx, monthSnap.Tx),
		})

		return nil
	})

	json.NewEncoder(w).Encode(map[string]interface{}{
		"daily":   daily,
		"monthly": monthly,
	})
}

//
// --------------------
// COMPAT
// --------------------

func HandleNetworkTotals(w http.ResponseWriter, r *http.Request) {
	HandleNetworkLive(w, r)
}

//
// --------------------
// HELPERS
// --------------------

func safeDelta(a, b uint64) float64 {
	if a < b {
		return 0
	}
	return float64(a-b) / 1024 / 1024 / 1024
}