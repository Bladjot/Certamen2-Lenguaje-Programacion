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
	cfg         ConfigSimulacion
	input       chan Evento
	logger      *Logger
	rand        *rand.Rand
	state       WorkerState
	historial   []Evento
	checkpoints []Checkpoint
	stats       WorkerStats
}

func NewWorker(id int, cfg ConfigSimulacion, input chan Evento, logger *Logger) *Worker {
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
		w.manejoEventoExterno(event)
	}
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.nombreEntidad(),
		Event:    "worker_stopped",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
	})
}

func (w *Worker) nombreEntidad() string {
	return fmt.Sprintf("worker-%d", w.id)
}

func (w *Worker) manejoEventoExterno(event Evento) {
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.nombreEntidad(),
		Event:    "external_received",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"event_timestamp": event.Tiempo,
		},
	})
	w.crearCheckpoint(len(w.historial), "live")
	if event.Tiempo < w.state.LVT {
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.nombreEntidad(),
			Event:    "straggler_detected",
			WorkerID: w.id,
			EventID:  event.ID,
			SimTime:  w.state.LVT,
			Details: map[string]any{
				"event_timestamp": event.Tiempo,
			},
		})
		w.ejecutarRollback(event)
		return
	}
	w.insertarEvento(event)
	w.procesarEvento(event, false)
}

func (w *Worker) insertarEvento(event Evento) {
	idx := sort.Search(len(w.historial), func(i int) bool {
		if w.historial[i].Tiempo == event.Tiempo {
			return w.historial[i].ID >= event.ID
		}
		return w.historial[i].Tiempo > event.Tiempo
	})
	w.historial = append(w.historial, Evento{})
	copy(w.historial[idx+1:], w.historial[idx:])
	w.historial[idx] = event
}

func (w *Worker) procesarEvento(event Evento, fromReplay bool) {
	prev := w.state.LVT
	w.state.LVT = event.Tiempo
	w.stats.EventosExternos++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.nombreEntidad(),
		Event:    "external_processed",
		WorkerID: w.id,
		EventID:  event.ID,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"from_replay":  fromReplay,
			"previous_lvt": prev,
		},
	})
	w.generarEventosInternos()
	w.stats.LastVirtualTime = w.state.LVT
}

func (w *Worker) generarEventosInternos() {
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
		w.stats.EventosInternos++
		w.logger.Log(LogEntry{
			WallTime: time.Now(),
			Entity:   w.nombreEntidad(),
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

func (w *Worker) ejecutarRollback(straggler Evento) {
	w.stats.Rollbacks++
	w.insertarEvento(straggler)
	cpIdx := w.buscarCheckpoint(straggler.Tiempo)
	if cpIdx < 0 {
		cpIdx = 0
	}
	rollbackFrom := w.state.LVT
	cp := w.checkpoints[cpIdx]
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.nombreEntidad(),
		Event:        "rollback_start",
		WorkerID:     w.id,
		SimTime:      cp.Estado.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
	w.state = cp.Estado
	w.stats.LastVirtualTime = w.state.LVT
	w.checkpoints = w.checkpoints[:cpIdx+1]
	for idx := cp.HistoryLen; idx < len(w.historial); idx++ {
		e := w.historial[idx]
		w.crearCheckpoint(idx, "replay")
		w.procesarEvento(e, true)
	}
	w.logger.Log(LogEntry{
		WallTime:     time.Now(),
		Entity:       w.nombreEntidad(),
		Event:        "rollback_end",
		WorkerID:     w.id,
		SimTime:      w.state.LVT,
		RollbackFrom: rollbackFrom,
		RollbackTo:   straggler.Tiempo,
	})
}

func (w *Worker) buscarCheckpoint(ts int) int {
	for i := len(w.checkpoints) - 1; i >= 0; i-- {
		if w.checkpoints[i].Estado.LVT <= ts {
			return i
		}
	}
	return -1
}

func (w *Worker) crearCheckpoint(historyLen int, mode string) {
	cp := Checkpoint{Estado: w.state, HistoryLen: historyLen}
	w.checkpoints = append(w.checkpoints, cp)
	w.stats.CheckpointsBuilt++
	w.logger.Log(LogEntry{
		WallTime: time.Now(),
		Entity:   w.nombreEntidad(),
		Event:    "checkpoint_created",
		WorkerID: w.id,
		SimTime:  w.state.LVT,
		Details: map[string]any{
			"history_len": historyLen,
			"mode":        mode,
		},
	})
}
