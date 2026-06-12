package server

import (
	"encoding/json"
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

type Client struct {
	username string
	conn     *websocket.Conn
}

type Hub struct {
	mu    sync.RWMutex
	rooms map[string][]*Client
}

func newHub() *Hub {
	return &Hub{
		rooms: make(map[string][]*Client),
	}
}

func (h *Hub) subscribe(conn *websocket.Conn, noteID, username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms[noteID] = append(h.rooms[noteID], &Client{username: username, conn: conn})
}

func (h *Hub) unsubscribe(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for noteID, clients := range h.rooms {
		for i, c := range clients {
			if c.conn == conn {
				h.rooms[noteID] = append(h.rooms[noteID][:i], h.rooms[noteID][i+1:]...)
				if len(h.rooms[noteID]) == 0 {
					delete(h.rooms, noteID)
				}
				break
			}
		}
	}
}

func (h *Hub) broadcastToRoom(noteID string, messageType int, message []byte, exclude *websocket.Conn) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.rooms[noteID] {
		if c.conn != exclude {
			if err := c.conn.WriteMessage(messageType, message); err != nil {
				log.Printf("Broadcast error: %v", err)
			}
		}
	}
}

func (h *Hub) activeRooms() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ids := make([]string, 0, len(h.rooms))
	for id := range h.rooms {
		ids = append(ids, id)
	}
	return ids
}

var hub = newHub()

type wsMessage struct {
	Type     string `json:"type"`
	NoteID   string `json:"note_id"`
	Username string `json:"username"`
	Text     string `json:"text"`
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	var currentNoteID string

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("WebSocket unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case "subscribe":
			if currentNoteID != "" {
				hub.unsubscribe(conn)
			}
			currentNoteID = msg.NoteID
			if currentNoteID != "" {
				hub.subscribe(conn, currentNoteID, msg.Username)
			}
			log.Printf("Client %s subscribed to room %s", msg.Username, currentNoteID)

		case "message":
			if currentNoteID != "" {
				broadcast, _ := json.Marshal(map[string]string{
					"type":     "message",
					"note_id":  currentNoteID,
					"username": msg.Username,
					"text":     msg.Text,
				})
				hub.broadcastToRoom(currentNoteID, websocket.TextMessage, broadcast, conn)
			}

		case "rooms":
			rooms := hub.activeRooms()
			resp, _ := json.Marshal(map[string]interface{}{
				"type":  "rooms",
				"rooms": rooms,
			})
			conn.WriteMessage(websocket.TextMessage, resp)
		}
	}

	hub.unsubscribe(conn)
}
