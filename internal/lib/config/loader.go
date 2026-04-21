package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	IWADPath     *string  `json:"iwad_path"`
	MapName      *string  `json:"map_name"`
	PlayMode     *bool    `json:"play_mode"`
	TopDownDebug *bool    `json:"topdown_debug"`
	WindowWidth  *int     `json:"window_width"`
	WindowHeight *int     `json:"window_height"`
	Zoom         *float64 `json:"zoom"`
	TickRate     *int     `json:"tick_rate"`
	RuntimeTicks *int     `json:"runtime_ticks"`
}

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config %q: %w", path, err)
	}

	if cfg.WindowWidth != nil && *cfg.WindowWidth <= 0 {
		return Config{}, invalidConfig("window_width must be > 0")
	}
	if cfg.WindowHeight != nil && *cfg.WindowHeight <= 0 {
		return Config{}, invalidConfig("window_height must be > 0")
	}
	if cfg.Zoom != nil && *cfg.Zoom <= 0 {
		return Config{}, invalidConfig("zoom must be > 0")
	}
	if cfg.TickRate != nil && *cfg.TickRate <= 0 {
		return Config{}, invalidConfig("tick_rate must be > 0")
	}
	if cfg.RuntimeTicks != nil && *cfg.RuntimeTicks < 0 {
		return Config{}, invalidConfig("runtime_ticks must be >= 0")
	}

	return cfg, nil
}

func invalidConfig(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidConfig, message)
}
