package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/philippgehrig/asimuth-automation/backend/asimut"
)

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	if err := s.ensureLoggedIn(); err != nil {
		http.Error(w, "failed to connect to Asimut", http.StatusServiceUnavailable)
		return
	}

	locations, err := s.asimut.GetLocations()
	if err != nil {
		// Session may have expired — retry once with fresh login
		log.Printf("GetLocations failed, retrying with fresh login: %v", err)
		if loginErr := s.asimut.Login(); loginErr != nil {
			http.Error(w, "failed to connect to Asimut", http.StatusServiceUnavailable)
			return
		}
		locations, err = s.asimut.GetLocations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if locations == nil {
		locations = []asimut.Location{}
	}
	writeJSON(w, locations)
}

func (s *Server) ensureLoggedIn() error {
	if !s.asimut.LoggedIn() {
		if err := s.asimut.Login(); err != nil {
			log.Printf("Asimut login failed: %v", err)
			return err
		}
	}
	return nil
}

func (s *Server) getAllowedRooms(w http.ResponseWriter, r *http.Request) {
	ids, err := s.db.GetAllowedRooms()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if ids == nil {
		ids = []int{}
	}
	writeJSON(w, ids)
}

func (s *Server) setAllowedRooms(w http.ResponseWriter, r *http.Request) {
	var ids []int
	if err := json.NewDecoder(r.Body).Decode(&ids); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := s.db.SetAllowedRooms(ids); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
