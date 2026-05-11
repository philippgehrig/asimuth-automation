package db

import (
	"testing"
)

func TestCreateAndListRecurrences(t *testing.T) {
	d := tempDB(t)

	// Create two recurrences (Monday and Wednesday).
	id1, err := d.CreateRecurrence(RecurringSchedule{
		DayOfWeek:       1, // Monday
		StartTime:       "09:00",
		DurationMinutes: 60,
		RoomPriorities:  []int{1, 2, 3},
	})
	if err != nil {
		t.Fatalf("create recurrence 1: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected non-empty id")
	}

	id2, err := d.CreateRecurrence(RecurringSchedule{
		DayOfWeek:       3, // Wednesday
		StartTime:       "14:00",
		DurationMinutes: 90,
		RoomPriorities:  []int{4, 5},
	})
	if err != nil {
		t.Fatalf("create recurrence 2: %v", err)
	}
	if id2 == "" {
		t.Fatal("expected non-empty id")
	}

	// List all recurrences.
	recurrences, err := d.ListRecurrences()
	if err != nil {
		t.Fatalf("list recurrences: %v", err)
	}
	if len(recurrences) != 2 {
		t.Fatalf("expected 2 recurrences, got %d", len(recurrences))
	}

	// Verify ordering (by day_of_week).
	if recurrences[0].DayOfWeek != 1 {
		t.Errorf("expected first recurrence on Monday (1), got %d", recurrences[0].DayOfWeek)
	}
	if recurrences[1].DayOfWeek != 3 {
		t.Errorf("expected second recurrence on Wednesday (3), got %d", recurrences[1].DayOfWeek)
	}

	// Verify fields.
	if recurrences[0].DurationMinutes != 60 {
		t.Errorf("expected 60 minutes, got %d", recurrences[0].DurationMinutes)
	}
	if len(recurrences[1].RoomPriorities) != 2 || recurrences[1].RoomPriorities[0] != 4 {
		t.Errorf("unexpected room priorities: %v", recurrences[1].RoomPriorities)
	}

	// All should be active by default.
	if !recurrences[0].Active || !recurrences[1].Active {
		t.Error("expected recurrences to be active by default")
	}

	// Get active recurrences (should be all).
	active, err := d.GetActiveRecurrences()
	if err != nil {
		t.Fatalf("get active recurrences: %v", err)
	}
	if len(active) != 2 {
		t.Fatalf("expected 2 active recurrences, got %d", len(active))
	}

	// Delete one.
	err = d.DeleteRecurrence(id2)
	if err != nil {
		t.Fatalf("delete recurrence: %v", err)
	}

	recurrences, err = d.ListRecurrences()
	if err != nil {
		t.Fatalf("list recurrences after delete: %v", err)
	}
	if len(recurrences) != 1 {
		t.Fatalf("expected 1 recurrence after delete, got %d", len(recurrences))
	}
}

func TestToggleRecurrenceActive(t *testing.T) {
	d := tempDB(t)

	id, err := d.CreateRecurrence(RecurringSchedule{
		DayOfWeek:       2, // Tuesday
		StartTime:       "10:00",
		DurationMinutes: 45,
		RoomPriorities:  []int{1},
	})
	if err != nil {
		t.Fatalf("create recurrence: %v", err)
	}

	// Initially active.
	active, err := d.GetActiveRecurrences()
	if err != nil {
		t.Fatalf("get active: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active recurrence, got %d", len(active))
	}

	// Deactivate.
	err = d.UpdateRecurrenceActive(id, false)
	if err != nil {
		t.Fatalf("deactivate recurrence: %v", err)
	}

	active, err = d.GetActiveRecurrences()
	if err != nil {
		t.Fatalf("get active after deactivate: %v", err)
	}
	if len(active) != 0 {
		t.Fatalf("expected 0 active recurrences, got %d", len(active))
	}

	// Verify it still appears in full list.
	all, err := d.ListRecurrences()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 total recurrence, got %d", len(all))
	}
	if all[0].Active {
		t.Error("expected recurrence to be inactive")
	}

	// Reactivate.
	err = d.UpdateRecurrenceActive(id, true)
	if err != nil {
		t.Fatalf("reactivate recurrence: %v", err)
	}

	active, err = d.GetActiveRecurrences()
	if err != nil {
		t.Fatalf("get active after reactivate: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active recurrence after reactivation, got %d", len(active))
	}
}
