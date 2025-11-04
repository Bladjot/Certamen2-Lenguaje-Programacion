package main

import (
	"fmt"
)

// estas se asignaran en algun momento al tipo de evento
const (
	TipoExterno = 0
	TipoInterno = 1
)

type Evento struct {
	tipo            int
	tiempo          int
	workerDestinoID int
}

type WorkerState struct {
	lvt         int
	finServicio int
}

type Checkpoint struct {
	estado WorkerState
}

type Worker struct {
	id                       int
	estado                   WorkerState
	canalEntrada             chan Evento
	checkpoints              []Checkpoint
	historialEventosExternos []Evento
}

func main() {
	fmt.Println("")
}
