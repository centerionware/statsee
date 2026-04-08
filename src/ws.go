package src

import (
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
		return
	}

	wsLock.Lock()
	wsClients[conn] = true
	wsLock.Unlock()

	defer func() {
		wsLock.Lock()
		delete(wsClients, conn)
		wsLock.Unlock()
		conn.Close()
	}()

	for {
		var msg map[string]string
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		if msg["type"] == "speedtest" {
        	speedTest.Start()
        }
	}
}

func broadcast(msg interface{}) {
	wsLock.Lock()
	defer wsLock.Unlock()

	for c := range wsClients {
		c.WriteJSON(msg)
	}
}