package server

import (
	"context"
	"log"
	"time"
)

func (s *Server) NoteGarbageCollection() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.queries.CleanUpOldNotes(context.Background()); err != nil {
				log.Printf("Error cleaning up old notes: %v", err)
			}
		}
	}()
}
