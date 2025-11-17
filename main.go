package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		workers     = flag.Int("workers", 2, "Cantidad de workers en la simulación")
		events      = flag.Int("events", 40, "Cantidad total de eventos externos generados")
		logPath     = flag.String("log", "execution.log", "Archivo de log a generar")
		seed        = flag.Int64("seed", time.Now().UnixNano(), "Semilla utilizada para los generadores de eventos")
		channelBuf  = flag.Int("channel-buffer", 8, "Tamaño de los canales entre scheduler y workers")
		internalMin = flag.Int("internal-min", 1, "Cantidad mínima de eventos internos por evento externo")
		internalMax = flag.Int("internal-max", 3, "Cantidad máxima de eventos internos por evento externo")
		jumpMin     = flag.Int("jump-min", 1, "Avance mínimo del LVT causado por un evento interno")
		jumpMax     = flag.Int("jump-max", 5, "Avance máximo del LVT causado por un evento interno")
		maxTime     = flag.Int("max-time", 50, "Límite superior del tiempo virtual (LVT) para scheduler y workers")
		speedup     = flag.Bool("speedup", false, "Ejecuta automáticamente la medición de speedup (1,2,4,8 workers)")
	)
	flag.Parse()

	cfg := ConfigSimulacion{
		NumWorkers:          *workers,
		TotalExternalEvents: *events,
		InternalMinEvents:   *internalMin,
		InternalMaxEvents:   *internalMax,
		InternalMinJump:     *jumpMin,
		InternalMaxJump:     *jumpMax,
		ChannelBuffer:       *channelBuf,
		LogPath:             *logPath,
		Seed:                *seed,
		MaxVirtualTime:      *maxTime,
	}

	if *speedup {
		if err := runExperimentoSpeedup(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "experimento de speedup fallido: %v\n", err)
			os.Exit(1)
		}
		return
	}

	res, err := RunSimulacion(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "simulacion fallida: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Simulación completada en %s. Eventos despachados: %d\n", res.Duration, res.EventsDispatched)
	for _, s := range res.WorkerStats {
		fmt.Printf("Worker %d -> externo: %d, interno: %d, rollbacks: %d, checkpoints: %d, LVT final: %d\n",
			s.ID, s.EventosExternos, s.EventosInternos, s.Rollbacks, s.CheckpointsBuilt, s.LastVirtualTime)
	}
	fmt.Printf("Logs guardados en %s\n", cfg.LogPath)
}
