# Asimut Room Booking Bot — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-hosted web app that automatically books practice rooms on Asimut at the exact moment they become available.

**Architecture:** Go backend (REST API + scheduler + Asimut HTTP client + SQLite) with a Vue 3 SPA frontend. Dockerized with two containers. The scheduler calculates trigger times based on the 27.5-hour advance window and fires precise HTTP requests to Asimut.

**Tech Stack:** Go 1.22+, SQLite (via modernc.org/sqlite), Vue 3 + TypeScript + Tailwind CSS + Yarn, Docker Compose

---

## File Structure

```
backend/
├── main.go                      # Entry point, wires dependencies
├── go.mod
├── go.sum
├── config/
│   └── config.go                # Environment variable loading
├── asimut/
│   ├── client.go                # HTTP client for Asimut API
│   └── client_test.go
├── db/
│   ├── sqlite.go                # DB init, migrations
│   ├── bookings.go              # Booking wish CRUD
│   ├── bookings_test.go
│   ├── recurrences.go           # Recurring schedule CRUD
│   └── recurrences_test.go
├── scheduler/
│   ├── scheduler.go             # Job scheduling + trigger logic
│   └── scheduler_test.go
├── api/
│   ├── router.go                # Route definitions + middleware
│   ├── auth.go                  # Password auth middleware
│   ├── auth_test.go
│   ├── bookings.go              # Booking wish handlers
│   ├── bookings_test.go
│   ├── recurrences.go           # Recurring schedule handlers
│   ├── recurrences_test.go
│   ├── rooms.go                 # Room list proxy handler
│   └── settings.go              # Settings/health handler
├── Dockerfile
frontend/
├── package.json
├── yarn.lock
├── tsconfig.json
├── vite.config.ts
├── tailwind.config.js
├── postcss.config.js
├── index.html
├── src/
│   ├── main.ts
│   ├── App.vue
│   ├── router.ts
│   ├── api.ts                   # API client with auth
│   ├── stores/
│   │   ├── auth.ts              # Auth state
│   │   ├── bookings.ts          # Booking wishes state
│   │   ├── recurrences.ts       # Recurring schedules state
│   │   └── rooms.ts             # Room list state
│   ├── views/
│   │   ├── LoginView.vue
│   │   ├── DashboardView.vue
│   │   ├── CreateBookingView.vue
│   │   ├── RoomsView.vue
│   │   └── SettingsView.vue
│   └── components/
│       ├── BookingCard.vue
│       ├── RoomPriorityList.vue
│       └── StatusBadge.vue
├── Dockerfile
docker-compose.yml
.env.example
```

---

## Task 1: Go Project Scaffold + Config

**Files:**
- Create: `backend/go.mod`
- Create: `backend/main.go`
- Create: `backend/config/config.go`

- [ ] **Step 1: Initialize Go module**

```bash
cd backend && go mod init github.com/philippgehrig/asimuth-automation/backend
```

- [ ] **Step 2: Create config loader**

Create `backend/config/config.go`:

```go
package config

import "os"

type Config struct {
	AsimutEmail    string
	AsimutPassword string
	AppPassword    string
	DatabasePath   string
	Port           string
}

func Load() *Config {
	return &Config{
		AsimutEmail:    os.Getenv("ASIMUT_EMAIL"),
		AsimutPassword: os.Getenv("ASIMUT_PASSWORD"),
		AppPassword:    os.Getenv("APP_PASSWORD"),
		DatabasePath:   getEnvOrDefault("DATABASE_PATH", "/data/asimut.db"),
		Port:           getEnvOrDefault("PORT", "8080"),
	}
}

func getEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

- [ ] **Step 3: Create minimal main.go**

Create `backend/main.go`:

```go
package main

import (
	"fmt"
	"log"

	"github.com/philippgehrig/asimuth-automation/backend/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting asimut-automation on port %s", cfg.Port)
	fmt.Println("Server not yet implemented")
}
```

- [ ] **Step 4: Verify it compiles**

```bash
cd backend && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat: scaffold Go backend with config loader"
```

---

## Task 2: SQLite Database Layer

**Files:**
- Create: `backend/db/sqlite.go`
- Create: `backend/db/bookings.go`
- Create: `backend/db/bookings_test.go`
- Create: `backend/db/recurrences.go`
- Create: `backend/db/recurrences_test.go`

- [ ] **Step 1: Add SQLite dependency**

```bash
cd backend && go get modernc.org/sqlite && go get github.com/google/uuid
```

- [ ] **Step 2: Write failing test for DB initialization**

Create `backend/db/bookings_test.go`:

```go
package db

import (
	"os"
	"testing"
)

func TestNewDB_CreatesTablesOnInit(t *testing.T) {
	path := t.TempDir() + "/test.db"
	defer os.Remove(path)

	database, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	defer database.Close()

	var count int
	err = database.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='booking_wishes'").Scan(&count)
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected booking_wishes table to exist, got count=%d", count)
	}
}
```

- [ ] **Step 3: Run test to verify failure**

```bash
cd backend && go test ./db/ -v
```

Expected: compilation error (New not defined)

- [ ] **Step 4: Implement DB initialization**

Create `backend/db/sqlite.go`:

```go
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type DB struct {
	db *sql.DB
}

func New(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	sqlDB.Exec("PRAGMA journal_mode=WAL")
	sqlDB.Exec("PRAGMA foreign_keys=ON")

	if err := migrate(sqlDB); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &DB{db: sqlDB}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS recurring_schedules (
			id TEXT PRIMARY KEY,
			day_of_week INTEGER NOT NULL,
			start_time TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			room_priorities TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);

		CREATE TABLE IF NOT EXISTS booking_wishes (
			id TEXT PRIMARY KEY,
			date TEXT NOT NULL,
			start_time TEXT NOT NULL,
			duration_minutes INTEGER NOT NULL,
			room_priorities TEXT NOT NULL,
			recurrence_id TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			result_room TEXT,
			result_duration INTEGER,
			failure_reason TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			FOREIGN KEY (recurrence_id) REFERENCES recurring_schedules(id) ON DELETE SET NULL
		);
	`)
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

```bash
cd backend && go test ./db/ -v -run TestNewDB
```

Expected: PASS

- [ ] **Step 6: Write failing test for booking CRUD**

Add to `backend/db/bookings_test.go`:

```go
func TestCreateAndListBookings(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()

	wish := BookingWish{
		Date:            "2026-05-15",
		StartTime:       "14:30",
		DurationMinutes: 90,
		RoomPriorities:  []int{114, 116, 81},
	}

	id, err := database.CreateBooking(wish)
	if err != nil {
		t.Fatalf("CreateBooking error: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty ID")
	}

	bookings, err := database.ListBookings()
	if err != nil {
		t.Fatalf("ListBookings error: %v", err)
	}
	if len(bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(bookings))
	}
	if bookings[0].ID != id {
		t.Fatalf("expected ID %s, got %s", id, bookings[0].ID)
	}
	if bookings[0].Status != "pending" {
		t.Fatalf("expected status pending, got %s", bookings[0].Status)
	}
}

func newTestDB(t *testing.T) *DB {
	t.Helper()
	path := t.TempDir() + "/test.db"
	database, err := New(path)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	return database
}
```

- [ ] **Step 7: Implement booking CRUD**

Create `backend/db/bookings.go`:

```go
package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type BookingWish struct {
	ID              string `json:"id"`
	Date            string `json:"date"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
	RoomPriorities  []int  `json:"room_priorities"`
	RecurrenceID    string `json:"recurrence_id,omitempty"`
	Status          string `json:"status"`
	ResultRoom      string `json:"result_room,omitempty"`
	ResultDuration  *int   `json:"result_duration,omitempty"`
	FailureReason   string `json:"failure_reason,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func (d *DB) CreateBooking(wish BookingWish) (string, error) {
	id := uuid.New().String()
	rooms, _ := json.Marshal(wish.RoomPriorities)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := d.db.Exec(`
		INSERT INTO booking_wishes (id, date, start_time, duration_minutes, room_priorities, recurrence_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'pending', ?, ?)`,
		id, wish.Date, wish.StartTime, wish.DurationMinutes, string(rooms), nilIfEmpty(wish.RecurrenceID), now, now,
	)
	if err != nil {
		return "", fmt.Errorf("insert booking: %w", err)
	}
	return id, nil
}

