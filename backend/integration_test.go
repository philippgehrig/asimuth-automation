//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/philippgehrig/asimuth-automation/backend/api"
	"github.com/philippgehrig/asimuth-automation/backend/asimut"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

func TestFullBookingFlow(t *testing.T) {
	// 1. Set up mock Asimut server
	mockAsimut := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/public/login.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			w.Header().Set("Location", "/public/hfm-freiburg.asimut.net")
			w.WriteHeader(302)
		case r.URL.Path == "/services/v2/heartbeat/me":
			w.Write([]byte(`{"response":{"heartbeat":{"loggedin":true},"success":true}}`))
		case r.URL.Path == "/services/v2/locations":
			w.Write([]byte(`{"response":{"locations":[{"id":114,"name":"MBP-326","secondary_name":"Test Room","bookable":true,"type":"location"}]}}`))
		case r.URL.Path == "/services/v2/eventdefault":
			w.Write([]byte(`{"response":{"eventdefault":{"events":[{"id":0,"ar":"Einzelüben","ca":1,"st":"2026-05-15T14:30:00+02:00","en":"2026-05-15T15:00:00+02:00","rs":[{"id":114,"dn":"MBP-326"}],"pe":[{"id":1,"ro":1,"dn":"Test"}],"ps":[{"me":false,"ri":1,"rs":"T","rh":"T","rc":1,"bo":[{"id":1,"fn":"T","ln":"U","un":"t"}]}],"ri":{"e":true},"vi":"visible","cl":[]}]}}}`))
		case r.URL.Path == "/services/v2/event/type=check":
			w.Write([]byte(`{"response":{"success":true,"event_ids":[0]}}`))
		case r.URL.Path == "/services/v2/event/type=save":
			w.Write([]byte(`{"response":{"success":true,"event_ids":[99999]}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer mockAsimut.Close()

	// 2. Set up real DB, scheduler, and server
	database, err := db.New(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	asimutClient := asimut.NewClient(mockAsimut.URL, "test@test.com", "pass")
	sched := scheduler.New()
	sched.Start()
	defer sched.Stop()

	srv := api.NewServer(database, asimutClient, sched, "testpass")
	router := srv.Router()

	// 3. Create a booking via API
	body, _ := json.Marshal(map[string]interface{}{
		"date":             "2026-05-15",
		"start_time":      "14:30",
		"duration_minutes": 30,
		"room_priorities":  []int{114},
	})

	req := httptest.NewRequest("POST", "/api/bookings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer testpass")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("create booking: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Verify booking exists
	req = httptest.NewRequest("GET", "/api/bookings", nil)
	req.Header.Set("Authorization", "Bearer testpass")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var bookings []db.BookingWish
	json.NewDecoder(w.Body).Decode(&bookings)
	if len(bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(bookings))
	}
	if bookings[0].Status != "scheduled" && bookings[0].Status != "pending" {
		t.Fatalf("expected pending/scheduled status, got %s", bookings[0].Status)
	}

	// 5. Verify rooms endpoint works (login first)
	if err := asimutClient.Login(); err != nil {
		t.Fatalf("asimut login: %v", err)
	}
	req = httptest.NewRequest("GET", "/api/rooms", nil)
	req.Header.Set("Authorization", "Bearer testpass")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("rooms: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 6. Verify delete works
	deleteReq := httptest.NewRequest("DELETE", "/api/bookings/"+bookings[0].ID, nil)
	deleteReq.Header.Set("Authorization", "Bearer testpass")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, deleteReq)

	if w.Code != 204 {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}
}
