package api

import (
	"net/http"
)

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	connected := s.asimut.LoggedIn()
	if !connected {
		if err := s.asimut.Login(); err == nil {
			connected = true
		}
	}
	writeJSON(w, map[string]bool{"asimut_connected": connected})
}