func (d *DB) ListBookings() ([]BookingWish, error) {
	rows, err := d.db.Query(`
		SELECT id, date, start_time, duration_minutes, room_priorities, 
		       COALESCE(recurrence_id, ''), status, COALESCE(result_room, ''),
		       result_duration, COALESCE(failure_reason, ''), created_at, updated_at
		FROM booking_wishes ORDER BY date, start_time`)
	if err != nil {
		return nil, fmt.Errorf("query bookings: %w", err)
	}
	defer rows.Close()

	var bookings []BookingWish
	for rows.Next() {
		var b BookingWish
		var roomsJSON string
		var resultDur *int
		err := rows.Scan(&b.ID, &b.Date, &b.StartTime, &b.DurationMinutes, &roomsJSON,
			&b.RecurrenceID, &b.Status, &b.ResultRoom, &resultDur, &b.FailureReason, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		json.Unmarshal([]byte(roomsJSON), &b.RoomPriorities)
		b.ResultDuration = resultDur
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func (d *DB) UpdateBookingStatus(id, status, resultRoom string, resultDuration *int, failureReason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := d.db.Exec(`
		UPDATE booking_wishes SET status=?, result_room=?, result_duration=?, failure_reason=?, updated_at=?
		WHERE id=?`, status, nilIfEmpty(resultRoom), resultDuration, nilIfEmpty(failureReason), now, id)
	return err
}

func (d *DB) DeleteBooking(id string) error {
	_, err := d.db.Exec("DELETE FROM booking_wishes WHERE id=?", id)
	return err
}

func (d *DB) GetPendingBookings() ([]BookingWish, error) {
	rows, err := d.db.Query(`
		SELECT id, date, start_time, duration_minutes, room_priorities, 
		       COALESCE(recurrence_id, ''), status, COALESCE(result_room, ''),
		       result_duration, COALESCE(failure_reason, ''), created_at, updated_at
		FROM booking_wishes WHERE status IN ('pending', 'scheduled')
		ORDER BY date, start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookings []BookingWish
	for rows.Next() {
		var b BookingWish
		var roomsJSON string
		var resultDur *int
		err := rows.Scan(&b.ID, &b.Date, &b.StartTime, &b.DurationMinutes, &roomsJSON,
			&b.RecurrenceID, &b.Status, &b.ResultRoom, &resultDur, &b.FailureReason, &b.CreatedAt, &b.UpdatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(roomsJSON), &b.RoomPriorities)
		b.ResultDuration = resultDur
		bookings = append(bookings, b)
	}
	return bookings, nil
}

func nilIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
```

- [ ] **Step 8: Run tests**

```bash
cd backend && go test ./db/ -v
```

Expected: all PASS

- [ ] **Step 9: Write and implement recurrence CRUD**

Create `backend/db/recurrences.go`:

```go
package db

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type RecurringSchedule struct {
	ID              string `json:"id"`
	DayOfWeek       int    `json:"day_of_week"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
	RoomPriorities  []int  `json:"room_priorities"`
	Active          bool   `json:"active"`
	CreatedAt       string `json:"created_at"`
}

func (d *DB) CreateRecurrence(r RecurringSchedule) (string, error) {
	id := uuid.New().String()
	rooms, _ := json.Marshal(r.RoomPriorities)
	now := time.Now().UTC().Format(time.RFC3339)

	_, err := d.db.Exec(`
		INSERT INTO recurring_schedules (id, day_of_week, start_time, duration_minutes, room_priorities, active, created_at)
		VALUES (?, ?, ?, ?, ?, 1, ?)`,
		id, r.DayOfWeek, r.StartTime, r.DurationMinutes, string(rooms), now,
	)
	if err != nil {
		return "", fmt.Errorf("insert recurrence: %w", err)
	}
	return id, nil
}

func (d *DB) ListRecurrences() ([]RecurringSchedule, error) {
	rows, err := d.db.Query(`
		SELECT id, day_of_week, start_time, duration_minutes, room_priorities, active, created_at
		FROM recurring_schedules ORDER BY day_of_week, start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []RecurringSchedule
	for rows.Next() {
		var r RecurringSchedule
		var roomsJSON string
		var active int
		err := rows.Scan(&r.ID, &r.DayOfWeek, &r.StartTime, &r.DurationMinutes, &roomsJSON, &active, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(roomsJSON), &r.RoomPriorities)
		r.Active = active == 1
		schedules = append(schedules, r)
	}
	return schedules, nil
}

func (d *DB) UpdateRecurrenceActive(id string, active bool) error {
	val := 0
	if active {
		val = 1
	}
	_, err := d.db.Exec("UPDATE recurring_schedules SET active=? WHERE id=?", val, id)
	return err
}

func (d *DB) DeleteRecurrence(id string) error {
	_, err := d.db.Exec("DELETE FROM recurring_schedules WHERE id=?", id)
	return err
}

func (d *DB) GetActiveRecurrences() ([]RecurringSchedule, error) {
	rows, err := d.db.Query(`
		SELECT id, day_of_week, start_time, duration_minutes, room_priorities, active, created_at
		FROM recurring_schedules WHERE active=1 ORDER BY day_of_week, start_time`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []RecurringSchedule
	for rows.Next() {
		var r RecurringSchedule
		var roomsJSON string
		var active int
		err := rows.Scan(&r.ID, &r.DayOfWeek, &r.StartTime, &r.DurationMinutes, &roomsJSON, &active, &r.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(roomsJSON), &r.RoomPriorities)
		r.Active = active == 1
		schedules = append(schedules, r)
	}
	return schedules, nil
}
```

Create `backend/db/recurrences_test.go`:

```go
package db

import "testing"

func TestCreateAndListRecurrences(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()

	r := RecurringSchedule{
		DayOfWeek:       1,
		StartTime:       "14:30",
		DurationMinutes: 90,
		RoomPriorities:  []int{114, 116},
	}

	id, err := database.CreateRecurrence(r)
	if err != nil {
		t.Fatalf("CreateRecurrence error: %v", err)
	}

	schedules, err := database.ListRecurrences()
	if err != nil {
		t.Fatalf("ListRecurrences error: %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("expected 1 recurrence, got %d", len(schedules))
	}
	if schedules[0].ID != id {
		t.Fatalf("ID mismatch")
	}
	if !schedules[0].Active {
		t.Fatal("expected active=true")
	}
}

func TestToggleRecurrenceActive(t *testing.T) {
	database := newTestDB(t)
	defer database.Close()

	id, _ := database.CreateRecurrence(RecurringSchedule{
		DayOfWeek:       3,
		StartTime:       "10:00",
		DurationMinutes: 60,
		RoomPriorities:  []int{114},
	})

	database.UpdateRecurrenceActive(id, false)

	schedules, _ := database.ListRecurrences()
	if schedules[0].Active {
		t.Fatal("expected active=false after toggle")
	}
}
```

- [ ] **Step 10: Run all DB tests**

```bash
cd backend && go test ./db/ -v
```

Expected: all PASS

- [ ] **Step 11: Commit**

```bash
git add backend/
git commit -m "feat: add SQLite database layer with booking and recurrence CRUD"
```

---

## Task 3: Asimut HTTP Client

**Files:**
- Create: `backend/asimut/client.go`
- Create: `backend/asimut/client_test.go`

- [ ] **Step 1: Write failing test for login**

Create `backend/asimut/client_test.go`:

```go
package asimut

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogin_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/public/login.php" && r.Method == "POST" {
			r.ParseForm()
			if r.FormValue("authenticate-useraccount") != "test@example.com" {
				t.Errorf("unexpected email: %s", r.FormValue("authenticate-useraccount"))
			}
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "abc123"})
			w.Header().Set("Location", "/public/hfm-freiburg.asimut.net")
			w.WriteHeader(302)
			return
		}
		if r.URL.Path == "/services/v2/heartbeat/me" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"response":{"heartbeat":{"loggedin":true,"me":{"id":965,"name":"Test","surname":"User","booking_horizon":"2026-05-15T10:00:00+02:00"}}}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test@example.com", "pass123")
	err := client.Login()
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if !client.LoggedIn() {
		t.Fatal("expected LoggedIn() to be true")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/public/login.php" {
			w.WriteHeader(200)
			w.Write([]byte("login page again - means failure"))
			return
		}
		if r.URL.Path == "/services/v2/heartbeat/me" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"response":{"heartbeat":{"loggedin":false}}}`))
			return
		}
		w.WriteHeader(404)
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad@example.com", "wrong")
	err := client.Login()
	if err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd backend && go test ./asimut/ -v
```

Expected: compilation error

- [ ] **Step 3: Implement Asimut client**

Create `backend/asimut/client.go`:

```go
package asimut

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    string
	email      string
	password   string
	httpClient *http.Client
	loggedIn   bool
	userInfo   *UserInfo
}

type UserInfo struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Surname        string `json:"surname"`
	Username       string `json:"username"`
	BookingHorizon string `json:"booking_horizon"`
}

type Location struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	SecondaryName string `json:"secondary_name"`
	Bookable      bool   `json:"bookable"`
	Type          string `json:"type"`
}

