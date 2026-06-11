package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]string
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*websocket.Conn]string),
	}
}

func (h *Hub) addClient(c *gin.Context, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var username = c.GetString("userID")

	h.clients[conn] = username
}

func (h *Hub) removeClient(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

func (h *Hub) broadcast(messageType int, message []byte, exclude *websocket.Conn) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for conn := range h.clients {
		if conn != exclude {
			if err := conn.WriteMessage(messageType, message); err != nil {
				log.Printf("Broadcast error: %v", err)
			}
		}
	}
}

var hub = newHub()

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	hub.addClient(c, conn)
	log.Printf("WebSocket client connected (%d total)", len(hub.clients))

	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}
		hub.broadcast(messageType, message, conn)
	}

	hub.removeClient(conn)
	log.Printf("WebSocket client disconnected (%d remaining)", len(hub.clients))
}
