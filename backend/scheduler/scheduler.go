package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Job represents a scheduled booking action.
type Job struct {
	ID          string
	TriggerTime time.Time
	Execute     func()
}

// Scheduler manages scheduled jobs and fires them at their trigger times.
type Scheduler struct {
	jobs     map[string]*Job
	mu       sync.Mutex
	stop     chan struct{}
	stopOnce sync.Once
	running  bool
}

// New creates a new Scheduler instance.
func New() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		stop: make(chan struct{}),
	}
}

// ParseTime parses a "HH:MM" string into [hour, minute].
func ParseTime(s string) ([2]int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return [2]int{}, fmt.Errorf("invalid time format %q: expected HH:MM", s)
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil {
		return [2]int{}, fmt.Errorf("invalid hour in %q: %w", s, err)
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil {
		return [2]int{}, fmt.Errorf("invalid minute in %q: %w", s, err)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return [2]int{}, fmt.Errorf("time out of range: %q", s)
	}
	return [2]int{hour, minute}, nil
}

// CalculateTriggerTime computes when to fire the booking request.
// Rule: at time T, you can book the slot starting at T-30min two days from now.
// Equivalently: trigger = slot_start - 48h + 30min = slot_start - 47h30m.
// Note: the spec's "27.5h" was a documentation error — the real advance window is 47h30m.
func CalculateTriggerTime(date, startTime string, loc *time.Location) (time.Time, error) {
	slotDate, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: %w", date, err)
	}

	hm, err := ParseTime(startTime)
	if err != nil {
		return time.Time{}, err
	}

	trigger := slotDate.AddDate(0, 0, -2)
	trigger = time.Date(trigger.Year(), trigger.Month(), trigger.Day(), hm[0], hm[1], 0, 0, loc)
	trigger = trigger.Add(30 * time.Minute)

	return trigger, nil
}

// Schedule adds a job to the scheduler (mutex protected).
func (s *Scheduler) Schedule(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Cancel removes a job from the scheduler.
func (s *Scheduler) Cancel(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
}

// Start sets running=true and spawns the loop goroutine.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = true
	go s.loop()
}

// Stop closes the stop channel to terminate the loop. It is safe to call multiple times.
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.stop)
	})
}

// loop ticks every 100ms, checks if any job's TriggerTime <= now, fires and removes it.
func (s *Scheduler) loop() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stop:
			return
		case now := <-ticker.C:
			var readyJobs []*Job

			s.mu.Lock()
			for id, job := range s.jobs {
				if !now.Before(job.TriggerTime) {
					readyJobs = append(readyJobs, job)
					delete(s.jobs, id)
				}
			}
			s.mu.Unlock()

			for _, job := range readyJobs {
				go job.Execute()
			}
		}
	}
}