type BookingResult struct {
	EventID int
	Success bool
	Message string
}

func NewClient(baseURL, email, password string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		email:    email,
		password: password,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *Client) LoggedIn() bool {
	return c.loggedIn
}

func (c *Client) Login() error {
	form := url.Values{
		"authenticate-url":          {"%2Fpublic%2Fhfm-freiburg.asimut.net"},
		"authenticate-useraccount":  {c.email},
		"authenticate-password":     {c.password},
		"authenticate-verification": {"ok"},
	}

	resp, err := c.httpClient.PostForm(c.baseURL+"/public/login.php", form)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	resp.Body.Close()

	heartbeat, err := c.getHeartbeat()
	if err != nil {
		return fmt.Errorf("heartbeat after login: %w", err)
	}
	if !heartbeat {
		return fmt.Errorf("login failed: not authenticated after login")
	}

	c.loggedIn = true
	return nil
}

func (c *Client) getHeartbeat() (bool, error) {
	resp, err := c.doJSON("GET", "/services/v2/heartbeat/me", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			Heartbeat struct {
				LoggedIn bool     `json:"loggedin"`
				Me       UserInfo `json:"me"`
			} `json:"heartbeat"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	if result.Response.Heartbeat.LoggedIn {
		c.userInfo = &result.Response.Heartbeat.Me
	}
	return result.Response.Heartbeat.LoggedIn, nil
}

func (c *Client) GetLocations() ([]Location, error) {
	resp, err := c.doJSON("GET", "/services/v2/locations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			Locations []Location `json:"locations"`
		} `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Response.Locations, nil
}

func (c *Client) BookRoom(roomID int, start, end time.Time) (*BookingResult, error) {
	eventDefault, err := c.getEventDefault(roomID, start)
	if err != nil {
		return nil, fmt.Errorf("get event default: %w", err)
	}

	eventDefault["en"] = end.Format("2006-01-02T15:04:05.000-07:00")

	payload := map[string]interface{}{
		"event":          eventDefault,
		"booking_type":   "single",
		"time_period_id": 0,
		"weekdays":       []int{int(start.Weekday())},
	}

	checkResp, err := c.doJSONBody("POST", "/services/v2/event/type=check", payload)
	if err != nil {
		return nil, fmt.Errorf("check booking: %w", err)
	}
	checkResp.Body.Close()

	saveResp, err := c.doJSONBody("POST", "/services/v2/event/type=save", payload)
	if err != nil {
		return nil, fmt.Errorf("save booking: %w", err)
	}
	defer saveResp.Body.Close()

	var saveResult struct {
		Response struct {
			EventIDs []int `json:"event_ids"`
			Success  bool  `json:"success"`
		} `json:"response"`
	}
	if err := json.NewDecoder(saveResp.Body).Decode(&saveResult); err != nil {
		return nil, fmt.Errorf("decode save response: %w", err)
	}

	result := &BookingResult{Success: saveResult.Response.Success}
	if len(saveResult.Response.EventIDs) > 0 {
		result.EventID = saveResult.Response.EventIDs[0]
	}
	return result, nil
}

func (c *Client) ExtendBooking(eventID int, newEnd time.Time) (*BookingResult, error) {
	eventResp, err := c.doJSON("GET", fmt.Sprintf("/services/v2/event/event_id=%d", eventID), nil)
	if err != nil {
		return nil, fmt.Errorf("get event: %w", err)
	}
	defer eventResp.Body.Close()

	var eventResult struct {
		Response struct {
			Event map[string]interface{} `json:"event"`
		} `json:"response"`
	}
	body, _ := io.ReadAll(eventResp.Body)
	if err := json.Unmarshal(body, &eventResult); err != nil {
		return nil, fmt.Errorf("decode event: %w", err)
	}

	event := eventResult.Response.Event
	event["en"] = newEnd.Format("2006-01-02T15:04:05.000-07:00")

	payload := map[string]interface{}{
		"event":          event,
		"booking_type":   "single",
		"time_period_id": 0,
		"weekdays":       []int{1},
	}

	path := fmt.Sprintf("/services/v2/event/event_id=%d;type=check", eventID)
	checkResp, err := c.doJSONBodyMethod("PATCH", path, payload)
	if err != nil {
		return nil, fmt.Errorf("check extension: %w", err)
	}
	checkResp.Body.Close()

	savePath := fmt.Sprintf("/services/v2/event/event_id=%d;type=save", eventID)
	saveResp, err := c.doJSONBodyMethod("PATCH", savePath, payload)
	if err != nil {
		return nil, fmt.Errorf("save extension: %w", err)
	}
	defer saveResp.Body.Close()

	var saveResult struct {
		Response struct {
			EventIDs []int `json:"event_ids"`
			Success  bool  `json:"success"`
		} `json:"response"`
	}
	if err := json.NewDecoder(saveResp.Body).Decode(&saveResult); err != nil {
		return nil, fmt.Errorf("decode extension response: %w", err)
	}

	return &BookingResult{
		EventID: eventID,
		Success: saveResult.Response.Success,
	}, nil
}

func (c *Client) getEventDefault(roomID int, start time.Time) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"st": start.Format("2006-01-02T15:04:05.000-07:00"),
		"ca": 1,
		"rs": []map[string]interface{}{{"id": roomID}},
	}

	resp, err := c.doJSONBody("POST", "/services/v2/eventdefault", payload)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Response struct {
			EventDefault struct {
				Events []map[string]interface{} `json:"events"`
			} `json:"eventdefault"`
		} `json:"response"`
	}

	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode eventdefault: %w", err)
	}

	if len(result.Response.EventDefault.Events) == 0 {
		return nil, fmt.Errorf("no events in eventdefault response")
	}
	return result.Response.EventDefault.Events[0], nil
}

func (c *Client) doJSON(method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", c.baseURL)
	return c.httpClient.Do(req)
}

func (c *Client) doJSONBody(method, path string, payload interface{}) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, c.baseURL+path, strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", c.baseURL)
	return c.httpClient.Do(req)
}

