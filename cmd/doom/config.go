package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/kelseyhightower/envconfig"

	"github.com/turtlepavlo/go-doom/internal/app"
	"github.com/turtlepavlo/go-doom/internal/lib/config"
)

var ErrInvalidAppConfig = errors.New("invalid app config")

type Config struct {
	App app.Config
}

func (cfg Config) ValidateWithContext(ctx context.Context) error {
	return validation.ValidateStructWithContext(ctx, &cfg,
		validation.Field(&cfg.App),
	)
}

func Load(ctx context.Context) (Config, error) {
	return LoadFromArgs(ctx, os.Args[1:])
}

func LoadFromArgs(ctx context.Context, args []string) (Config, error) {
	defaults := app.DefaultConfig()

	cliCfg, cliExplicit, cliConfigPath, err := parseCLI(args, defaults)
	if err != nil {
		return Config{}, err
	}

	// File config selection uses either `-config` or `DOOM_CONFIG_PATH`.
	type envConfigPath struct {
		ConfigPath string `envconfig:"DOOM_CONFIG_PATH"`
	}
	var envPath envConfigPath
	if err := envconfig.Process("", &envPath); err != nil {
		return Config{}, err
	}

	loader := config.NewLoader()
	merged := defaults

	configPath := strings.TrimSpace(cliConfigPath)
	if configPath == "" {
		configPath = strings.TrimSpace(envPath.ConfigPath)
	}

	if configPath != "" {
		fileCfg, loadErr := loader.Load(configPath)
		if loadErr != nil {
			return Config{}, fmt.Errorf("load config file %q: %w", configPath, loadErr)
		}
		merged = applyOptional(merged, fileCfg)
	} else if defaultPath := filepath.Join("configs", "default.json"); fileExists(defaultPath) {
		fileCfg, loadErr := loader.Load(defaultPath)
		if loadErr != nil {
			return Config{}, fmt.Errorf("load default config file %q: %w", defaultPath, loadErr)
		}
		merged = applyOptional(merged, fileCfg)
	}

	// Env values override code defaults + file config.
	if err := envconfig.Process("", &merged); err != nil {
		return Config{}, err
	}

	// CLI flags override env + file config.
	merged = applyCLI(merged, cliCfg, cliExplicit)

	merged.IWADPath = strings.TrimSpace(merged.IWADPath)
	merged.MapName = strings.TrimSpace(merged.MapName)

	cfg := Config{App: merged}
	if err := cfg.ValidateWithContext(ctx); err != nil {
		return Config{}, fmt.Errorf("%w: %w", ErrInvalidAppConfig, err)
	}

	return cfg, nil
}

func parseCLI(args []string, defaults app.Config) (app.Config, map[string]bool, string, error) {
	fs := flag.NewFlagSet("doom", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var (
		configPath   string
		iwadPath     string
		mapName      string
		playMode     bool
		topDownDebug bool
		windowWidth  int
		windowHeight int
		zoom         float64
		tickRate     int
		runtimeTicks int
	)

	fs.StringVar(&configPath, "config", "", "path to JSON config file")
	fs.StringVar(&iwadPath, "iwad", defaults.IWADPath, "path to IWAD/PWAD file")
	fs.IntVar(&runtimeTicks, "runtime-ticks", defaults.RuntimeTicks, "run headless runtime loop for N ticks after loading WAD")
	fs.IntVar(&tickRate, "tick-rate", defaults.TickRate, "runtime loop tick rate")
	fs.StringVar(&mapName, "map", defaults.MapName, "parse selected map marker (for example E1M1 or MAP01)")
	fs.BoolVar(&playMode, "play", defaults.PlayMode, "run playable first-person mode")
	fs.IntVar(&windowWidth, "width", defaults.WindowWidth, "window width in play mode")
	fs.IntVar(&windowHeight, "height", defaults.WindowHeight, "window height in play mode")
	fs.Float64Var(&zoom, "zoom", defaults.Zoom, "zoom multiplier in play mode")
	fs.BoolVar(&topDownDebug, "topdown", defaults.TopDownDebug, "use top-down debug renderer in play mode")

	if err := fs.Parse(args); err != nil {
		return app.Config{}, nil, "", err
	}

	explicit := make(map[string]bool, 10)
	fs.Visit(func(item *flag.Flag) {
		explicit[item.Name] = true
	})

	cliCfg := defaults
	cliCfg.IWADPath = iwadPath
	cliCfg.MapName = mapName
	cliCfg.PlayMode = playMode
	cliCfg.TopDownDebug = topDownDebug
	cliCfg.WindowWidth = windowWidth
	cliCfg.WindowHeight = windowHeight
	cliCfg.Zoom = zoom
	cliCfg.TickRate = tickRate
	cliCfg.RuntimeTicks = runtimeTicks

	return cliCfg, explicit, configPath, nil
}

func applyOptional(dst app.Config, src config.Config) app.Config {
	assignIfPresent(src.IWADPath, &dst.IWADPath)
	assignIfPresent(src.MapName, &dst.MapName)
	assignIfPresent(src.PlayMode, &dst.PlayMode)
	assignIfPresent(src.TopDownDebug, &dst.TopDownDebug)
	assignIfPresent(src.WindowWidth, &dst.WindowWidth)
	assignIfPresent(src.WindowHeight, &dst.WindowHeight)
	assignIfPresent(src.Zoom, &dst.Zoom)
	assignIfPresent(src.TickRate, &dst.TickRate)
	assignIfPresent(src.RuntimeTicks, &dst.RuntimeTicks)
	return dst
}

func applyCLI(dst app.Config, cli app.Config, explicit map[string]bool) app.Config {
	if explicit["iwad"] {
		dst.IWADPath = cli.IWADPath
	}
	if explicit["map"] {
		dst.MapName = cli.MapName
	}
	if explicit["play"] {
		dst.PlayMode = cli.PlayMode
	}
	if explicit["topdown"] {
		dst.TopDownDebug = cli.TopDownDebug
	}
	if explicit["width"] {
		dst.WindowWidth = cli.WindowWidth
	}
	if explicit["height"] {
		dst.WindowHeight = cli.WindowHeight
	}
	if explicit["zoom"] {
		dst.Zoom = cli.Zoom
	}
	if explicit["tick-rate"] {
		dst.TickRate = cli.TickRate
	}
	if explicit["runtime-ticks"] {
		dst.RuntimeTicks = cli.RuntimeTicks
	}
	return dst
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func assignIfPresent[T any](value *T, dst *T) {
	if value != nil {
		*dst = *value
	}
}
