package api

import (
	"net/http"
)

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]bool{"asimut_connected": s.asimut.LoggedIn()})
}
