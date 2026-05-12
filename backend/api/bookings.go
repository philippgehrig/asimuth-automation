package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

func (s *Server) listBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := s.db.ListBookings()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if bookings == nil {
		bookings = []db.BookingWish{}
	}
	writeJSON(w, bookings)
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var wish db.BookingWish
	if err := json.NewDecoder(r.Body).Decode(&wish); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if wish.Date == "" {
		http.Error(w, "date is required", http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", wish.Date); err != nil {
		http.Error(w, "date must be in YYYY-MM-DD format", http.StatusBadRequest)
		return
	}
	hm, err := scheduler.ParseTime(wish.StartTime)
	if err != nil || hm[0] < 0 || hm[0] > 23 || hm[1] < 0 || hm[1] > 59 {
		http.Error(w, "invalid start_time", http.StatusBadRequest)
		return
	}
	if wish.DurationMinutes < 30 || wish.DurationMinutes > 180 {
		http.Error(w, "duration_minutes must be between 30 and 180", http.StatusBadRequest)
		return
	}
	if len(wish.RoomPriorities) == 0 {
		http.Error(w, "room_priorities must not be empty", http.StatusBadRequest)
		return
	}

	id, err := s.db.CreateBooking(wish)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	wish.ID = id
	s.ScheduleBookingJob(id, wish)

	writeJSON(w, map[string]string{"id": id})
}

func (s *Server) deleteBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.scheduler.Cancel(id)

	if err := s.db.DeleteBooking(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ScheduleBookingJob calculates the trigger time and schedules the booking execution.
func (s *Server) ScheduleBookingJob(id string, wish db.BookingWish) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		log.Printf("failed to load timezone: %v", err)
		return
	}

	trigger, err := scheduler.CalculateTriggerTime(wish.Date, wish.StartTime, loc)
	if err != nil {
		log.Printf("failed to calculate trigger time for booking %s: %v", id, err)
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, fmt.Sprintf("invalid schedule: %v", err))
		return
	}

	if trigger.Before(time.Now()) {
		// Trigger time already passed — execute immediately instead of failing
		_ = s.db.UpdateBookingStatus(id, "executing", "", nil, "")
		go s.executeBooking(id, wish)
		return
	}

	_ = s.db.UpdateBookingStatus(id, "scheduled", "", nil, "")

	s.scheduler.Schedule(&scheduler.Job{
		ID:          id,
		TriggerTime: trigger,
		Execute: func() {
			s.executeBooking(id, wish)
		},
	})
}

// executeBooking logs into Asimut and attempts to book a room from the priority list.
func (s *Server) executeBooking(id string, wish db.BookingWish) {
	if err := s.asimut.Login(); err != nil {
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, fmt.Sprintf("login failed: %v", err))
		return
	}

	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, fmt.Sprintf("timezone error: %v", err))
		return
	}

	slotDate, err := time.ParseInLocation("2006-01-02", wish.Date, loc)
	if err != nil {
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, fmt.Sprintf("invalid date: %v", err))
		return
	}

	hm, err := scheduler.ParseTime(wish.StartTime)
	if err != nil {
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, fmt.Sprintf("invalid start time: %v", err))
		return
	}

	start := time.Date(slotDate.Year(), slotDate.Month(), slotDate.Day(), hm[0], hm[1], 0, 0, loc)

	// Initial booking duration: 30 minutes
	initialDuration := 30 * time.Minute
	end := start.Add(initialDuration)

	var bookedRoom string
	var eventID int

	for _, roomID := range wish.RoomPriorities {
		result, err := s.asimut.BookRoom(roomID, start, end)
		if err == nil && result.Success {
			bookedRoom = fmt.Sprintf("%d", roomID)
			eventID = result.EventID
			break
		}
	}

	if bookedRoom == "" {
		_ = s.db.UpdateBookingStatus(id, "failed", "", nil, "no room available")
		return
	}

	// Extend in 15-minute increments up to desired duration
	totalMinutes := 30
	desiredMinutes := wish.DurationMinutes

	for totalMinutes < desiredMinutes {
		newEnd := end.Add(15 * time.Minute)
		_, err := s.asimut.ExtendBooking(eventID, newEnd)
		if err != nil {
			break
		}
		end = newEnd
		totalMinutes += 15
	}

	status := "booked"
	if totalMinutes < desiredMinutes {
		status = "partially_booked"
	}

	resultDuration := totalMinutes
	_ = s.db.UpdateBookingStatus(id, status, bookedRoom, &resultDuration, "")
}
