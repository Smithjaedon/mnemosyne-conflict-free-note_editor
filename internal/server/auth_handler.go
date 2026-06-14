package server

import (
	"Backend/db"
	"Backend/internal/middleware"
	"net/http"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func extractToken(c *gin.Context) string {
	if t := c.GetHeader("Authorization"); len(t) > 7 && t[:7] == "Bearer " {
		return t[7:]
	}
	if t, err := c.Cookie("access_token"); err == nil {
		return t
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	return ""
}

func (s *Server) RegisterHandler(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	userID := uuid.NewString()

	user, err := s.queries.CreateUser(c.Request.Context(), db.CreateUserParams{
		ID:             userID,
		Username:       req.Username,
		Email:          req.Email,
		HashedPassword: string(hash),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create user"})
		return
	}

	token, err := middleware.CreateAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "message": "registered"})
}

func (s *Server) LoginHandler(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.queries.GetUserByUsername(c.Request.Context(), req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := middleware.CreateAccessToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create access token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "message": "logged in"})
}

func (s *Server) MeHandler(c *gin.Context) {
	token := extractToken(c)
	if token == "" {
		c.JSON(http.StatusOK, gin.H{"user": nil})
		return
	}

	userID, err := middleware.RequireAuth(token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user": nil})
		return
	}

	user, err := s.queries.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"user": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": user.ID, "email": user.Email, "username": user.Username})
}

func (s *Server) LogoutHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (s *Server) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractToken(c)
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing access token"})
			c.Abort()
			return
		}

		userID, err := middleware.RequireAuth(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid access token"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Next()
	}
}
