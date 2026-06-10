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
		AllowOrigins:     []string{"http://localhost:5173"}, // Add your frontend URL
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true, // Enable cookies/auth
	}))

	r.GET("/health", s.healthHandler)

	r.POST("/register", s.RegisterHandler)
	r.POST("/login", s.LoginHandler)
	r.POST("/logout", s.LogoutHandler)

	auth := r.Group("/")
	auth.Use(s.AuthMiddleware())
	{
		auth.GET("/me", s.MeHandler)
		auth.POST("/notes", s.CreateNoteHandler)
		auth.GET("/notes", s.GetNotesHandler)
		auth.GET("/users", s.SearchUsersHandler)
		auth.GET("/notes/:id", s.GetNoteHandler)
		auth.PUT("/notes/:id", s.UpdateNoteHandler)
		auth.DELETE("/notes/:id", s.DeleteNoteHandler)
	}

	return r
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

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

	c.JSON(http.StatusOK, note)
}

func (s *Server) GetNotesHandler(c *gin.Context) {
	userID := c.GetString("userID")

	notes, err := s.queries.GetNotesByOwnerID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, notes)
}

func (s *Server) GetNoteHandler(c *gin.Context) {
	noteID := c.Param("id")
	userID := c.GetString("userID")

	note, err := s.queries.GetNoteByID(c.Request.Context(), noteID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if note.OwnerID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "unauthorized"})
		return
	}

	c.JSON(http.StatusOK, note)
}

func (s *Server) UpdateNoteHandler(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"message": "Note deleted"})
}
