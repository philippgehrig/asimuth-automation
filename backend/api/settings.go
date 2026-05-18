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

func (s *Server) reconnect(w http.ResponseWriter, r *http.Request) {
	s.asimut.InvalidateSession()
	if err := s.asimut.Login(); err != nil {
		writeJSON(w, map[string]interface{}{"asimut_connected": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]interface{}{"asimut_connected": true})
}
