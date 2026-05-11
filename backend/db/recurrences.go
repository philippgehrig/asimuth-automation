package db

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// RecurringSchedule represents a weekly recurring booking pattern.
type RecurringSchedule struct {
	ID              string `json:"id"`
	DayOfWeek       int    `json:"day_of_week"`
	StartTime       string `json:"start_time"`
	DurationMinutes int    `json:"duration_minutes"`
	RoomPriorities  []int  `json:"room_priorities"`
	Active          bool   `json:"active"`
	CreatedAt       string `json:"created_at"`
}

// CreateRecurrence inserts a new recurring schedule and returns the generated ID.
func (d *DB) CreateRecurrence(r RecurringSchedule) (string, error) {
	r.ID = uuid.New().String()

	priorities, err := json.Marshal(r.RoomPriorities)
	if err != nil {
		return "", fmt.Errorf("marshal room priorities: %w", err)
	}

	// New recurrences default to active.
	active := 1

	_, err = d.conn.Exec(`
		INSERT INTO recurring_schedules (id, day_of_week, start_time, duration_minutes, room_priorities, active)
		VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.DayOfWeek, r.StartTime, r.DurationMinutes, string(priorities), active,
	)
	if err != nil {
		return "", fmt.Errorf("insert recurrence: %w", err)
	}

	return r.ID, nil
}

// ListRecurrences returns all recurring schedules.
func (d *DB) ListRecurrences() ([]RecurringSchedule, error) {
	rows, err := d.conn.Query(`
		SELECT id, day_of_week, start_time, duration_minutes, room_priorities, active, created_at
		FROM recurring_schedules
		ORDER BY day_of_week, start_time`)
	if err != nil {
		return nil, fmt.Errorf("query recurrences: %w", err)
	}
	defer rows.Close()

	return scanRecurrences(rows)
}

// GetActiveRecurrences returns only active recurring schedules.
func (d *DB) GetActiveRecurrences() ([]RecurringSchedule, error) {
	rows, err := d.conn.Query(`
		SELECT id, day_of_week, start_time, duration_minutes, room_priorities, active, created_at
		FROM recurring_schedules
		WHERE active = 1
		ORDER BY day_of_week, start_time`)
	if err != nil {
		return nil, fmt.Errorf("query active recurrences: %w", err)
	}
	defer rows.Close()

	return scanRecurrences(rows)
}

// UpdateRecurrenceActive sets the active flag on a recurring schedule.
func (d *DB) UpdateRecurrenceActive(id string, active bool) error {
	val := 0
	if active {
		val = 1
	}
	_, err := d.conn.Exec(`UPDATE recurring_schedules SET active = ? WHERE id = ?`, val, id)
	if err != nil {
		return fmt.Errorf("update recurrence active: %w", err)
	}
	return nil
}

// DeleteRecurrence removes a recurring schedule by ID.
func (d *DB) DeleteRecurrence(id string) error {
	_, err := d.conn.Exec(`DELETE FROM recurring_schedules WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete recurrence: %w", err)
	}
	return nil
}

// scanRecurrences reads all rows into a slice of RecurringSchedule.
func scanRecurrences(rows interface{ Next() bool; Scan(...interface{}) error; Err() error }) ([]RecurringSchedule, error) {
	var schedules []RecurringSchedule
	for rows.Next() {
		var r RecurringSchedule
		var priorities string
		var active int

		err := rows.Scan(&r.ID, &r.DayOfWeek, &r.StartTime, &r.DurationMinutes, &priorities, &active, &r.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("scan recurrence: %w", err)
		}

		if err := json.Unmarshal([]byte(priorities), &r.RoomPriorities); err != nil {
			return nil, fmt.Errorf("unmarshal room priorities: %w", err)
		}

		r.Active = active == 1
		schedules = append(schedules, r)
	}
	return schedules, rows.Err()
}
