package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
	"github.com/turtlepavlo/go-doom/internal/application/playmap"
	"github.com/turtlepavlo/go-doom/internal/application/rungame"
	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/ebitenplay"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/render"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/runtime"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/wad"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

func Run(ctx context.Context, cfg Config, out io.Writer) error {
	loadCtx, cancelLoad := context.WithTimeout(ctx, 10*time.Second)
	defer cancelLoad()

	loadArchiveService, err := loadiwad.New(
		wad.NewBinaryReader(),
		wad.NewToArchiveConvert(),
	)
	if err != nil {
		return fmt.Errorf("bootstrap load IWAD service: %w", err)
	}

	archive, err := loadArchiveService.Execute(loadCtx, cfg.IWADPath)
	if err != nil {
		return fmt.Errorf("load WAD: %w", err)
	}
	printArchiveSummary(out, archive)

	selectedMap := strings.TrimSpace(cfg.MapName)
	if cfg.PlayMode && selectedMap == "" {
		if len(archive.Maps) == 0 {
			return errors.New("play mode requires a WAD with map markers")
		}
		selectedMap = archive.Maps[0].Name
		fmt.Fprintf(out, "Play mode default map: %s\n", selectedMap)
	}

	var parsedLevel domain.Level
	if selectedMap != "" {
		parsedLevel, err = loadLevel(loadCtx, cfg.IWADPath, selectedMap)
		if err != nil {
			return fmt.Errorf("load map: %w", err)
		}
		printLevelSummary(out, parsedLevel)
	}

	if cfg.PlayMode {
		if parsedLevel.Name == "" {
			return errors.New("play mode requires a valid map")
		}
		return runPlayable(
			ctx,
			parsedLevel,
			cfg.TickRate,
			cfg.WindowWidth,
			cfg.WindowHeight,
			cfg.Zoom,
			cfg.TopDownDebug,
		)
	}

	if cfg.RuntimeTicks <= 0 {
		return nil
	}

	runtimeCtx, cancelRuntime := context.WithTimeout(
		ctx,
		estimateRuntimeTimeout(cfg.RuntimeTicks, cfg.TickRate),
	)
	defer cancelRuntime()

	if err := runRuntime(runtimeCtx, cfg.RuntimeTicks, cfg.TickRate, out); err != nil {
		return fmt.Errorf("run runtime loop: %w", err)
	}
	return nil
}

func runRuntime(ctx context.Context, maxTicks int, tickRate int, out io.Writer) error {
	timer, err := runtime.NewFixedTimer(tickRate)
	if err != nil {
		return fmt.Errorf("create fixed timer: %w", err)
	}

	simulation, err := runtime.NewDomainSimulation(runtime.NewEngine())
	if err != nil {
		return fmt.Errorf("create domain simulation: %w", err)
	}

	runtimeService, err := rungame.New(
		runtime.NewNoopControlPoller(),
		controls.NewCommandMapper(),
		simulation,
		runtime.NewHeadlessRenderer(out),
		timer,
	)
	if err != nil {
		return fmt.Errorf("create runtime service: %w", err)
	}

	return runtimeService.Run(ctx, maxTicks)
}

func loadLevel(ctx context.Context, wadPath string, mapName string) (domain.Level, error) {
	loadMapService, err := loadmap.New(
		wad.NewMapReader(),
		wad.NewToLevelConvert(),
	)
	if err != nil {
		return domain.Level{}, fmt.Errorf("bootstrap load map service: %w", err)
	}

	level, err := loadMapService.Execute(ctx, wadPath, mapName)
	if err != nil {
		return domain.Level{}, err
	}
	return level, nil
}

func runPlayable(ctx context.Context, level domain.Level, tickRate int, width int, height int, zoom float64, topDownDebug bool) error {
	spawn, ok := runtime.FindPlayerSpawn(level)
	if !ok {
		spawn = runtime.PlayerSpawn{}
	}

	engine := runtime.NewEnginePose(spawn.X, spawn.Y, spawn.Angle)
	simulation, err := runtime.NewLevelSimulation(engine, &level)
	if err != nil {
		return fmt.Errorf("create level simulation: %w", err)
	}

	playService, err := playmap.New(
		controls.NewCommandMapper(),
		simulation,
	)
	if err != nil {
		return fmt.Errorf("create playmap service: %w", err)
	}

	controlPoller := controls.NewControlPoller()
	var renderer ebitenplay.FrameRenderer
	if topDownDebug {
		renderer = render.NewTopDownRenderer(level, width, height, zoom)
	} else {
		renderer = render.NewFirstPersonRenderer(level, width, height, zoom)
	}
	game := ebitenplay.New(ctx, playService, controlPoller, renderer, simulation.Frame())

	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle(fmt.Sprintf("go-doom | %s | W/S move, A/D turn, Q/E strafe, Space/LMB fire, Esc quit", level.Name))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(tickRate)

	err = ebiten.RunGame(game)
	if err != nil && !errors.Is(err, ebitenplay.ErrExitRequested) {
		return err
	}
	return nil
}

func printArchiveSummary(out io.Writer, archive domain.Archive) {
	fmt.Fprintf(
		out,
		"Loaded %s with %d lumps and %d map markers\n",
		archive.Header.Kind,
		len(archive.Lumps),
		len(archive.Maps),
	)
	for i, gameMap := range archive.Maps {
		if i >= 5 {
			fmt.Fprintf(out, "... and %d more maps\n", len(archive.Maps)-5)
			break
		}
		fmt.Fprintf(out, "  %s: %d lumps\n", gameMap.Name, len(gameMap.Lumps))
	}
}

func printLevelSummary(out io.Writer, level domain.Level) {
	fmt.Fprintf(
		out,
		"Parsed map %s: things=%d linedefs=%d sidedefs=%d vertexes=%d sectors=%d\n",
		level.Name,
		len(level.Things),
		len(level.Linedefs),
		len(level.Sidedefs),
		len(level.Vertexes),
		len(level.Sectors),
	)
}

func estimateRuntimeTimeout(maxTicks int, tickRate int) time.Duration {
	if maxTicks <= 0 {
		return 30 * time.Second
	}

	resolvedTickRate := tickRate
	if resolvedTickRate <= 0 {
		resolvedTickRate = 35
	}

	seconds := float64(maxTicks) / float64(resolvedTickRate)
	return time.Duration(seconds*float64(time.Second)) + 5*time.Second
}
