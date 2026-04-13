package src

import (
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	prevRx   uint64
	prevTx   uint64
	prevTime time.Time

	NetIface = getInterface()
)

func getInterface() string {
	if v := os.Getenv("NET_IFACE"); v != "" {
		return v
	}
	return "eth0"
}

func readHostNetDev() (uint64, uint64, error) {
	base := "/host/sys/class/net/" + NetIface + "/statistics"

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

func GetNetwork() map[string]interface{} {
	rx, tx, err := readHostNetDev()
	now := time.Now()

	var rxRate, txRate float64

	if err != nil {
		log.Println("net read error:", err)
		return nil
	}

	if prevTime.IsZero() {
		prevRx = rx
		prevTx = tx
		prevTime = now
		return nil
	}

	elapsed := now.Sub(prevTime).Seconds()
	if elapsed <= 0 {
		return nil
	}

	if rx < prevRx || tx < prevTx {
		prevRx = rx
		prevTx = tx
		prevTime = now
		return nil
	}

	rxDelta := float64(rx - prevRx)
	txDelta := float64(tx - prevTx)

	rxRate = rxDelta / elapsed / 1024 / 1024
	txRate = txDelta / elapsed / 1024 / 1024

	prevRx = rx
	prevTx = tx
	prevTime = now

	UpdateNetworkTotals(rx, tx)

	return map[string]interface{}{
		NetIface: map[string]float64{
			"rate_recv": rxRate,
			"rate_sent": txRate,
		},
	}
}

func DebugNet() {
	data, err := os.ReadFile("/host/proc/net/dev")
	if err != nil {
		log.Println("FAILED to read /host/proc/net/dev:", err)
		return
	}
	log.Println("==== /host/proc/net/dev ====")
	log.Println(string(data))
}