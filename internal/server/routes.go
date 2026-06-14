package server

import (
	"Backend/db"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"https://mnemosyne-frontend-nine.vercel.app", "http://localhost:5173", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.GET("/health", s.healthHandler)

	r.POST("/register", s.RegisterHandler)
	r.POST("/login", s.LoginHandler)
	r.POST("/logout", s.LogoutHandler)
	r.GET("/me", s.MeHandler)

	auth := r.Group("/")
	auth.Use(s.AuthMiddleware())
	{
		auth.GET("/live", s.handleWebSocket)
		auth.POST("/notes", s.CreateNoteHandler)
		auth.GET("/notes", s.GetNotesHandler)
		auth.GET("/notes/:id", s.GetNoteHandler)
		auth.PUT("/notes/:id", s.UpdateNoteHandler)
		auth.DELETE("/notes/:id", s.DeleteNoteHandler)

		auth.GET("/notes/shared", s.GetSharedNotesByUserIdHandler)
		auth.GET("/notes/:id/shared", s.GetSharedNoteByIDHandler)

		auth.GET("/notes/:id/users", s.GetNoteUsersHandler)
		auth.POST("/notes/:id/share", s.AddSharedNoteHandler)
		auth.PUT("/notes/:id/share/:userId", s.UpdateSharedNoteHandler)
		auth.DELETE("/notes/:id/share/:userId", s.RemoveSharedNoteHandler)

		auth.GET("/notes/:id/viewers", s.GetNoteViewersHandler)
		auth.POST("/notes/:id/close", s.LeaveNoteHandler)
		auth.GET("/notes/:id/permission", s.GetPermissionHandler)

		auth.GET("/users", s.SearchUsersHandler)
	}

	return r
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

// ─── Notes CRUD (owned) ───────────────────────────────────

func (s *Server) CreateNoteHandler(c *gin.Context) {
	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("userID")

	noteID := uuid.NewString()

	note, err := s.queries.CreateNote(c.Request.Context(), db.CreateNoteParams{
		ID:      noteID,
		OwnerID: userID,
		Title:   req.Title,
		Content: req.Content,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.queries.AddNoteUser(c.Request.Context(), db.AddNoteUserParams{
		UserID:      userID,
		NoteID:      noteID,
		Permissions: "owner",
	})

	c.JSON(http.StatusOK, note)
}

func (s *Server) GetNotesHandler(c *gin.Context) {
	userID := c.GetString("userID")

	notes, err := s.queries.GetNotesByUser(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	shared, err := s.queries.GetSharedNoteByOwnerID(c.Request.Context(), userID)
	if err != nil {
		shared = nil
	}
	sharedSet := make(map[string]struct{}, len(shared))
	for _, id := range shared {
		sharedSet[id] = struct{}{}
	}

	resp := make([]gin.H, len(notes))
	for i, n := range notes {
		_, isShared := sharedSet[n.ID]
		resp[i] = gin.H{
			"id":              n.ID,
			"title":           n.Title,
			"content":         n.Content,
			"content_version": n.ContentVersion,
			"created_at":      n.CreatedAt,
			"updated_at":      n.UpdatedAt,
			"owner_id":        n.OwnerID,
			"view_count":      s.GetNoteViewCounter(n.ID),
			"is_shared":       isShared,
		}
	}

	c.JSON(http.StatusOK, resp)
}

func (s *Server) GetNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	if _, err := s.queries.GetUserPermissionForNote(c.Request.Context(), db.GetUserPermissionForNoteParams{
		UserID: userID,
		NoteID: noteID,
	}); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	note, err := s.queries.GetNoteByID(c.Request.Context(), noteID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.IncrementNoteViewCounter(noteID)
	c.JSON(http.StatusOK, note)
}

func (s *Server) UpdateNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	var perm string
	var err error
	if perm, err = s.queries.GetUserPermissionForNote(c.Request.Context(), db.GetUserPermissionForNoteParams{
		UserID: userID,
		NoteID: noteID,
	}); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	if perm != "owner" && perm != "editor" {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Title   string `json:"title" binding:"required"`
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	note, err := s.queries.UpdateNote(c.Request.Context(), db.UpdateNoteParams{
		ID:      noteID,
		Title:   req.Title,
		Content: req.Content,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, note)
}

func (s *Server) DeleteNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	existing, err := s.queries.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	err = s.queries.DeleteNote(c.Request.Context(), noteID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.DeleteNoteViewCounter(noteID)

	c.JSON(http.StatusOK, gin.H{"message": "Note deleted"})
}

// ─── Shared notes (read) ──────────────────────────────────

func (s *Server) GetSharedNotesByUserIdHandler(c *gin.Context) {
	userID := c.GetString("userID")

	share_notes, err := s.queries.GetSharedNotesByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, share_notes)
}

func (s *Server) GetSharedNoteByIDHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	note, err := s.queries.GetSharedNoteByID(c.Request.Context(), db.GetSharedNoteByIDParams{
		UserID: userID,
		ID:     noteID,
	})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "note not found or not shared with you"})
		return
	}

	s.IncrementNoteViewCounter(noteID)
	c.JSON(http.StatusOK, note)
}

