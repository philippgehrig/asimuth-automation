package api

import (
	"net/http"
)

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	locations, err := s.asimut.GetLocations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, locations)
}
