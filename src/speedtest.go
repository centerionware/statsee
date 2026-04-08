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
		log.Println("[speedtest] already running")
		return
	}
	s.running = true
	s.mu.Unlock()

	go s.run()
}

func (s *SpeedTestManager) run() {
	log.Println("[speedtest] started")

	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		log.Println("[speedtest] finished")
	}()

	broadcast(map[string]interface{}{
		"type": "speedtest_start",
	})

	user, err := speedtest.FetchUserInfo()
	if err != nil {
		log.Println("[speedtest] user error:", err)
		s.fail(err)
		return
	}

	servers, err := speedtest.FetchServers()
	if err != nil {
		log.Println("[speedtest] server error:", err)
		s.fail(err)
		return
	}

	targets, err := servers.FindServer([]int{})
	if err != nil || len(targets) == 0 {
		log.Println("[speedtest] no servers found")
		s.fail(err)
		return
	}

	server := targets[0]

	log.Println("[speedtest] using server:", server.Name)

	server.PingTest(nil)

	download, err := s.runDownload(server)
	if err != nil {
		s.fail(err)
		return
	}

	upload, err := s.runUpload(server)
	if err != nil {
		s.fail(err)
		return
	}

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
		_ = server.DownloadTest()
		close(done)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last float64

	for {
		select {
		case <-done:
			log.Println("[speedtest] download final:", server.DLSpeed)
			return float64(server.DLSpeed), nil

		case <-ticker.C:
			current := float64(server.DLSpeed)
			if current != last {
				last = current

				log.Println("[speedtest] download:", current)

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
		_ = server.UploadTest()
		close(done)
	}()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var last float64

	for {
		select {
		case <-done:
			log.Println("[speedtest] upload final:", server.ULSpeed)
			return float64(server.ULSpeed), nil

		case <-ticker.C:
			current := float64(server.ULSpeed)
			if current != last {
				last = current

				log.Println("[speedtest] upload:", current)

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
	log.Println("[speedtest] failed:", err)

	broadcast(map[string]interface{}{
		"type":  "speedtest_error",
		"error": err.Error(),
	})
}