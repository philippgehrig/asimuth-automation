package scheduler

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestCalculateTriggerTime(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	tests := []struct {
		name      string
		date      string
		startTime string
		want      time.Time
	}{
		{
			name:      "14:30 slot on 2026-05-13 triggers 2026-05-11 15:00",
			date:      "2026-05-13",
			startTime: "14:30",
			want:      time.Date(2026, 5, 11, 15, 0, 0, 0, loc),
		},
		{
			name:      "09:00 slot on 2026-05-15 triggers 2026-05-13 09:30",
			date:      "2026-05-15",
			startTime: "09:00",
			want:      time.Date(2026, 5, 13, 9, 30, 0, 0, loc),
		},
		{
			name:      "20:15 slot on 2026-05-12 triggers 2026-05-10 20:45",
			date:      "2026-05-12",
			startTime: "20:15",
			want:      time.Date(2026, 5, 10, 20, 45, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateTriggerTime(tt.date, tt.startTime, loc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("CalculateTriggerTime(%q, %q) = %v, want %v", tt.date, tt.startTime, got, tt.want)
			}
		})
	}
}

func TestCalculateTriggerTime_InvalidInput(t *testing.T) {
	loc, _ := time.LoadLocation("Europe/Berlin")

	tests := []struct {
		name      string
		date      string
		startTime string
	}{
		{"invalid date", "not-a-date", "14:30"},
		{"invalid time format", "2026-05-13", "1430"},
		{"invalid hour", "2026-05-13", "25:00"},
		{"invalid minute", "2026-05-13", "14:61"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CalculateTriggerTime(tt.date, tt.startTime, loc)
			if err == nil {
				t.Errorf("expected error for date=%q startTime=%q, got nil", tt.date, tt.startTime)
			}
		})
	}
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input   string
		want    [2]int
		wantErr bool
	}{
		{"14:30", [2]int{14, 30}, false},
		{"09:00", [2]int{9, 0}, false},
		{"1430", [2]int{}, true},
		{"25:00", [2]int{}, true},
		{"14:61", [2]int{}, true},
		{"abc:def", [2]int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseTime(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("expected error for %q", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for %q: %v", tt.input, err)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseTime(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestScheduler_FiresJobAtTriggerTime(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var fired atomic.Int32
	job := &Job{
		ID:          "test-fire",
		TriggerTime: time.Now().Add(200 * time.Millisecond),
		Execute: func() {
			fired.Store(1)
		},
	}
	s.Schedule(job)

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			t.Fatal("job did not fire within 2s")
			return
		case <-ticker.C:
			if fired.Load() == 1 {
				return
			}
		}
	}
}

func TestScheduler_DoesNotFireEarly(t *testing.T) {
	s := New()
	s.Start()
	defer s.Stop()

	var fired atomic.Int32
	job := &Job{
		ID:          "test-no-early",
		TriggerTime: time.Now().Add(5 * time.Second),
		Execute: func() {
			fired.Store(1)
		},
	}
	s.Schedule(job)

	time.Sleep(500 * time.Millisecond)

	if fired.Load() == 1 {
		t.Fatal("job fired early")
	}
}

func TestScheduler_StopIsIdempotent(t *testing.T) {
	s := New()
	s.Start()
	s.Stop()
	s.Stop() // Should not panic
}
