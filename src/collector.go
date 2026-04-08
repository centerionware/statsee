package src

import "time"

func StartCollector() {
	ticker := time.NewTicker(1 * time.Second)

	for range ticker.C {
		collectStats()
	}
}

func collectStats() {
	ts := time.Now().Unix()

	msg := map[string]interface{}{
		"type": "stats",
		"ts":   ts,
		"cpu":  GetCPU(),
		"ram":  GetMemory(),
		"disk": GetDisk(),
		"net":  GetNetwork(),
	}

	broadcast(msg)
}