func (c *Client) doJSONBodyMethod(method, path string, payload interface{}) (*http.Response, error) {
	return c.doJSONBody(method, path, payload)
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./asimut/ -v
```

Expected: all PASS

- [ ] **Step 5: Add booking test with mock server**

Add to `backend/asimut/client_test.go`:

```go
func TestBookRoom_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/public/login.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "abc123"})
			w.Header().Set("Location", "/public/hfm-freiburg.asimut.net")
			w.WriteHeader(302)
		case r.URL.Path == "/services/v2/heartbeat/me":
			w.Write([]byte(`{"response":{"heartbeat":{"loggedin":true,"me":{"id":965,"name":"Test","surname":"User"}}}}`))
		case r.URL.Path == "/services/v2/eventdefault":
			w.Write([]byte(`{"response":{"eventdefault":{"events":[{"id":0,"ar":"Einzelüben","ca":1,"st":"2026-05-15T14:30:00+02:00","en":"2026-05-15T15:30:00+02:00","rs":[{"id":114,"dn":"MBP-326"}],"pe":[{"id":965,"ro":1,"dn":"Test User"}],"ps":[{"me":false,"ri":1,"rs":"Teilnehmer*in","rh":"Teilnehmer*in","rc":1,"bo":[{"id":965,"fn":"Test","ln":"User","un":"test"}]}],"ri":{"e":true,"c":false,"r":true,"p":true,"a":false},"vi":"visible","cl":[]}]}}}`))
		case r.URL.Path == "/services/v2/event/type=check":
			w.Write([]byte(`{"response":{"success":true,"event_ids":[0],"bookingrules":{"issues":[]}}}`))
		case r.URL.Path == "/services/v2/event/type=save":
			w.Write([]byte(`{"response":{"success":true,"event_ids":[470262]}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "test@example.com", "pass")
	client.Login()

	start := time.Date(2026, 5, 15, 14, 30, 0, 0, time.FixedZone("CEST", 2*3600))
	end := time.Date(2026, 5, 15, 15, 0, 0, 0, time.FixedZone("CEST", 2*3600))

	result, err := client.BookRoom(114, start, end)
	if err != nil {
		t.Fatalf("BookRoom error: %v", err)
	}
	if !result.Success {
		t.Fatal("expected success")
	}
	if result.EventID != 470262 {
		t.Fatalf("expected event ID 470262, got %d", result.EventID)
	}
}
```

- [ ] **Step 6: Run tests**

```bash
cd backend && go test ./asimut/ -v
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add backend/
git commit -m "feat: add Asimut HTTP client with login, booking, and extension"
```

---

## Task 4: Scheduler

**Files:**
- Create: `backend/scheduler/scheduler.go`
- Create: `backend/scheduler/scheduler_test.go`

- [ ] **Step 1: Write failing test for trigger time calculation**

Create `backend/scheduler/scheduler_test.go`:

```go
package scheduler

import (
	"testing"
	"time"
)

func TestCalculateTriggerTime(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		name     string
		date     string
		start    string
		expected time.Time
	}{
		{
			name:     "14:30 slot on Wednesday triggers Monday 15:00",
			date:     "2026-05-13",
			start:    "14:30",
			expected: time.Date(2026, 5, 11, 15, 0, 0, 0, loc),
		},
		{
			name:     "09:00 slot on Friday triggers Wednesday 09:30",
			date:     "2026-05-15",
			start:    "09:00",
			expected: time.Date(2026, 5, 13, 9, 30, 0, 0, loc),
		},
		{
			name:     "20:15 slot on Tuesday triggers Sunday 20:45",
			date:     "2026-05-12",
			start:    "20:15",
			expected: time.Date(2026, 5, 10, 20, 45, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTriggerTime(tt.date, tt.start, loc)
			if !result.Equal(tt.expected) {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd backend && go test ./scheduler/ -v
```

Expected: compilation error

- [ ] **Step 3: Implement trigger time calculation**

Create `backend/scheduler/scheduler.go`:

```go
package scheduler

import (
	"log"
	"sync"
	"time"
)

type Job struct {
	ID          string
	TriggerTime time.Time
	Execute     func()
}

type Scheduler struct {
	jobs    map[string]*Job
	mu      sync.Mutex
	stop    chan struct{}
	running bool
}

func New() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		stop: make(chan struct{}),
	}
}

// CalculateTriggerTime computes when to fire the booking request.
// Formula: trigger = (slot_date - 2 days) + slot_start_time + 30min
func CalculateTriggerTime(date, startTime string, loc *time.Location) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", date, loc)
	parts := parseTime(startTime)
	slotStart := time.Date(d.Year(), d.Month(), d.Day(), parts[0], parts[1], 0, 0, loc)
	return slotStart.Add(-48*time.Hour + 30*time.Minute)
}

func parseTime(s string) [2]int {
	var h, m int
	for i, c := range s {
		if c == ':' {
			h = atoi(s[:i])
			m = atoi(s[i+1:])
			break
		}
	}
	return [2]int{h, m}
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

func (s *Scheduler) Schedule(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	log.Printf("Scheduled job %s for %v", job.ID, job.TriggerTime)
}

func (s *Scheduler) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
}

func (s *Scheduler) Start() {
	s.running = true
	go s.loop()
}

func (s *Scheduler) Stop() {
	close(s.stop)
	s.running = false
}

func (s *Scheduler) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			s.mu.Lock()
			var toFire []string
			for id, job := range s.jobs {
				if now.After(job.TriggerTime) || now.Equal(job.TriggerTime) {
					toFire = append(toFire, id)
				}
			}
			for _, id := range toFire {
				job := s.jobs[id]
				delete(s.jobs, id)
				go job.Execute()
				log.Printf("Fired job %s", id)
			}
			s.mu.Unlock()
		}
	}
}
```

- [ ] **Step 4: Run tests**

```bash
cd backend && go test ./scheduler/ -v
```

Expected: PASS

- [ ] **Step 5: Add test for scheduler execution**

Add to `backend/scheduler/scheduler_test.go`:

```go
func TestScheduler_FiresJobAtTriggerTime(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	fired := make(chan bool, 1)
	job := &Job{
		ID:          "test-1",
		TriggerTime: time.Now().Add(200 * time.Millisecond),
		Execute: func() {
			fired <- true
		},
	}
	s.Schedule(job)

	select {
	case <-fired:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("job did not fire within timeout")
	}
}

func TestScheduler_DoesNotFireEarly(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	fired := make(chan bool, 1)
	job := &Job{
		ID:          "test-2",
		TriggerTime: time.Now().Add(5 * time.Second),
		Execute: func() {
			fired <- true
		},
	}
	s.Schedule(job)

	select {
	case <-fired:
		t.Fatal("job fired too early")
	case <-time.After(500 * time.Millisecond):
		// correct - not fired yet
	}
}
```

- [ ] **Step 6: Run tests**

```bash
cd backend && go test ./scheduler/ -v -timeout 10s
```

Expected: all PASS

- [ ] **Step 7: Commit**

```bash
git add backend/
git commit -m "feat: add scheduler with precise trigger time calculation"
```

---

## Task 5: REST API

**Files:**
- Create: `backend/api/router.go`
- Create: `backend/api/auth.go`
- Create: `backend/api/auth_test.go`
- Create: `backend/api/bookings.go`
- Create: `backend/api/recurrences.go`
- Create: `backend/api/rooms.go`
- Create: `backend/api/settings.go`

- [ ] **Step 1: Add HTTP router dependency**

```bash
cd backend && go get github.com/go-chi/chi/v5
```

- [ ] **Step 2: Write failing test for auth middleware**

Create `backend/api/auth_test.go`:

```go
package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthMiddleware_RejectsNoPassword(t *testing.T) {
	handler := AuthMiddleware("secret123")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/bookings", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthMiddleware_AcceptsValidPassword(t *testing.T) {
	handler := AuthMiddleware("secret123")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))

	req := httptest.NewRequest("GET", "/api/bookings", nil)
	req.Header.Set("Authorization", "Bearer secret123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 3: Implement auth middleware**

Create `backend/api/auth.go`:

```go
package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func AuthMiddleware(password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			token := strings.TrimPrefix(auth, "Bearer ")

			if subtle.ConstantTimeCompare([]byte(token), []byte(password)) != 1 {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

- [ ] **Step 4: Run auth tests**

```bash
cd backend && go test ./api/ -v -run TestAuth
```

Expected: PASS

- [ ] **Step 5: Implement router and handlers**

Create `backend/api/router.go`:

```go
package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/philippgehrig/asimuth-automation/backend/asimut"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

type Server struct {
	db        *db.DB
	asimut    *asimut.Client
	scheduler *scheduler.Scheduler
	password  string
}

func NewServer(database *db.DB, asimutClient *asimut.Client, sched *scheduler.Scheduler, password string) *Server {
	return &Server{
		db:        database,
		asimut:    asimutClient,
		scheduler: sched,
		password:  password,
	}
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Use(AuthMiddleware(s.password))
		r.Get("/bookings", s.listBookings)
		r.Post("/bookings", s.createBooking)
		r.Delete("/bookings/{id}", s.deleteBooking)
		r.Get("/recurrences", s.listRecurrences)
		r.Post("/recurrences", s.createRecurrence)
		r.Patch("/recurrences/{id}", s.updateRecurrence)
		r.Delete("/recurrences/{id}", s.deleteRecurrence)
		r.Get("/rooms", s.listRooms)
		r.Get("/settings/status", s.getStatus)
	})

	return r
}
```

Create `backend/api/bookings.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philippgehrig/asimuth-automation/backend/db"
)

func (s *Server) listBookings(w http.ResponseWriter, r *http.Request) {
	bookings, err := s.db.ListBookings()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, bookings)
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var wish db.BookingWish
	if err := json.NewDecoder(r.Body).Decode(&wish); err != nil {
		http.Error(w, "invalid JSON", 400)
		return
	}

	id, err := s.db.CreateBooking(wish)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	s.scheduleBookingJob(id, wish)

	writeJSON(w, map[string]string{"id": id})
}

func (s *Server) deleteBooking(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	s.scheduler.Cancel(id)
	if err := s.db.DeleteBooking(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
```

Create `backend/api/recurrences.go`:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/philippgehrig/asimuth-automation/backend/db"
)

func (s *Server) listRecurrences(w http.ResponseWriter, r *http.Request) {
	recurrences, err := s.db.ListRecurrences()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, recurrences)
}

func (s *Server) createRecurrence(w http.ResponseWriter, r *http.Request) {
	var rec db.RecurringSchedule
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		http.Error(w, "invalid JSON", 400)
		return
	}

	id, err := s.db.CreateRecurrence(rec)
	if err != nil {
		http.Error(w, err.Error(), 500)
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
		http.Error(w, "invalid JSON", 400)
		return
	}
	if body.Active != nil {
		if err := s.db.UpdateRecurrenceActive(id, *body.Active); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
	w.WriteHeader(204)
}

func (s *Server) deleteRecurrence(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.db.DeleteRecurrence(id); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.WriteHeader(204)
}
```

Create `backend/api/rooms.go`:

```go
package api

import "net/http"

func (s *Server) listRooms(w http.ResponseWriter, r *http.Request) {
	locations, err := s.asimut.GetLocations()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, locations)
}
```

Create `backend/api/settings.go`:

```go
package api

import "net/http"

func (s *Server) getStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]interface{}{
		"asimut_connected": s.asimut.LoggedIn(),
	})
}
```

- [ ] **Step 6: Add booking job scheduling logic**

Add to `backend/api/bookings.go`:

```go
func (s *Server) scheduleBookingJob(id string, wish db.BookingWish) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	triggerTime := scheduler.CalculateTriggerTime(wish.Date, wish.StartTime, loc)

	if triggerTime.Before(time.Now()) {
		return
	}

	s.db.UpdateBookingStatus(id, "scheduled", "", nil, "")

	job := &scheduler.Job{
		ID:          id,
		TriggerTime: triggerTime,
		Execute: func() {
			s.executeBooking(id, wish)
		},
	}
	s.scheduler.Schedule(job)
}

