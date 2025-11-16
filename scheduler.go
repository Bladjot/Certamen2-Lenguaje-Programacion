package main

import (
	"math/rand"
	"time"
)

// Scheduler genera eventos externos en orden creciente de tiempo.
type Scheduler struct {
	cfg         SimulationConfig
	workerChans []chan Evento
	logger      *Logger
	rand        *rand.Rand
	clock       int
	nextEventID int
}

func NewScheduler(cfg SimulationConfig, workerChans []chan Evento, logger *Logger) *Scheduler {
	return &Scheduler{
		cfg:         cfg,
		workerChans: workerChans,
		logger:      logger,
		rand:        rand.New(rand.NewSource(cfg.Seed + 42)),
	}
}

func (s *Scheduler) Run() int {
	if s.cfg.TotalExternalEvents == 0 {
		return 0
	}
	sent := 0
	for i := 0; i < s.cfg.TotalExternalEvents; i++ {
		s.clock += s.rand.Intn(4) + 1
		if s.clock > s.cfg.MaxVirtualTime {
			break
		}
		target := s.rand.Intn(s.cfg.NumWorkers)
		event := Evento{
			ID:              s.nextEventID,
			Tipo:            TipoExterno,
			Tiempo:          s.clock,
			WorkerDestinoID: target,
		}
		s.nextEventID++
		s.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   "scheduler",
			Event:    "external_dispatched",
			SimTime:  s.clock,
			EventID:  event.ID,
			Target:   target,
		})
		s.workerChans[target] <- event
		sent++
	}
	return sent
}
