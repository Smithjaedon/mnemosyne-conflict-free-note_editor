package server

import (
	"Backend/db"
	"Backend/internal/middleware"
	"context"
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

func (h *Hub) getRoomUsernames(noteID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, c := range h.rooms[noteID] {
		seen[c.username] = struct{}{}
	}
	usernames := make([]string, 0, len(seen))
	for u := range seen {
		usernames = append(usernames, u)
	}
	return usernames
}

var hub = newHub()

type wsMessage struct {
	Type     string `json:"type"`
	NoteID   string `json:"note_id"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Cursor   int    `json:"cursor"`
	Version  int32  `json:"version"`
	Target   string `json:"target"`
}

func presenceMsg(msgType, noteID, username string) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"type":     msgType,
		"note_id":  noteID,
		"username": username,
	})
	return data
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	var currentNoteID string
	var currentUsername string
	var currentUserID string

	token := extractToken(c)
	if token != "" {
		if claims, err := middleware.ValidateAccessToken(token); err == nil {
			if uid, err := middleware.GetUserIDFromClaims(claims); err == nil {
				currentUserID = uid
				if user, err := s.queries.GetUserByID(context.Background(), currentUserID); err == nil {
					currentUsername = user.Username
				}
			}
		}
	}

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
				hub.broadcastToRoom(currentNoteID, websocket.TextMessage, presenceMsg("user_left", currentNoteID, currentUsername), nil)
				hub.unsubscribe(conn)
			}
			currentNoteID = msg.NoteID
			if currentUsername == "" {
				currentUsername = msg.Username
			}
			if currentNoteID != "" {
				hub.subscribe(conn, currentNoteID, currentUsername)
				hub.broadcastToRoom(currentNoteID, websocket.TextMessage, presenceMsg("user_joined", currentNoteID, currentUsername), conn)
			}
			log.Printf("Client %s subscribed to room %s", currentUsername, currentNoteID)

		case "message":
			if currentNoteID != "" {
				broadcast, _ := json.Marshal(map[string]interface{}{
					"type":     "message",
					"note_id":  currentNoteID,
					"username": msg.Username,
					"text":     msg.Text,
					"cursor":   msg.Cursor,
					"target":   msg.Target,
				})
				hub.broadcastToRoom(currentNoteID, websocket.TextMessage, broadcast, conn)
			}

		case "update":
			if currentNoteID != "" {
				perm, err := s.queries.GetUserPermissionForNote(context.Background(), db.GetUserPermissionForNoteParams{
					UserID: currentUserID,
					NoteID: currentNoteID,
				})
				if err != nil || (perm != "owner" && perm != "editor") {
					continue
				}

				existing, err := s.queries.GetNoteByID(context.Background(), currentNoteID)
				if err != nil {
					continue
				}

				cs := Changeset{
					Ops:     []Op{{End: 0, Text: msg.Content, DeleteCount: len(existing.Content)}},
					Version: msg.Version,
					Title:   msg.Title,
				}
				if err := s.handleChangeset(context.Background(), currentNoteID, cs); err != nil {
					continue
				}

				updated, err := s.queries.GetNoteByID(context.Background(), currentNoteID)
				newVersion := existing.ContentVersion
				if err == nil {
					newVersion = updated.ContentVersion
				}

				broadcast, _ := json.Marshal(map[string]interface{}{
					"type":            "update",
					"note_id":         currentNoteID,
					"username":        msg.Username,
					"title":           msg.Title,
					"content":         msg.Content,
					"cursor":          msg.Cursor,
					"target":          msg.Target,
					"content_version": newVersion,
				})
				hub.broadcastToRoom(currentNoteID, websocket.TextMessage, broadcast, nil)
			}

		}
	}

	if currentNoteID != "" && currentUsername != "" {
		hub.broadcastToRoom(currentNoteID, websocket.TextMessage, presenceMsg("user_left", currentNoteID, currentUsername), nil)
	}
	hub.unsubscribe(conn)
}