func (s *Server) executeBooking(id string, wish db.BookingWish) {
	loc, _ := time.LoadLocation("Europe/Berlin")
	date, _ := time.ParseInLocation("2006-01-02", wish.Date, loc)
	parts := scheduler.ParseTime(wish.StartTime)
	start := time.Date(date.Year(), date.Month(), date.Day(), parts[0], parts[1], 0, 0, loc)
	end := start.Add(30 * time.Minute)

	if err := s.asimut.Login(); err != nil {
		s.db.UpdateBookingStatus(id, "failed", "", nil, "login failed: "+err.Error())
		return
	}

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
		s.db.UpdateBookingStatus(id, "failed", "", nil, "all rooms taken")
		return
	}

	totalBooked := 30
	extensionsNeeded := (wish.DurationMinutes - 30) / 15
	currentEnd := end

	for i := 0; i < extensionsNeeded; i++ {
		newEnd := currentEnd.Add(15 * time.Minute)
		result, err := s.asimut.ExtendBooking(eventID, newEnd)
		if err != nil || !result.Success {
			dur := totalBooked
			s.db.UpdateBookingStatus(id, "partially_booked", bookedRoom, &dur, "extension failed")
			return
		}
		currentEnd = newEnd
		totalBooked += 15
	}

	s.db.UpdateBookingStatus(id, "booked", bookedRoom, &totalBooked, "")
}
```

Add these imports to the top of `backend/api/bookings.go`:

```go
import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)
```

- [ ] **Step 7: Export ParseTime from scheduler**

Update the function in `backend/scheduler/scheduler.go` — rename `parseTime` to `ParseTime` (export it):

```go
func ParseTime(s string) [2]int {
	var h, m int
	for i, c := range s {
		if c == ':' {
			h = atoi(s[:i])
			m = atoi(s[i+1:])
			break
		}
	}
	return [2]int{h, m}
}
```

Update `CalculateTriggerTime` to use `ParseTime` instead of `parseTime`.

- [ ] **Step 8: Run all tests**

```bash
cd backend && go test ./... -v
```

Expected: all PASS

- [ ] **Step 9: Commit**

```bash
git add backend/
git commit -m "feat: add REST API with auth, bookings, recurrences, rooms, and settings"
```

---

## Task 6: Wire Up Main + Recurrence Generator

**Files:**
- Modify: `backend/main.go`

- [ ] **Step 1: Complete main.go**

Replace `backend/main.go`:

```go
package main

import (
	"log"
	"net/http"
	"time"

	"github.com/philippgehrig/asimuth-automation/backend/api"
	"github.com/philippgehrig/asimuth-automation/backend/asimut"
	"github.com/philippgehrig/asimuth-automation/backend/config"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

func main() {
	cfg := config.Load()

	database, err := db.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to init DB: %v", err)
	}
	defer database.Close()

	asimutClient := asimut.NewClient("https://hfm-freiburg.asimut.net", cfg.AsimutEmail, cfg.AsimutPassword)

	sched := scheduler.New()
	sched.Start()

	srv := api.NewServer(database, asimutClient, sched, cfg.AppPassword)

	rescheduleExisting(database, srv)

	go generateRecurrences(database, srv)

	log.Printf("Starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func rescheduleExisting(database *db.DB, srv *api.Server) {
	bookings, err := database.GetPendingBookings()
	if err != nil {
		log.Printf("Error loading pending bookings: %v", err)
		return
	}
	for _, b := range bookings {
		srv.ScheduleBookingJob(b.ID, b)
	}
	log.Printf("Rescheduled %d pending bookings", len(bookings))
}

func generateRecurrences(database *db.DB, srv *api.Server) {
	for {
		recurrences, err := database.GetActiveRecurrences()
		if err != nil {
			log.Printf("Error loading recurrences: %v", err)
			time.Sleep(1 * time.Hour)
			continue
		}

		loc, _ := time.LoadLocation("Europe/Berlin")
		now := time.Now().In(loc)

		for _, r := range recurrences {
			for weeksAhead := 0; weeksAhead < 4; weeksAhead++ {
				target := nextWeekday(now, time.Weekday((r.DayOfWeek+1)%7), weeksAhead)
				dateStr := target.Format("2006-01-02")

				existing, _ := database.GetBookingByRecurrenceAndDate(r.ID, dateStr)
				if existing != nil {
					continue
				}

				wish := db.BookingWish{
					Date:            dateStr,
					StartTime:       r.StartTime,
					DurationMinutes: r.DurationMinutes,
					RoomPriorities:  r.RoomPriorities,
					RecurrenceID:    r.ID,
				}
				id, err := database.CreateBooking(wish)
				if err != nil {
					log.Printf("Error creating recurrence booking: %v", err)
					continue
				}
				srv.ScheduleBookingJob(id, wish)
			}
		}

		time.Sleep(1 * time.Hour)
	}
}

func nextWeekday(from time.Time, weekday time.Weekday, weeksAhead int) time.Time {
	daysUntil := int(weekday) - int(from.Weekday())
	if daysUntil <= 0 {
		daysUntil += 7
	}
	return from.AddDate(0, 0, daysUntil+weeksAhead*7)
}
```

- [ ] **Step 2: Export ScheduleBookingJob from Server**

Rename `scheduleBookingJob` to `ScheduleBookingJob` in `backend/api/bookings.go`.

- [ ] **Step 3: Add GetBookingByRecurrenceAndDate to DB**

Add to `backend/db/bookings.go`:

```go
func (d *DB) GetBookingByRecurrenceAndDate(recurrenceID, date string) (*BookingWish, error) {
	var b BookingWish
	var roomsJSON string
	var resultDur *int
	err := d.db.QueryRow(`
		SELECT id, date, start_time, duration_minutes, room_priorities,
		       COALESCE(recurrence_id, ''), status, COALESCE(result_room, ''),
		       result_duration, COALESCE(failure_reason, ''), created_at, updated_at
		FROM booking_wishes WHERE recurrence_id=? AND date=?`, recurrenceID, date).
		Scan(&b.ID, &b.Date, &b.StartTime, &b.DurationMinutes, &roomsJSON,
			&b.RecurrenceID, &b.Status, &b.ResultRoom, &resultDur, &b.FailureReason, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(roomsJSON), &b.RoomPriorities)
	b.ResultDuration = resultDur
	return &b, nil
}
```

- [ ] **Step 4: Verify compilation**

```bash
cd backend && go build ./...
```

Expected: no errors

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat: wire up main with scheduler, recurrence generation, and server start"
```

---

## Task 7: Backend Dockerfile

**Files:**
- Create: `backend/Dockerfile`

- [ ] **Step 1: Create Dockerfile**

Create `backend/Dockerfile`:

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /asimut-bot .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /asimut-bot /asimut-bot
RUN mkdir -p /data
EXPOSE 8080
CMD ["/asimut-bot"]
```

- [ ] **Step 2: Commit**

```bash
git add backend/Dockerfile
git commit -m "feat: add backend Dockerfile"
```

---

## Task 8: Vue Frontend Scaffold

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Create: `frontend/index.html`
- Create: `frontend/src/main.ts`
- Create: `frontend/src/App.vue`
- Create: `frontend/src/router.ts`
- Create: `frontend/src/api.ts`

- [ ] **Step 1: Initialize Vue project**

```bash
mkdir -p frontend/src && cd frontend && yarn init -y
```

- [ ] **Step 2: Add dependencies**

```bash
cd frontend && yarn add vue vue-router@4 pinia && yarn add -D vite @vitejs/plugin-vue typescript vue-tsc tailwindcss postcss autoprefixer @types/node
```

- [ ] **Step 3: Create vite.config.ts**

Create `frontend/vite.config.ts`:

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
```

- [ ] **Step 4: Create tsconfig.json**

Create `frontend/tsconfig.json`:

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "jsx": "preserve",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "esModuleInterop": true,
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "skipLibCheck": true,
    "noEmit": true,
    "paths": {
      "@/*": ["./src/*"]
    },
    "baseUrl": "."
  },
  "include": ["src/**/*.ts", "src/**/*.vue"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

Create `frontend/tsconfig.node.json`:

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "allowSyntheticDefaultImports": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: Create Tailwind config**

```bash
cd frontend && npx tailwindcss init -p
```

Update `frontend/tailwind.config.js`:

```javascript
/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  theme: {
    extend: {},
  },
  plugins: [],
}
```

- [ ] **Step 6: Create index.html and entry files**

Create `frontend/index.html`:

```html
<!DOCTYPE html>
<html lang="de">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>Asimut Booking Bot</title>
</head>
<body>
  <div id="app"></div>
  <script type="module" src="/src/main.ts"></script>
</body>
</html>
```

Create `frontend/src/main.ts`:

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')
```

Create `frontend/src/style.css`:

```css
@tailwind base;
@tailwind components;
@tailwind utilities;
```

Create `frontend/src/App.vue`:

```vue
<template>
  <div class="min-h-screen bg-gray-50">
    <router-view v-if="isAuthenticated" />
    <LoginView v-else />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from './stores/auth'
import LoginView from './views/LoginView.vue'

const auth = useAuthStore()
const isAuthenticated = computed(() => auth.isAuthenticated)
</script>
```

- [ ] **Step 7: Create API client**

Create `frontend/src/api.ts`:

```typescript
const API_BASE = '/api'

function getToken(): string {
  return localStorage.getItem('app_token') || ''
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const resp = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${getToken()}`,
      ...options.headers,
    },
  })

  if (resp.status === 401) {
    localStorage.removeItem('app_token')
    window.location.reload()
    throw new Error('Unauthorized')
  }

  if (!resp.ok) {
    throw new Error(`API error: ${resp.status}`)
  }

  if (resp.status === 204) return undefined as T
  return resp.json()
}

export const api = {
  getBookings: () => request<any[]>('/bookings'),
  createBooking: (data: any) => request<{ id: string }>('/bookings', { method: 'POST', body: JSON.stringify(data) }),
  deleteBooking: (id: string) => request<void>(`/bookings/${id}`, { method: 'DELETE' }),
  getRecurrences: () => request<any[]>('/recurrences'),
  createRecurrence: (data: any) => request<{ id: string }>('/recurrences', { method: 'POST', body: JSON.stringify(data) }),
  updateRecurrence: (id: string, data: any) => request<void>(`/recurrences/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
  deleteRecurrence: (id: string) => request<void>(`/recurrences/${id}`, { method: 'DELETE' }),
  getRooms: () => request<any[]>('/rooms'),
  getStatus: () => request<{ asimut_connected: boolean }>('/settings/status'),
}
```

- [ ] **Step 8: Create router**

Create `frontend/src/router.ts`:

```typescript
import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: () => import('./views/DashboardView.vue') },
    { path: '/create', component: () => import('./views/CreateBookingView.vue') },
    { path: '/rooms', component: () => import('./views/RoomsView.vue') },
    { path: '/settings', component: () => import('./views/SettingsView.vue') },
  ],
})

