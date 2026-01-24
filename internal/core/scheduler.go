package core

import (
	"log/slog"
	"sync"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	logger     *slog.Logger
	cron       *cron.Cron
	engine     *TachyonEngine
	startEntry cron.EntryID
	stopEntry  cron.EntryID
	mu         sync.Mutex
	config     ScheduleConfig
}

type ScheduleConfig struct {
	Enabled   bool
	StartHour int // 0-23
	StopHour  int // 0-23
}

func NewScheduler(logger *slog.Logger, engine *TachyonEngine) *Scheduler {
	return &Scheduler{
		logger: logger,
		cron:   cron.New(),
		engine: engine,
	}
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) UpdateSchedule(cfg ScheduleConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = cfg

	// Remove existing jobs
	if s.startEntry != 0 {
		s.cron.Remove(s.startEntry)
	}
	if s.stopEntry != 0 {
		s.cron.Remove(s.stopEntry)
	}

	if !cfg.Enabled {
		return
	}

	// Add Start Job (Resume All)
	// Cron spec: "0 0 8 * * *" (At 08:00)
	// robfig/cron/v3 standard parser expects 5 fields: min hour dom month dow
	// We construct spec string.

	startSpec := specFromHour(cfg.StartHour)
	stopSpec := specFromHour(cfg.StopHour)

	id1, err := s.cron.AddFunc(startSpec, func() {
		s.logger.Info("Scheduler: Starting Downloads")
		// Engine specific method to ResumeAll needed
		// For now, iterate active/paused logic?
		// Let's assume Engine exposes ResumeAll() or we add it.
		// s.engine.ResumeAll()
	})
	if err == nil {
		s.startEntry = id1
	} else {
		s.logger.Error("Failed to schedule start", "error", err)
	}

	id2, err := s.cron.AddFunc(stopSpec, func() {
		s.logger.Info("Scheduler: Stopping Downloads")
		// s.engine.PauseAll()
	})
	if err == nil {
		s.stopEntry = id2
	} else {
		s.logger.Error("Failed to schedule stop", "error", err)
	}

	s.logger.Info("Schedule updated", "start", cfg.StartHour, "stop", cfg.StopHour)
}

func specFromHour(hour int) string {
	// Minute 0, Hour X, Every Day
	// "0 X * * *"
	// Handle wrap if needed? No, 0-23 is standard.
	// Ensure valid string.
	return "0 " + string(rune('0'+(hour/10))) + string(rune('0'+(hour%10))) + " * * *"
	// Better: fmt.Sprintf("0 %d * * *", hour)
}
