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
			got := CalculateTriggerTime(tt.date, tt.startTime, loc)
			if !got.Equal(tt.want) {
				t.Errorf("CalculateTriggerTime(%q, %q) = %v, want %v", tt.date, tt.startTime, got, tt.want)
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