export default router
```

- [ ] **Step 9: Create auth store**

Create `frontend/src/stores/auth.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('app_token') || '')

  const isAuthenticated = computed(() => token.value !== '')

  function login(password: string) {
    token.value = password
    localStorage.setItem('app_token', password)
  }

  function logout() {
    token.value = ''
    localStorage.removeItem('app_token')
  }

  return { token, isAuthenticated, login, logout }
})
```

- [ ] **Step 10: Verify it builds**

```bash
cd frontend && yarn build
```

Expected: successful build (may have warnings about missing views — that's fine, we add them next)

- [ ] **Step 11: Commit**

```bash
git add frontend/
git commit -m "feat: scaffold Vue frontend with router, API client, and auth store"
```

---

## Task 9: Frontend Views

**Files:**
- Create: `frontend/src/views/LoginView.vue`
- Create: `frontend/src/views/DashboardView.vue`
- Create: `frontend/src/views/CreateBookingView.vue`
- Create: `frontend/src/views/RoomsView.vue`
- Create: `frontend/src/views/SettingsView.vue`
- Create: `frontend/src/components/BookingCard.vue`
- Create: `frontend/src/components/StatusBadge.vue`
- Create: `frontend/src/components/RoomPriorityList.vue`
- Create: `frontend/src/stores/bookings.ts`
- Create: `frontend/src/stores/recurrences.ts`
- Create: `frontend/src/stores/rooms.ts`

- [ ] **Step 1: Create stores**

Create `frontend/src/stores/bookings.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export interface BookingWish {
  id: string
  date: string
  start_time: string
  duration_minutes: number
  room_priorities: number[]
  recurrence_id?: string
  status: string
  result_room?: string
  result_duration?: number
  failure_reason?: string
  created_at: string
  updated_at: string
}

export const useBookingsStore = defineStore('bookings', () => {
  const bookings = ref<BookingWish[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    bookings.value = await api.getBookings()
    loading.value = false
  }

  async function create(data: Partial<BookingWish>) {
    await api.createBooking(data)
    await fetch()
  }

  async function remove(id: string) {
    await api.deleteBooking(id)
    await fetch()
  }

  return { bookings, loading, fetch, create, remove }
})
```

Create `frontend/src/stores/recurrences.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export interface RecurringSchedule {
  id: string
  day_of_week: number
  start_time: string
  duration_minutes: number
  room_priorities: number[]
  active: boolean
  created_at: string
}

export const useRecurrencesStore = defineStore('recurrences', () => {
  const recurrences = ref<RecurringSchedule[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    recurrences.value = await api.getRecurrences()
    loading.value = false
  }

  async function create(data: Partial<RecurringSchedule>) {
    await api.createRecurrence(data)
    await fetch()
  }

  async function toggleActive(id: string, active: boolean) {
    await api.updateRecurrence(id, { active })
    await fetch()
  }

  async function remove(id: string) {
    await api.deleteRecurrence(id)
    await fetch()
  }

  return { recurrences, loading, fetch, create, toggleActive, remove }
})
```

Create `frontend/src/stores/rooms.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { api } from '../api'

export interface Room {
  id: number
  name: string
  secondary_name: string
  bookable: boolean
  type: string
}

export const useRoomsStore = defineStore('rooms', () => {
  const rooms = ref<Room[]>([])
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    rooms.value = await api.getRooms()
    loading.value = false
  }

  return { rooms, loading, fetch }
})
```

- [ ] **Step 2: Create LoginView**

Create `frontend/src/views/LoginView.vue`:

```vue
<template>
  <div class="flex items-center justify-center min-h-screen p-4">
    <form @submit.prevent="handleLogin" class="w-full max-w-sm space-y-4">
      <h1 class="text-2xl font-bold text-center">Asimut Booking Bot</h1>
      <input
        v-model="password"
        type="password"
        placeholder="Password"
        class="w-full px-4 py-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
        autofocus
      />
      <button
        type="submit"
        class="w-full py-3 text-white bg-blue-600 rounded-lg hover:bg-blue-700"
      >
        Login
      </button>
      <p v-if="error" class="text-sm text-red-500 text-center">{{ error }}</p>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { api } from '../api'

const auth = useAuthStore()
const password = ref('')
const error = ref('')

async function handleLogin() {
  auth.login(password.value)
  try {
    await api.getStatus()
  } catch {
    auth.logout()
    error.value = 'Invalid password'
  }
}
</script>
```

- [ ] **Step 3: Create StatusBadge component**

Create `frontend/src/components/StatusBadge.vue`:

```vue
<template>
  <span :class="classes" class="px-2 py-1 text-xs font-medium rounded-full">
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ status: string }>()

const label = computed(() => {
  const labels: Record<string, string> = {
    pending: 'Pending',
    scheduled: 'Scheduled',
    booked: 'Booked',
    partially_booked: 'Partial',
    failed: 'Failed',
  }
  return labels[props.status] || props.status
})

