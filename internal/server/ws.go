package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	noteClients = make(map[string]map[*websocket.Conn]bool)
	clientsMu   sync.Mutex
)

func (s *Server) HandleWebSocket(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	clientsMu.Lock()
	if noteClients[noteID] == nil {
		noteClients[noteID] = make(map[*websocket.Conn]bool)
	}
	noteClients[noteID][conn] = true
	count := len(noteClients[noteID])
	clientsMu.Unlock()

	broadcastToNote(noteID, presenceMsg(userID, "join", count), conn)

	defer func() {
		clientsMu.Lock()
		delete(noteClients[noteID], conn)
		count := len(noteClients[noteID])
		if count == 0 {
			delete(noteClients, noteID)
		}
		clientsMu.Unlock()
		conn.Close()

		broadcastToNote(noteID, presenceMsg(userID, "leave", count), nil)
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}
		broadcastToNote(noteID, message, conn)
	}
}

func broadcastToNote(noteID string, message interface{}, sender *websocket.Conn) {
	clientsMu.Lock()
	defer clientsMu.Unlock()
	for client := range noteClients[noteID] {
		if client != sender {
			client.WriteJSON(message)
		}
	}
}

func presenceMsg(userID, event string, count int) map[string]interface{} {
	return map[string]interface{}{
		"type":       "presence",
		"user_id":    userID,
		"event":      event,
		"view_count": count,
	}
}
