package src

import "github.com/shirou/gopsutil/v3/disk"

func GetDisk() interface{} {
	io, _ := disk.IOCounters()
	return io
}