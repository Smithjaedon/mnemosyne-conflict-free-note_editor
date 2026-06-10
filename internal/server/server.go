package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"Backend/db"
	"Backend/internal/database"
)

type Server struct {
	port    int
	db      database.Service
	queries *db.Queries
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	dbService, queries := database.New()

	NewServer := &Server{
		port: port,

		db:      dbService,
		queries: queries,
	}

	NewServer.NoteGarbageCollection()

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
