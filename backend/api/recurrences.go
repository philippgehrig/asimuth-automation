package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

func (s *Server) listRecurrences(w http.ResponseWriter, r *http.Request) {
	recurrences, err := s.db.ListRecurrences()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, recurrences)
}

func (s *Server) createRecurrence(w http.ResponseWriter, r *http.Request) {
	var rec db.RecurringSchedule
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if rec.DayOfWeek < 0 || rec.DayOfWeek > 6 {
		http.Error(w, "day_of_week must be 0 (Monday) to 6 (Sunday)", http.StatusBadRequest)
		return
	}
	if _, err := scheduler.ParseTime(rec.StartTime); err != nil {
		http.Error(w, "invalid start_time format (expected HH:MM)", http.StatusBadRequest)
		return
	}
	if rec.DurationMinutes < 30 || rec.DurationMinutes > 180 {
		http.Error(w, "duration_minutes must be between 30 and 180", http.StatusBadRequest)
		return
	}
	if len(rec.RoomPriorities) == 0 {
		http.Error(w, "room_priorities must not be empty", http.StatusBadRequest)
		return
	}

	id, err := s.db.CreateRecurrence(rec)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"id": id})
}

func (s *Server) updateRecurrence(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var body struct {
		Active *bool `json:"active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if body.Active != nil {
		if err := s.db.UpdateRecurrenceActive(id, *body.Active); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteRecurrence(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if err := s.db.DeleteRecurrence(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
