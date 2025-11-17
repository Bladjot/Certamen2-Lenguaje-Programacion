package main

import (
	"fmt"
	"sync"
	"time"
)

func RunSimulacion(cfg ConfigSimulacion) (*ResultadoSimulacion, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	logger, err := NewLogger(cfg.LogPath)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	start := time.Now()
	workerChans := make([]chan Evento, cfg.NumWorkers)
	workers := make([]*Worker, cfg.NumWorkers)
	var wg sync.WaitGroup
	for i := 0; i < cfg.NumWorkers; i++ {
		workerChans[i] = make(chan Evento, cfg.ChannelBuffer)
		workers[i] = NewWorker(i, cfg, workerChans[i], logger)
		wg.Add(1)
		go workers[i].Run(&wg)
	}
	scheduler := NewScheduler(cfg, workerChans, logger)
	actualEvents := scheduler.Run()
	for _, ch := range workerChans {
		close(ch)
	}
	wg.Wait()
	duration := time.Since(start)
	stats := make([]WorkerStats, len(workers))
	for i, worker := range workers {
		stats[i] = worker.stats
	}
	return &ResultadoSimulacion{
		Duration:         duration,
		EventsDispatched: actualEvents,
		WorkerStats:      stats,
	}, nil
}

func runExperimentoSpeedup(base ConfigSimulacion) error {
	counts := []int{1, 2, 4, 8}
	results := make([]time.Duration, len(counts))
	for idx, workers := range counts {
		cfg := base
		cfg.NumWorkers = workers
		cfg.LogPath = fmt.Sprintf("speedup_w%d.log", workers)
		res, err := RunSimulacion(cfg)
		if err != nil {
			return err
		}
		results[idx] = res.Duration
		fmt.Printf("Workers: %d \tDuracion: %s \tLog: %s \tEventos: %d\n", workers, res.Duration, cfg.LogPath, res.EventsDispatched)
	}
	baseDuration := results[0]
	fmt.Println("\nSpeedup analisis:")
	for idx, workers := range counts {
		speedup := float64(baseDuration) / float64(results[idx])
		fmt.Printf("Workers: %d \tSpeedup: %.2f\n", workers, speedup)
	}
	return nil
}
