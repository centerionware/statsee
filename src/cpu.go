package src

import "github.com/shirou/gopsutil/v3/cpu"

func GetCPU() float64 {
	percent, _ := cpu.Percent(0, false)
	if len(percent) > 0 {
		return percent[0]
	}
	return 0
}