const classes = computed(() => {
  const map: Record<string, string> = {
    pending: 'bg-gray-100 text-gray-700',
    scheduled: 'bg-blue-100 text-blue-700',
    booked: 'bg-green-100 text-green-700',
    partially_booked: 'bg-yellow-100 text-yellow-700',
    failed: 'bg-red-100 text-red-700',
  }
  return map[props.status] || 'bg-gray-100 text-gray-700'
})
</script>
```

- [ ] **Step 4: Create BookingCard component**

Create `frontend/src/components/BookingCard.vue`:

```vue
<template>
  <div class="p-4 bg-white rounded-lg shadow-sm border">
    <div class="flex justify-between items-start">
      <div>
        <p class="font-medium">{{ booking.date }} at {{ booking.start_time }}</p>
        <p class="text-sm text-gray-500">{{ booking.duration_minutes }} min</p>
        <p v-if="booking.result_room" class="text-sm text-green-600">
          Room: {{ booking.result_room }}
        </p>
        <p v-if="booking.failure_reason" class="text-sm text-red-500">
          {{ booking.failure_reason }}
        </p>
      </div>
      <div class="flex items-center gap-2">
        <StatusBadge :status="booking.status" />
        <button
          v-if="booking.status === 'pending' || booking.status === 'scheduled'"
          @click="$emit('delete', booking.id)"
          class="text-red-400 hover:text-red-600"
        >
          ✕
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import StatusBadge from './StatusBadge.vue'
import type { BookingWish } from '../stores/bookings'

defineProps<{ booking: BookingWish }>()
defineEmits<{ delete: [id: string] }>()
</script>
```

- [ ] **Step 5: Create RoomPriorityList component**

Create `frontend/src/components/RoomPriorityList.vue`:

```vue
<template>
  <div class="space-y-2">
    <label class="block text-sm font-medium text-gray-700">Room Priority</label>
    <div class="space-y-1">
      <div
        v-for="(roomId, index) in modelValue"
        :key="roomId"
        class="flex items-center gap-2 p-2 bg-white border rounded"
      >
        <span class="text-sm text-gray-400 w-6">{{ index + 1 }}.</span>
        <span class="flex-1 text-sm">{{ getRoomName(roomId) }}</span>
        <button @click="moveUp(index)" :disabled="index === 0" class="text-gray-400 hover:text-gray-600 disabled:opacity-30">↑</button>
        <button @click="moveDown(index)" :disabled="index === modelValue.length - 1" class="text-gray-400 hover:text-gray-600 disabled:opacity-30">↓</button>
        <button @click="remove(index)" class="text-red-400 hover:text-red-600">✕</button>
      </div>
    </div>
    <select @change="addRoom($event)" class="w-full px-3 py-2 border rounded text-sm">
      <option value="">Add room...</option>
      <option v-for="room in availableRooms" :key="room.id" :value="room.id">
        {{ room.name }} - {{ room.secondary_name }}
      </option>
    </select>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoomsStore } from '../stores/rooms'

const props = defineProps<{ modelValue: number[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: number[]] }>()

const roomsStore = useRoomsStore()

const availableRooms = computed(() =>
  roomsStore.rooms.filter(r => r.type === 'location' && r.bookable && !props.modelValue.includes(r.id))
)

function getRoomName(id: number): string {
  const room = roomsStore.rooms.find(r => r.id === id)
  return room ? `${room.name} - ${room.secondary_name}` : `Room ${id}`
}

function addRoom(event: Event) {
  const select = event.target as HTMLSelectElement
  const id = parseInt(select.value)
  if (id) {
    emit('update:modelValue', [...props.modelValue, id])
    select.value = ''
  }
}

function remove(index: number) {
  const copy = [...props.modelValue]
  copy.splice(index, 1)
  emit('update:modelValue', copy)
}

function moveUp(index: number) {
  if (index === 0) return
  const copy = [...props.modelValue]
  ;[copy[index - 1], copy[index]] = [copy[index], copy[index - 1]]
  emit('update:modelValue', copy)
}

function moveDown(index: number) {
  if (index === props.modelValue.length - 1) return
  const copy = [...props.modelValue]
  ;[copy[index], copy[index + 1]] = [copy[index + 1], copy[index]]
  emit('update:modelValue', copy)
}
</script>
```

- [ ] **Step 6: Create DashboardView**

Create `frontend/src/views/DashboardView.vue`:

```vue
<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex justify-between items-center">
      <h1 class="text-xl font-bold">Bookings</h1>
      <router-link to="/create" class="px-4 py-2 text-sm text-white bg-blue-600 rounded-lg hover:bg-blue-700">
        + New
      </router-link>
    </div>

    <nav class="flex gap-2 text-sm">
      <router-link to="/rooms" class="text-blue-600 hover:underline">Rooms</router-link>
      <router-link to="/settings" class="text-blue-600 hover:underline">Settings</router-link>
    </nav>

    <div v-if="bookingsStore.loading" class="text-gray-500">Loading...</div>

    <div v-else class="space-y-2">
      <BookingCard
        v-for="booking in bookingsStore.bookings"
        :key="booking.id"
        :booking="booking"
        @delete="bookingsStore.remove($event)"
      />
      <p v-if="bookingsStore.bookings.length === 0" class="text-gray-500 text-center py-8">
        No bookings yet
      </p>
    </div>

    <div v-if="recurrencesStore.recurrences.length > 0" class="pt-4 border-t">
      <h2 class="text-lg font-semibold mb-2">Recurring</h2>
      <div v-for="r in recurrencesStore.recurrences" :key="r.id" class="flex items-center justify-between p-3 bg-white border rounded-lg mb-2">
        <div>
          <p class="text-sm font-medium">{{ dayName(r.day_of_week) }} {{ r.start_time }}</p>
          <p class="text-xs text-gray-500">{{ r.duration_minutes }} min</p>
        </div>
        <div class="flex items-center gap-2">
          <button @click="recurrencesStore.toggleActive(r.id, !r.active)" :class="r.active ? 'bg-green-500' : 'bg-gray-300'" class="w-10 h-5 rounded-full relative">
            <span :class="r.active ? 'translate-x-5' : 'translate-x-0'" class="absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform"></span>
          </button>
          <button @click="recurrencesStore.remove(r.id)" class="text-red-400 hover:text-red-600 text-sm">✕</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useBookingsStore } from '../stores/bookings'
import { useRecurrencesStore } from '../stores/recurrences'
import BookingCard from '../components/BookingCard.vue'

const bookingsStore = useBookingsStore()
const recurrencesStore = useRecurrencesStore()

const dayNames = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
function dayName(d: number) { return dayNames[d] }

