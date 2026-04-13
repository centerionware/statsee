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

const bucket = "net"

// --------------------
// OLD + NEW TYPES
// --------------------

type NetSnapshot struct {
	Rx uint64 `json:"rx"`
	Tx uint64 `json:"tx"`
	Ts int64  `json:"ts"`
}

type Accumulator struct {
	In  float64 `json:"in"`
	Out float64 `json:"out"`
}

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

	runMigration()
}

func CloseDB() {
	_ = db.Close()
}

// --------------------
// KEY HELPERS
// --------------------

func oldDailyKey(t time.Time) string   { return "daily:" + t.Format("2006-01-02") }
func oldMonthlyKey(t time.Time) string { return "monthly:" + t.Format("2006-01") }

func newDailyKey(t time.Time) string   { return "acc:daily:" + t.Format("2006-01-02") }
func newMonthlyKey(t time.Time) string { return "acc:monthly:" + t.Format("2006-01") }

func metaMigratedKey() []byte { return []byte("meta:migrated") }
func metaLastKey() []byte     { return []byte("meta:last") }

// --------------------
// MIGRATION (RUN ONCE)
// --------------------

func runMigration() {
	_ = db.Update(func(txn *bolt.Tx) error {
		b, err := txn.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		if b.Get(metaMigratedKey()) != nil {
			log.Println("Migration already completed")
			return nil
		}

		log.Println("Running DB migration (old -> accumulator format)...")

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			key := string(k)

			var snap NetSnapshot
			if err := json.Unmarshal(v, &snap); err != nil {
				continue
			}

			// ---------------------------
			// MIGRATE DAILY
			// ---------------------------
			if strings.HasPrefix(key, "daily:") {
				date := strings.TrimPrefix(key, "daily:")

				acc := Accumulator{
					In:  float64(snap.Rx) / 1024 / 1024 / 1024,
					Out: float64(snap.Tx) / 1024 / 1024 / 1024,
				}

				data, _ := json.Marshal(acc)
				_ = b.Put([]byte("acc:daily:"+date), data)
			}

			// ---------------------------
			// MIGRATE MONTHLY
			// ---------------------------
			if strings.HasPrefix(key, "monthly:") {
				month := strings.TrimPrefix(key, "monthly:")

				acc := Accumulator{
					In:  float64(snap.Rx) / 1024 / 1024 / 1024,
					Out: float64(snap.Tx) / 1024 / 1024 / 1024,
				}

				data, _ := json.Marshal(acc)
				_ = b.Put([]byte("acc:monthly:"+month), data)
			}
		}

		_ = b.Put(metaMigratedKey(), []byte("done"))

		log.Println("Migration complete")
		return nil
	})
}

// --------------------
// UPDATE LOOP (NEW MODEL)
// --------------------

func UpdateNetworkTotals(rx, tx uint64) {
	now := time.Now().UTC()

	_ = db.Update(func(txn *bolt.Tx) error {
		b, err := txn.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			return err
		}

		// last snapshot
		var prev NetSnapshot
		if v := b.Get(metaLastKey()); v != nil {
			_ = json.Unmarshal(v, &prev)
		}

		current := NetSnapshot{Rx: rx, Tx: tx, Ts: now.Unix()}

		// reboot detection
		reboot := rx < prev.Rx || tx < prev.Tx

		var dRx, dTx uint64

		if prev.Rx == 0 && prev.Tx == 0 {
			dRx, dTx = 0, 0
		} else if reboot {
			dRx, dTx = 0, 0
			log.Println("Reboot detected (baseline reset)")
		} else {
			dRx = rx - prev.Rx
			dTx = tx - prev.Tx
		}

		// store snapshot
		raw, _ := json.Marshal(current)
		_ = b.Put(metaLastKey(), raw)

		// accumulate daily
		dk := newDailyKey(now)
		var d Accumulator
		if v := b.Get([]byte(dk)); v != nil {
			_ = json.Unmarshal(v, &d)
		}

		d.In += float64(dRx) / 1024 / 1024 / 1024
		d.Out += float64(dTx) / 1024 / 1024 / 1024

		dd, _ := json.Marshal(d)
		_ = b.Put([]byte(dk), dd)

		// accumulate monthly
		mk := newMonthlyKey(now)
		var m Accumulator
		if v := b.Get([]byte(mk)); v != nil {
			_ = json.Unmarshal(v, &m)
		}

		m.In += float64(dRx) / 1024 / 1024 / 1024
		m.Out += float64(dTx) / 1024 / 1024 / 1024

		md, _ := json.Marshal(m)
		_ = b.Put([]byte(mk), md)

		return nil
	})
}

// --------------------
// LIVE API (FIXED)
// --------------------

func HandleNetworkLive(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()

	var daily, monthly Accumulator

	_ = db.View(func(txn *bolt.Tx) error {
		b := txn.Bucket([]byte(bucket))
		if b == nil {
			return nil
		}

		_ = json.Unmarshal(b.Get([]byte(newDailyKey(now))), &daily)
		_ = json.Unmarshal(b.Get([]byte(newMonthlyKey(now))), &monthly)

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

			if strings.HasPrefix(key, "acc:daily:") {
				var a Accumulator
				_ = json.Unmarshal(v, &a)

				daily = append(daily, DailyStat{
					Date: strings.TrimPrefix(key, "acc:daily:"),
					In:   a.In,
					Out:  a.Out,
				})
			}

			if strings.HasPrefix(key, "acc:monthly:") {
				var a Accumulator
				_ = json.Unmarshal(v, &a)

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