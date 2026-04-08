package src

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

var db *bolt.DB

const (
	dbPath = "/db/statsee.db"
)

// Stored structures
type NetSnapshot struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
	Ts int64  `json:"ts"`
}

var lastPersist time.Time

// --------------------
// INIT / CLOSE
// --------------------

func InitDB() {
	// Ensure directory exists (important for Kubernetes PVC mount)
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
// WRITE LOGIC
// --------------------

func UpdateNetworkTotals(rx, tx uint64) {
	now := time.Now()

	// Only persist every 5 seconds
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

		// Store current state (for crash recovery)
		if err := b.Put([]byte("meta:current"), data); err != nil {
			return err
		}

		// Check if we need to create a daily snapshot
		dayKey := "daily:" + now.Format("2006-01-02")

		if b.Get([]byte(dayKey)) == nil {
			// First write of the day → snapshot
			if err := b.Put([]byte(dayKey), data); err != nil {
				return err
			}
			log.Println("Created daily snapshot:", dayKey)
		}

		return nil
	})

	if err != nil {
		log.Println("DB write error:", err)
	}
}

//
// --------------------
// READ / API
// --------------------

func HandleNetworkTotals(w http.ResponseWriter, r *http.Request) {
	result := make(map[string]NetTotals)

	err := db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte("net"))
		if b == nil {
			return nil
		}

		now := time.Now()
		todayKey := "daily:" + now.Format("2006-01-02")
		yesterdayKey := "daily:" + now.Add(-24*time.Hour).Format("2006-01-02")

		var todaySnap, yesterdaySnap, currentSnap NetSnapshot

		// Load snapshots
		if v := b.Get([]byte(todayKey)); v != nil {
			json.Unmarshal(v, &todaySnap)
		}
		if v := b.Get([]byte(yesterdayKey)); v != nil {
			json.Unmarshal(v, &yesterdaySnap)
		}
		if v := b.Get([]byte("meta:current")); v != nil {
			json.Unmarshal(v, &currentSnap)
		}

		// Calculate daily usage
		dailyIn := float64(currentSnap.Rx-todaySnap.Rx) / 1024 / 1024 / 1024
		dailyOut := float64(currentSnap.Tx-todaySnap.Tx) / 1024 / 1024 / 1024

		// Calculate yesterday usage (for sanity / future UI)
		_ = yesterdaySnap // (you’ll likely use this soon)

		result[Iface] = NetTotals{
			DailyIn:    dailyIn,
			DailyOut:   dailyOut,
			MonthlyIn:  dailyIn,  // placeholder for now
			MonthlyOut: dailyOut, // placeholder
		}

		return nil
	})

	if err != nil {
		log.Println("DB read error:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}