onMounted(() => {
  bookingsStore.fetch()
  recurrencesStore.fetch()
})
</script>
```

- [ ] **Step 7: Create CreateBookingView**

Create `frontend/src/views/CreateBookingView.vue`:

```vue
<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">← Back</router-link>
      <h1 class="text-xl font-bold">New Booking</h1>
    </div>

    <form @submit.prevent="submit" class="space-y-4">
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium">Recurring</label>
        <input type="checkbox" v-model="isRecurring" class="rounded" />
      </div>

      <div v-if="!isRecurring">
        <label class="block text-sm font-medium text-gray-700">Date</label>
        <input v-model="form.date" type="date" class="w-full px-3 py-2 border rounded" required />
      </div>

      <div v-else>
        <label class="block text-sm font-medium text-gray-700">Day of Week</label>
        <select v-model.number="form.day_of_week" class="w-full px-3 py-2 border rounded">
          <option v-for="(name, i) in dayNames" :key="i" :value="i">{{ name }}</option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700">Start Time</label>
        <input v-model="form.start_time" type="time" step="900" class="w-full px-3 py-2 border rounded" required />
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-700">Duration (minutes)</label>
        <select v-model.number="form.duration_minutes" class="w-full px-3 py-2 border rounded">
          <option :value="30">30 min</option>
          <option :value="45">45 min</option>
          <option :value="60">1 hour</option>
          <option :value="90">1.5 hours</option>
          <option :value="120">2 hours</option>
          <option :value="150">2.5 hours</option>
          <option :value="180">3 hours</option>
        </select>
      </div>

      <RoomPriorityList v-model="form.room_priorities" />

      <button
        type="submit"
        :disabled="form.room_priorities.length === 0"
        class="w-full py-3 text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50"
      >
        {{ isRecurring ? 'Create Recurring Schedule' : 'Create Booking' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useBookingsStore } from '../stores/bookings'
import { useRecurrencesStore } from '../stores/recurrences'
import { useRoomsStore } from '../stores/rooms'
import RoomPriorityList from '../components/RoomPriorityList.vue'

const router = useRouter()
const bookingsStore = useBookingsStore()
const recurrencesStore = useRecurrencesStore()
const roomsStore = useRoomsStore()

const isRecurring = ref(false)
const dayNames = ['Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday', 'Sunday']

const form = reactive({
  date: '',
  day_of_week: 0,
  start_time: '14:00',
  duration_minutes: 60,
  room_priorities: [] as number[],
})

onMounted(() => {
  roomsStore.fetch()
})

async function submit() {
  if (isRecurring.value) {
    await recurrencesStore.create({
      day_of_week: form.day_of_week,
      start_time: form.start_time,
      duration_minutes: form.duration_minutes,
      room_priorities: form.room_priorities,
    })
  } else {
    await bookingsStore.create({
      date: form.date,
      start_time: form.start_time,
      duration_minutes: form.duration_minutes,
      room_priorities: form.room_priorities,
    })
  }
  router.push('/')
}
</script>
```

- [ ] **Step 8: Create RoomsView**

Create `frontend/src/views/RoomsView.vue`:

```vue
<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">← Back</router-link>
      <h1 class="text-xl font-bold">Rooms</h1>
    </div>

    <input
      v-model="search"
      type="text"
      placeholder="Search rooms..."
      class="w-full px-3 py-2 border rounded"
    />

    <div class="space-y-1">
      <div
        v-for="room in filteredRooms"
        :key="room.id"
        class="p-3 bg-white border rounded text-sm"
      >
        <p class="font-medium">{{ room.name }}</p>
        <p class="text-gray-500">{{ room.secondary_name }}</p>
        <p class="text-xs text-gray-400">ID: {{ room.id }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoomsStore } from '../stores/rooms'

const roomsStore = useRoomsStore()
const search = ref('')

const filteredRooms = computed(() => {
  const q = search.value.toLowerCase()
  return roomsStore.rooms
    .filter(r => r.type === 'location' && r.bookable)
    .filter(r => r.name.toLowerCase().includes(q) || r.secondary_name.toLowerCase().includes(q))
})

onMounted(() => {
  roomsStore.fetch()
})
</script>
```

- [ ] **Step 9: Create SettingsView**

Create `frontend/src/views/SettingsView.vue`:

```vue
<template>
  <div class="max-w-lg mx-auto p-4 space-y-4">
    <div class="flex items-center gap-2">
      <router-link to="/" class="text-gray-500 hover:text-gray-700">← Back</router-link>
      <h1 class="text-xl font-bold">Settings</h1>
    </div>

    <div class="p-4 bg-white rounded-lg border">
      <h2 class="font-medium mb-2">Asimut Connection</h2>
      <p v-if="status === null" class="text-gray-500">Checking...</p>
      <p v-else-if="status" class="text-green-600">Connected</p>
      <p v-else class="text-red-500">Not connected</p>
    </div>

    <button @click="logout" class="w-full py-2 text-red-600 border border-red-200 rounded-lg hover:bg-red-50">
      Logout
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { api } from '../api'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const status = ref<boolean | null>(null)

onMounted(async () => {
  try {
    const result = await api.getStatus()
    status.value = result.asimut_connected
  } catch {
    status.value = false
  }
})

function logout() {
  auth.logout()
  window.location.reload()
}
</script>
```

- [ ] **Step 10: Verify build**

```bash
cd frontend && yarn build
```

Expected: successful build

- [ ] **Step 11: Commit**

```bash
git add frontend/
git commit -m "feat: add frontend views - dashboard, create booking, rooms, settings"
```

---

## Task 10: Frontend Dockerfile + Docker Compose

**Files:**
- Create: `frontend/Dockerfile`
- Create: `docker-compose.yml`
- Create: `.env.example`

- [ ] **Step 1: Create frontend Dockerfile**

Create `frontend/Dockerfile`:

```dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json yarn.lock ./
RUN yarn install --frozen-lockfile
COPY . .
RUN yarn build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 3000
```

Create `frontend/nginx.conf`:

```nginx
server {
    listen 3000;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }
}
```

- [ ] **Step 2: Create docker-compose.yml**

Create `docker-compose.yml`:

```yaml
services:
  backend:
    build: ./backend
    ports:
      - "8080:8080"
    environment:
      - ASIMUT_EMAIL=${ASIMUT_EMAIL}
      - ASIMUT_PASSWORD=${ASIMUT_PASSWORD}
      - APP_PASSWORD=${APP_PASSWORD}
      - DATABASE_PATH=/data/asimut.db
    volumes:
      - db_data:/data
    restart: unless-stopped

  frontend:
    build: ./frontend
    ports:
      - "3000:3000"
    depends_on:
      - backend
    restart: unless-stopped

volumes:
  db_data:
```

- [ ] **Step 3: Create .env.example**

Create `.env.example`:

```
ASIMUT_EMAIL=your.email@mh-freiburg.de
ASIMUT_PASSWORD=your_password
APP_PASSWORD=choose_a_web_ui_password
```

- [ ] **Step 4: Commit**

```bash
git add docker-compose.yml .env.example frontend/Dockerfile frontend/nginx.conf
git commit -m "feat: add Docker Compose setup with frontend and backend services"
```

---

## Task 11: Integration Test — End-to-End Booking Flow

**Files:**
- Create: `backend/integration_test.go`

- [ ] **Step 1: Write integration test**

Create `backend/integration_test.go`:

```go
//go:build integration

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/philippgehrig/asimuth-automation/backend/api"
	"github.com/philippgehrig/asimuth-automation/backend/asimut"
	"github.com/philippgehrig/asimuth-automation/backend/db"
	"github.com/philippgehrig/asimuth-automation/backend/scheduler"
)

func TestFullBookingFlow(t *testing.T) {
	database, err := db.New(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()

	mockAsimut := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/public/login.php":
			http.SetCookie(w, &http.Cookie{Name: "PHPSESSID", Value: "test"})
			w.Header().Set("Location", "/public/hfm-freiburg.asimut.net")
			w.WriteHeader(302)
		case r.URL.Path == "/services/v2/heartbeat/me":
			w.Write([]byte(`{"response":{"heartbeat":{"loggedin":true,"me":{"id":1,"name":"Test","surname":"User"}}}}`))
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

	asimutClient := asimut.NewClient(mockAsimut.URL, "test@test.com", "pass")
	sched := scheduler.New()
	sched.Start()
	defer sched.Stop()

	srv := api.NewServer(database, asimutClient, sched, "testpass")
	router := srv.Router()

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

	// Verify booking was created
	req = httptest.NewRequest("GET", "/api/bookings", nil)
	req.Header.Set("Authorization", "Bearer testpass")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var bookings []db.BookingWish
	json.NewDecoder(w.Body).Decode(&bookings)
	if len(bookings) != 1 {
		t.Fatalf("expected 1 booking, got %d", len(bookings))
	}

	// Verify rooms endpoint
	asimutClient.Login()
	req = httptest.NewRequest("GET", "/api/rooms", nil)
	req.Header.Set("Authorization", "Bearer testpass")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("rooms: expected 200, got %d", w.Code, w.Body.String())
	}

	_ = time.Now()
}
```

- [ ] **Step 2: Run integration test**

```bash
cd backend && go test -tags=integration -v -run TestFullBookingFlow
```

Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add backend/integration_test.go
git commit -m "feat: add integration test for full booking flow"
```

---

## Task 12: Final Wiring + README

**Files:**
- Create: `README.md`

- [ ] **Step 1: Create README**

Create `README.md`:

```markdown
# Asimut Room Booking Bot

Automatically books practice rooms on Asimut (hfm-freiburg.asimut.net) at the exact moment they become available.

## Setup

1. Copy `.env.example` to `.env` and fill in your credentials
2. Run `docker compose up -d`
3. Open `http://localhost:3000` and enter your app password

## Development

### Backend
```bash
cd backend
go run .
```

### Frontend
```bash
cd frontend
yarn install
yarn dev
```

## How it works

1. Create a booking wish via the web UI (date, time, duration, room priority list)
2. The scheduler calculates when the booking window opens (27.5h before the slot)
3. At the exact trigger time, the bot logs into Asimut and books the first available room from your priority list
4. If the initial 30-min slot is booked, it immediately extends in 15-min increments until your desired duration is reached
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: add README with setup and usage instructions"
```

---

## Task 13: Verify Docker Build

- [ ] **Step 1: Build containers**

```bash
docker compose build
```

Expected: both containers build successfully

- [ ] **Step 2: Verify containers start**

```bash
docker compose up -d && sleep 3 && docker compose logs --tail=5
```

Expected: backend shows "Starting server on :8080", frontend nginx starts

- [ ] **Step 3: Commit any fixes and push**

```bash
git push
```
