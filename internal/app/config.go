package app

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type Config struct {
	IWADPath     string `envconfig:"DOOM_IWAD_PATH"`
	MapName      string `envconfig:"DOOM_MAP"`
	PlayMode     bool   `envconfig:"DOOM_PLAY"`
	TopDownDebug bool   `envconfig:"DOOM_TOPDOWN"`

	WindowWidth  int     `envconfig:"DOOM_WIDTH"`
	WindowHeight int     `envconfig:"DOOM_HEIGHT"`
	Zoom         float64 `envconfig:"DOOM_ZOOM"`

	TickRate     int `envconfig:"DOOM_TICK_RATE"`
	RuntimeTicks int `envconfig:"DOOM_RUNTIME_TICKS"`
}

func (cfg Config) ValidateWithContext(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &cfg,
		validation.Field(&cfg.IWADPath, validation.Required),
	)
}

func DefaultConfig() Config {
	return Config{
		IWADPath:     "",
		MapName:      "",
		PlayMode:     false,
		TopDownDebug: false,
		WindowWidth:  1280,
		WindowHeight: 720,
		Zoom:         1.0,
		TickRate:     35,
		RuntimeTicks: 0,
	}
}
