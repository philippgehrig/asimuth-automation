package scheduler

import (
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
	jobs    map[string]*Job
	mu      sync.Mutex
	stop    chan struct{}
	running bool
}

// New creates a new Scheduler instance.
func New() *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		stop: make(chan struct{}),
	}
}

// ParseTime parses a "HH:MM" string into [hour, minute].
func ParseTime(s string) [2]int {
	parts := strings.Split(s, ":")
	hour, _ := strconv.Atoi(parts[0])
	minute, _ := strconv.Atoi(parts[1])
	return [2]int{hour, minute}
}

// CalculateTriggerTime computes when to fire a booking request.
// Formula: trigger = (slot_date - 2 days) + slot_start_time + 30min.
// date is formatted as "2006-01-02", startTime as "HH:MM".
func CalculateTriggerTime(date, startTime string, loc *time.Location) time.Time {
	slotDate, _ := time.ParseInLocation("2006-01-02", date, loc)
	hm := ParseTime(startTime)

	trigger := slotDate.AddDate(0, 0, -2)
	trigger = time.Date(trigger.Year(), trigger.Month(), trigger.Day(), hm[0], hm[1], 0, 0, loc)
	trigger = trigger.Add(30 * time.Minute)

	return trigger
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

// Stop closes the stop channel to terminate the loop.
func (s *Scheduler) Stop() {
	close(s.stop)
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
			s.mu.Lock()
			for id, job := range s.jobs {
				if !job.TriggerTime.After(now) {
					go job.Execute()
					delete(s.jobs, id)
				}
			}
			s.mu.Unlock()
		}
	}
}
