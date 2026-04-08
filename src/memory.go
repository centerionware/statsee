package src

import "github.com/shirou/gopsutil/v3/mem"

func GetMemory() map[string]float64 {
	memStat, _ := mem.VirtualMemory()

	return map[string]float64{
		"used": float64(memStat.Used) / 1024 / 1024,
		"free": float64(memStat.Available) / 1024 / 1024,
	}
}