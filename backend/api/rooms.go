package api

import (
	"net/http"
)

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	if !s.asimut.LoggedIn() {
		if err := s.asimut.Login(); err != nil {
			http.Error(w, "failed to connect to Asimut", http.StatusServiceUnavailable)
			return
		}
	}

	locations, err := s.asimut.GetLocations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, locations)
}
