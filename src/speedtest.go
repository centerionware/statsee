package src

import (
	"log"
	"sync"
	"time"

	"github.com/showwin/speedtest-go/speedtest"
)

type SpeedTestManager struct {
	mu      sync.Mutex
	running bool
}

var speedTest = &SpeedTestManager{}

func (s *SpeedTestManager) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		log.Println("speedtest already running")
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
}

func (s *SpeedTestManager) run() {
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	broadcast(map[string]interface{}{
		"type": "speedtest_start",
	})

	// --------------------
	// Fetch config
	// --------------------
	user, err := speedtest.FetchUserInfo()
	if err != nil {
		log.Println("speedtest user error:", err)
		s.fail(err)
		return
	}

	servers, err := speedtest.FetchServers(user)
	if err != nil {
		log.Println("speedtest server error:", err)
		s.fail(err)
		return
	}

	targets, err := servers.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		log.Println("no speedtest servers found")
		s.fail(err)
		return
	}

	server := targets[0]

	// --------------------
	// Ping
	// --------------------
	server.PingTest()

	broadcast(map[string]interface{}{
		"type":   "speedtest_progress",
		"stage":  "ping",
		"ping":   server.Latency.Seconds() * 1000,
		"server": server.Name,
	})

	// --------------------
	// Download
	// --------------------
	download, err := s.runDownload(server)
	if err != nil {
		log.Println("download error:", err)
		s.fail(err)
		return
	}

	// --------------------
	// Upload
	// --------------------
	upload, err := s.runUpload(server)
	if err != nil {
		log.Println("upload error:", err)
		s.fail(err)
		return
	}

	// --------------------
	// Done
	// --------------------
	broadcast(map[string]interface{}{
		"type":     "speedtest_done",
		"download": download,
		"upload":   upload,
		"ping":     server.Latency.Seconds() * 1000,
		"server":   server.Name,
	})
}

func (s *SpeedTestManager) runDownload(server *speedtest.Server) (float64, error) {
	done := make(chan struct{})

	go func() {
		_ = server.DownloadTest(false)
		close(done)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last float64

	for {
		select {
		case <-done:
			return server.DLSpeed, nil

		case <-ticker.C:
			current := server.DLSpeed

			// Only send if changed (reduces spam)
			if current != last {
				last = current

				broadcast(map[string]interface{}{
					"type":     "speedtest_progress",
					"stage":    "download",
					"download": current,
				})
			}
		}
	}
}

func (s *SpeedTestManager) runUpload(server *speedtest.Server) (float64, error) {
	done := make(chan struct{})

	go func() {
		_ = server.UploadTest(false)
		close(done)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last float64

	for {
		select {
		case <-done:
			return server.ULSpeed, nil

		case <-ticker.C:
			current := server.ULSpeed

			if current != last {
				last = current

				broadcast(map[string]interface{}{
					"type":   "speedtest_progress",
					"stage":  "upload",
					"upload": current,
				})
			}
		}
	}
}

func (s *SpeedTestManager) fail(err error) {
	broadcast(map[string]interface{}{
		"type":  "speedtest_error",
		"error": err.Error(),
	})
}