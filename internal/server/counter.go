package server

import "sync"

var (
	notesViewCounter = make(map[string]int)
	notesViewMu      sync.Mutex
)

func (s *Server) IncrementNoteViewCounter(noteID string) int {
	notesViewMu.Lock()
	defer notesViewMu.Unlock()
	notesViewCounter[noteID]++
	return notesViewCounter[noteID]
}

func (s *Server) DecrementNoteViewCounter(noteID string) int {
	notesViewMu.Lock()
	defer notesViewMu.Unlock()
	if notesViewCounter[noteID] > 0 {
		notesViewCounter[noteID]--
	}
	return notesViewCounter[noteID]
}

func (s *Server) GetNoteViewCounter(noteID string) int {
	notesViewMu.Lock()
	defer notesViewMu.Unlock()
	return notesViewCounter[noteID]
}

func (s *Server) DeleteNoteViewCounter(noteID string) {
	notesViewMu.Lock()
	defer notesViewMu.Unlock()
	delete(notesViewCounter, noteID)
}

func (s *Server) CleanupNoteViewCounter() {
	notesViewMu.Lock()
	defer notesViewMu.Unlock()

	for noteID, count := range notesViewCounter {
		if count == 0 {
			delete(notesViewCounter, noteID)
		}
	}
}
