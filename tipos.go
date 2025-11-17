package main

import "time"

// Tipo de evento
const (
	TipoExterno = iota
	TipoInterno
)

// Evento representa un evento externo generado por el scheduler.
type Evento struct {
	ID              int `json:"event_id"`
	Tipo            int `json:"tipo"`
	Tiempo          int `json:"timestamp"`
	WorkerDestinoID int `json:"worker_id"`
}

// WorkerState modela las variables que deben sobrevivir a un rollback.
type WorkerState struct {
	LVT int `json:"lvt"`
}

// Checkpoint almacena el estado y el número de eventos procesados.
type Checkpoint struct {
	Estado     WorkerState
	HistoryLen int
}

// ConfigSimulacion parametriza la ejecución.
type ConfigSimulacion struct {
	NumWorkers          int
	TotalExternalEvents int
	InternalMinEvents   int
	InternalMaxEvents   int
	InternalMinJump     int
	InternalMaxJump     int
	ChannelBuffer       int
	LogPath             string
	Seed                int64
	MaxVirtualTime      int
}

// ResultadoSimulacion resume los datos claves de una corrida.
type ResultadoSimulacion struct {
	Duration         time.Duration
	EventsDispatched int
	WorkerStats      []WorkerStats
}

// WorkerStats expone métricas por worker.
type WorkerStats struct {
	ID               int
	EventosExternos  int
	EventosInternos  int
	Rollbacks        int
	LastVirtualTime  int
	CheckpointsBuilt int
}
