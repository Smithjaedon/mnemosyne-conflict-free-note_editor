package server

import (
	"Backend/db"
	"Backend/internal/middleware"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var otUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type OTClient struct {
	username string
	conn     *websocket.Conn
}

type OTHub struct {
	mu    sync.RWMutex
	rooms map[string][]*OTClient
}

type Op struct {
	End         int    `json:"end"`
	Text        string `json:"text"`
	DeleteCount int    `json:"delete_count"`
}

type Changeset struct {
	Prev    int   `json:"prev"`
	Curr    int   `json:"curr"`
	Ops     []Op  `json:"ops"`
	Version int32 `json:"version"`
	Title   string `json:"title"`
}

var changeHistory = make(map[string][]Changeset)

func newOTHub() *OTHub {
	return &OTHub{
		rooms: make(map[string][]*OTClient),
	}
}

func (h *OTHub) otSubscribe(conn *websocket.Conn, noteID, username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms[noteID] = append(h.rooms[noteID], &OTClient{username: username, conn: conn})
}

func (h *OTHub) otUnsubscribe(conn *websocket.Conn) {
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

func (h *OTHub) otBroadcastToRoom(noteID string, messageType int, message []byte, exclude *websocket.Conn) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.rooms[noteID] {
		if c.conn != exclude {
			if err := c.conn.WriteMessage(messageType, message); err != nil {
				log.Printf("OT broadcast error: %v", err)
			}
		}
	}
}

func (h *OTHub) otGetRoomUsernames(noteID string) []string {
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

var otHub = newOTHub()

type otMessage struct {
	Type     string `json:"type"`
	NoteID   string `json:"note_id"`
	Username string `json:"username"`
	Text     string `json:"text"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Cursor   int    `json:"cursor"`
	Version  int32  `json:"version"`
}

func otPresenceMsg(msgType, noteID, username string) []byte {
	data, _ := json.Marshal(map[string]interface{}{
		"type":     msgType,
		"note_id":  noteID,
		"username": username,
	})
	return data
}

func (s *Server) handleOTWebSocket(c *gin.Context) {
	conn, err := otUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("OT WebSocket upgrade error: %v", err)
		return
	}

	var currentNoteID string
	var currentUsername string
	var currentUserID string

	token, err := c.Cookie("access_token")
	if err == nil {
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
			log.Printf("OT WebSocket read error: %v", err)
			break
		}

		var msg otMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("OT WebSocket unmarshal error: %v", err)
			continue
		}

		switch msg.Type {
		case "subscribe":
			if currentNoteID != "" {
				otHub.otBroadcastToRoom(currentNoteID, websocket.TextMessage, otPresenceMsg("user_left", currentNoteID, currentUsername), nil)
				otHub.otUnsubscribe(conn)
			}
			currentNoteID = msg.NoteID
			if currentUsername == "" {
				currentUsername = msg.Username
			}
			if currentNoteID != "" {
				otHub.otSubscribe(conn, currentNoteID, currentUsername)
				otHub.otBroadcastToRoom(currentNoteID, websocket.TextMessage, otPresenceMsg("user_joined", currentNoteID, currentUsername), conn)
			}
			log.Printf("OT client %s subscribed to room %s", currentUsername, currentNoteID)

		case "message":
			if currentNoteID != "" {
				broadcast, _ := json.Marshal(map[string]interface{}{
					"type":     "message",
					"note_id":  currentNoteID,
					"username": msg.Username,
					"text":     msg.Text,
					"cursor":   msg.Cursor,
				})
				otHub.otBroadcastToRoom(currentNoteID, websocket.TextMessage, broadcast, conn)
			}

		case "update":
			if currentNoteID != "" {
				broadcast, _ := json.Marshal(map[string]interface{}{
					"type":     "update",
					"note_id":  currentNoteID,
					"username": msg.Username,
					"content":  msg.Content,
					"cursor":   msg.Cursor,
				})
				otHub.otBroadcastToRoom(currentNoteID, websocket.TextMessage, broadcast, conn)

				var perm string
				var err error

				perm, err = s.queries.GetUserPermissionForNote(context.Background(), db.GetUserPermissionForNoteParams{
					UserID: currentUserID,
					NoteID: currentNoteID,
				})
				if err != nil || (perm != "owner" && perm != "editor") {
					continue
				}

				if currentUserID != "" {
					existing, err := s.queries.GetNoteByID(context.Background(), currentNoteID)
					if err == nil && existing.OwnerID == currentUserID || perm == "editor" {
						title := msg.Title
						if title == "" {
							title = existing.Title
						}
						s.queries.UpdateNote(context.Background(), db.UpdateNoteParams{
							ID:      currentNoteID,
							Title:   title,
							Content: msg.Content,
						})
					}
				}
			}
		}
	}

	if currentNoteID != "" && currentUsername != "" {
		otHub.otBroadcastToRoom(currentNoteID, websocket.TextMessage, otPresenceMsg("user_left", currentNoteID, currentUsername), nil)
	}
	otHub.otUnsubscribe(conn)
}

func (s *Server) handleChangeset(ctx context.Context, noteID string, payload Changeset) error {
	note, err := s.queries.GetNoteByID(ctx, noteID)
	if err != nil {
		return fmt.Errorf("note not found: %w", err)
	}
	version := note.ContentVersion

	if payload.Version == version {
		if err := s.apply(ctx, &note, payload); err != nil {
			return err
		}
	} else if payload.Version < version {
		for _, prev := range changeHistory[noteID] {
			if prev.Version > payload.Version {
				payload = transform(payload, prev)
			}
		}
		if err := s.apply(ctx, &note, payload); err != nil {
			return err
		}
	} else {
		return fmt.Errorf("version conflict")
	}

	changeHistory[noteID] = append(changeHistory[noteID], payload)
	if len(changeHistory[noteID]) > 5 {
		changeHistory[noteID] = changeHistory[noteID][1:]
	}
	return nil
}

func (s *Server) handlePayload(c *gin.Context, noteID string, payload Changeset) {
	if err := s.handleChangeset(c.Request.Context(), noteID, payload); err != nil {
		c.JSON(409, gin.H{"error": err.Error()})
	}
}

func (s *Server) apply(ctx context.Context, note *db.Note, payload Changeset) error {
	content := note.Content

	end := payload.Ops[0].End + payload.Ops[0].DeleteCount
	if end > len(content) {
		end = len(content)
	}

	content = content[:payload.Ops[0].End] + payload.Ops[0].Text + content[end:]

	title := note.Title
	if payload.Title != "" {
		title = payload.Title
	}

	updated, err := s.queries.UpdateNote(ctx, db.UpdateNoteParams{
		ID:      note.ID,
		Title:   title,
		Content: content,
	})
	if err != nil {
		return err
	}
	*note = updated
	return nil
}

func transform(currChangeset Changeset, prevChangeset Changeset) Changeset {
	var shift int

	if prevChangeset.Ops[0].End <= currChangeset.Ops[0].End {
		shift = len(prevChangeset.Ops[0].Text) - int(prevChangeset.Ops[0].DeleteCount)
		currChangeset.Ops[0].End += shift
		if currChangeset.Ops[0].End < 0 {
			currChangeset.Ops[0].End = 0
		}

	}

	return currChangeset
}