// ─── Share management ─────────────────────────────────────

func (s *Server) GetNoteUsersHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	existing, err := s.queries.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	users, err := s.queries.GetNoteUsers(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (s *Server) AddSharedNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	existing, err := s.queries.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		TargetUserID string `json:"user_id" binding:"required"`
		Permissions  string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	note, err := s.queries.AddNoteUser(c.Request.Context(), db.AddNoteUserParams{
		UserID:      req.TargetUserID,
		NoteID:      noteID,
		Permissions: req.Permissions,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, note)
}

func (s *Server) UpdateSharedNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	targetUserID := c.Param("userId")
	userID := c.GetString("userID")

	existing, err := s.queries.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Permissions string `json:"permissions" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = s.queries.UpdateNoteUserPermissions(c.Request.Context(), db.UpdateNoteUserPermissionsParams{
		UserID:      targetUserID,
		NoteID:      noteID,
		Permissions: req.Permissions,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Permissions updated"})
}

func (s *Server) RemoveSharedNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	targetUserID := c.Param("userId")
	userID := c.GetString("userID")

	existing, err := s.queries.GetNoteByID(c.Request.Context(), noteID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if existing.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	err = s.queries.RemoveNoteUser(c.Request.Context(), db.RemoveNoteUserParams{
		UserID: targetUserID,
		NoteID: noteID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User removed from note"})
}

// ─── Presence ─────────────────────────────────────────────

func (s *Server) GetNoteViewersHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	if _, err := s.queries.GetUserPermissionForNote(c.Request.Context(), db.GetUserPermissionForNoteParams{
		UserID: userID,
		NoteID: noteID,
	}); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	count := s.GetNoteViewCounter(noteID)
	viewers := hub.getRoomUsernames(noteID)
	c.JSON(http.StatusOK, gin.H{"view_count": count, "viewers": viewers})
}

func (s *Server) LeaveNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	if _, err := s.queries.GetUserPermissionForNote(c.Request.Context(), db.GetUserPermissionForNoteParams{
		UserID: userID,
		NoteID: noteID,
	}); err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	s.DecrementNoteViewCounter(noteID)
	c.JSON(http.StatusOK, gin.H{"view_count": s.GetNoteViewCounter(noteID)})
}

// ─── Users ────────────────────────────────────────────────

func (s *Server) SearchUsersHandler(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query is required"})
		return
	}

	users, err := s.queries.SearchUsers(c.Request.Context(), "%"+q+"%")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (s *Server) GetPermissionHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	permission, err := s.queries.GetUserPermissionForNote(c.Request.Context(), db.GetUserPermissionForNoteParams{
		UserID: userID,
		NoteID: noteID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"permission": permission})
}
