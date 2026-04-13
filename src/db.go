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

var Iface = getInterface()

type NetSnapshot struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
	Ts int64  `json:"ts"`
}

type Accumulator struct {
	In  float64 `json:"in"`
	Out float64 `json:"out"`
}

const bucket = "net"

// --------------------
// INIT
// --------------------

func InitDB() {
	_ = os.MkdirAll("/db", 0755)

	var err error
	db, err = bolt.Open(dbPath, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("DB initialized at", dbPath)
}

func CloseDB() {
	_ = db.Close()
}

// --------------------
// CORE STATE KEYS
// --------------------

func keyDaily(t time.Time) string   { return "acc:daily:" + t.Format("2006-01-02") }
func keyMonthly(t time.Time) string { return "acc:monthly:" + t.Format("2006-01") }

func keyDailySnap(t time.Time) string   { return "snap:daily:" + t.Format("2006-01-02") }
func keyMonthlySnap(t time.Time) string { return "snap:monthly:" + t.Format("2006-01") }

// --------------------
// UPDATE (FIXED ENGINE)
// --------------------

func UpdateNetworkTotals(rx, tx uint64) {
	now := time.Now().UTC()

	err := db.Update(func(txn *bolt.Tx) error {
		b, err := txn.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		// ---------------------------
		// load previous snapshot
		// ---------------------------
		var prev NetSnapshot
		if v := b.Get([]byte("meta:last")); v != nil {
			_ = json.Unmarshal(v, &prev)
		}

		current := NetSnapshot{Rx: rx, Tx: tx, Ts: now.Unix()}

		// ---------------------------
		// reboot detection
		// ---------------------------
		reboot := rx < prev.Rx || tx < prev.Tx

		var deltaRx, deltaTx uint64

		if prev.Rx == 0 && prev.Tx == 0 {
			// first run
			deltaRx, deltaTx = 0, 0
		} else if reboot {
			// VM reboot → reset baseline
			deltaRx, deltaTx = 0, 0
			log.Println("Reboot detected - resetting baseline")
		} else {
			deltaRx = rx - prev.Rx
			deltaTx = tx - prev.Tx
		}

		// store last snapshot
		data, _ := json.Marshal(current)
		_ = b.Put([]byte("meta:last"), data)

		// ---------------------------
		// accumulate daily
		// ---------------------------
		dk := keyDaily(now)
		var d Accumulator
		if v := b.Get([]byte(dk)); v != nil {
			_ = json.Unmarshal(v, &d)
		}

		d.In += float64(deltaRx) / 1024 / 1024 / 1024
		d.Out += float64(deltaTx) / 1024 / 1024 / 1024

		dd, _ := json.Marshal(d)
		_ = b.Put([]byte(dk), dd)

		// ---------------------------
		// accumulate monthly
		// ---------------------------
		mk := keyMonthly(now)
		var m Accumulator
		if v := b.Get([]byte(mk)); v != nil {
			_ = json.Unmarshal(v, &m)
		}

		m.In += float64(deltaRx) / 1024 / 1024 / 1024
		m.Out += float64(deltaTx) / 1024 / 1024 / 1024

		md, _ := json.Marshal(m)
		_ = b.Put([]byte(mk), md)

		return nil
	})

	if err != nil {
		log.Println("DB update error:", err)
	}
}

// --------------------
// LIVE API (NOW SIMPLE + CORRECT)
// --------------------

func HandleNetworkLive(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	var daily, monthly Accumulator

	_ = db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		_ = json.Unmarshal(b.Get([]byte(keyDaily(now))), &daily)
		_ = json.Unmarshal(b.Get([]byte(keyMonthly(now))), &monthly)

		return nil
	})

	resp := map[string]NetTotals{
		Iface: {
			DailyIn:    daily.In,
			DailyOut:   daily.Out,
			MonthlyIn:  monthly.In,
			MonthlyOut: monthly.Out,
		},
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// --------------------
// HISTORY API (FIXED)
// --------------------

func HandleNetworkHistory(w http.ResponseWriter, r *http.Request) {
	var daily []DailyStat
	var monthly []MonthlyStat

	_ = db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)

			var a Accumulator
			_ = json.Unmarshal(v, &a)

			if strings.HasPrefix(key, "acc:daily:") {
				daily = append(daily, DailyStat{
					Date: strings.TrimPrefix(key, "acc:daily:"),
					In:   a.In,
					Out:  a.Out,
				})
			}

			if strings.HasPrefix(key, "acc:monthly:") {
				monthly = append(monthly, MonthlyStat{
					Month: strings.TrimPrefix(key, "acc:monthly:"),
					In:    a.In,
					Out:   a.Out,
				})
			}
		}

		return nil
	})

	sort.Slice(daily, func(i, j int) bool { return daily[i].Date < daily[j].Date })
	sort.Slice(monthly, func(i, j int) bool { return monthly[i].Month < monthly[j].Month })

	_ = json.NewEncoder(w).Encode(map[string]any{
		"daily":   daily,
		"monthly": monthly,
	})
}

// --------------------
// COMPAT
// --------------------

func HandleNetworkTotals(w http.ResponseWriter, r *http.Request) {
	HandleNetworkLive(w, r)
}