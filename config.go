package main

import "errors"

func (cfg ConfigSimulacion) validate() error {
	if cfg.NumWorkers <= 0 {
		return errors.New("numWorkers debe ser > 0")
	}
	if cfg.TotalExternalEvents < cfg.NumWorkers {
		return errors.New("total events debe ser >= numWorkers")
	}
	if cfg.InternalMinEvents <= 0 || cfg.InternalMaxEvents < cfg.InternalMinEvents {
		return errors.New("límites inválidos para la cantidad de eventos internos")
	}
	if cfg.InternalMinJump <= 0 || cfg.InternalMaxJump < cfg.InternalMinJump {
		return errors.New("límites inválidos para el salto interno")
	}
	if cfg.ChannelBuffer <= 0 {
		return errors.New("el buffer del canal debe ser > 0")
	}
	if cfg.LogPath == "" {
		return errors.New("ruta de log requerida")
	}
	if cfg.MaxVirtualTime <= 0 {
		return errors.New("el tiempo virtual máximo debe ser > 0")
	}
	return nil
}
