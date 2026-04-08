package src

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	wsClients = make(map[*websocket.Conn]bool)
	wsLock    sync.Mutex
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
)

func WSHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("[WS] upgrade error:", err)
		return
	}

	log.Println("[WS] client connected")

	wsLock.Lock()
	wsClients[conn] = true
	wsLock.Unlock()

	defer func() {
		log.Println("[WS] client disconnected")
		wsLock.Lock()
		delete(wsClients, conn)
		wsLock.Unlock()
		conn.Close()
	}()

	for {
		var msg map[string]string
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("[WS] read error:", err)
			return
		}

		log.Println("[WS] received:", msg)

		if msg["type"] == "speedtest" {
			log.Println("[WS] starting speedtest")
			speedTest.Start()
		}
	}
}

func broadcast(msg interface{}) {
	wsLock.Lock()
	defer wsLock.Unlock()

	for c := range wsClients {
		if err := c.WriteJSON(msg); err != nil {
			log.Println("[WS] write error:", err)
			c.Close()
			delete(wsClients, c)
		}
	}
}