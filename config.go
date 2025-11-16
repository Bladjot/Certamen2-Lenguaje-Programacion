package main

import "errors"

func (cfg SimulationConfig) validate() error {
	if cfg.NumWorkers <= 0 {
		return errors.New("numWorkers must be > 0")
	}
	if cfg.TotalExternalEvents < cfg.NumWorkers {
		return errors.New("total events must be >= numWorkers")
	}
	if cfg.InternalMinEvents <= 0 || cfg.InternalMaxEvents < cfg.InternalMinEvents {
		return errors.New("invalid internal event count bounds")
	}
	if cfg.InternalMinJump <= 0 || cfg.InternalMaxJump < cfg.InternalMinJump {
		return errors.New("invalid internal jump bounds")
	}
	if cfg.ChannelBuffer <= 0 {
		return errors.New("channel buffer must be > 0")
	}
	if cfg.LogPath == "" {
		return errors.New("log path required")
	}
	if cfg.MaxVirtualTime <= 0 {
		return errors.New("max virtual time must be > 0")
	}
	return nil
}
