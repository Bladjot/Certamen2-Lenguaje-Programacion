package main

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// Worker procesa eventos y maneja checkpoints/rollbacks.
type Worker struct {
	id          int
	cfg         SimulationConfig
	input       chan Evento
	logger      *Logger
	rand        *rand.Rand
	state       WorkerState
	history     []Evento
	checkpoints []Checkpoint
	stats       WorkerStats
}

func NewWorker(id int, cfg SimulationConfig, input chan Evento, logger *Logger) *Worker {
	w := &Worker{
		id:     id,
		cfg:    cfg,
		input:  input,
		logger: logger,
		rand:   rand.New(rand.NewSource(cfg.Seed + int64(id)*17 + 99)),
		state:  WorkerState{LVT: 0},
	}
	w.stats.ID = id
	w.checkpoints = append(w.checkpoints, Checkpoint{Estado: w.state, HistoryLen: 0})
	return w
}

func (w *Worker) Run(wg *sync.WaitGroup) {
	defer wg.Done()
	for event := range w.input {
		w.handleExternal(event)
	}
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "worker_stopped",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
	})
}

func (w *Worker) entityName() string {
	return fmt.Sprintf("worker-%d", w.id)
}

func (w *Worker) handleExternal(event Evento) {
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "external_received",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"event_timestamp": event.Tiempo,
		},
	})
	w.createCheckpoint(len(w.history), "live")
	if event.Tiempo < w.state.LVT {
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.entityName(),
			Event:    "straggler_detected",
			WorkerID: w.id,
			EventID:  event.ID,
			SimTime:  w.state.LVT,
			Details: map[string]any{
				"event_timestamp": event.Tiempo,
			},
		})
		w.performRollback(event)
		return
	}
	w.appendEvent(event)
	w.processEvent(event, false)
}

func (w *Worker) appendEvent(event Evento) {
	idx := sort.Search(len(w.history), func(i int) bool {
		if w.history[i].Tiempo == event.Tiempo {
			return w.history[i].ID >= event.ID
		}
		return w.history[i].Tiempo > event.Tiempo
	})
	w.history = append(w.history, Evento{})
	copy(w.history[idx+1:], w.history[idx:])
	w.history[idx] = event
}

func (w *Worker) processEvent(event Evento, fromReplay bool) {
	prev := w.state.LVT
	w.state.LVT = event.Tiempo
	w.stats.ExternalEvents++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "external_processed",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"from_replay":  fromReplay,
			"previous_lvt": prev,
		},
	})
	w.generateInternalEvents()
	w.stats.LastVirtualTime = w.state.LVT
}

func (w *Worker) generateInternalEvents() {
	count := w.rand.Intn(w.cfg.InternalMaxEvents-w.cfg.InternalMinEvents+1) + w.cfg.InternalMinEvents
	for i := 0; i < count; i++ {
		if w.state.LVT >= w.cfg.MaxVirtualTime {
			return
		}
		jump := w.rand.Intn(w.cfg.InternalMaxJump-w.cfg.InternalMinJump+1) + w.cfg.InternalMinJump
		prev := w.state.LVT
		newTime := prev + jump
		if newTime > w.cfg.MaxVirtualTime {
			newTime = w.cfg.MaxVirtualTime
		}
		w.state.LVT = newTime
		w.stats.InternalEvents++
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.entityName(),
			Event:    "internal_processed",
			WorkerID: w.id,
			SimTime:  w.state.LVT,
			Details: map[string]any{
				"previous_lvt": prev,
				"jump":         w.state.LVT - prev,
			},
		})
	}
}

func (w *Worker) performRollback(straggler Evento) {
	w.stats.Rollbacks++
	w.appendEvent(straggler)
	cpIdx := w.findCheckpointFor(straggler.Tiempo)
	if cpIdx < 0 {
		cpIdx = 0
	}
	rollbackFrom := w.state.LVT
	cp := w.checkpoints[cpIdx]
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.entityName(),
		Event:        "rollback_start",
		WorkerID:     w.id,
		SimTime:      cp.Estado.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
	w.state = cp.Estado
	w.stats.LastVirtualTime = w.state.LVT
	w.checkpoints = w.checkpoints[:cpIdx+1]
	for idx := cp.HistoryLen; idx < len(w.history); idx++ {
		e := w.history[idx]
		w.createCheckpoint(idx, "replay")
		w.processEvent(e, true)
	}
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.entityName(),
		Event:        "rollback_end",
		WorkerID:     w.id,
		SimTime:      w.state.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
}

func (w *Worker) findCheckpointFor(ts int) int {
	for i := len(w.checkpoints) - 1; i >= 0; i-- {
		if w.checkpoints[i].Estado.LVT <= ts {
			return i
		}
	}
	return -1
}

func (w *Worker) createCheckpoint(historyLen int, mode string) {
	cp := Checkpoint{Estado: w.state, HistoryLen: historyLen}
	w.checkpoints = append(w.checkpoints, cp)
	w.stats.CheckpointsBuilt++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.entityName(),
		Event:    "checkpoint_created",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"history_len": historyLen,
			"mode":        mode,
		},
	})
}
