package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
	"github.com/turtlepavlo/go-doom/internal/application/playmap"
	"github.com/turtlepavlo/go-doom/internal/application/rungame"
	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/converters"
	inputinfra "github.com/turtlepavlo/go-doom/internal/infrastructure/input"
	renderinfra "github.com/turtlepavlo/go-doom/internal/infrastructure/render"
	runtimeinfra "github.com/turtlepavlo/go-doom/internal/infrastructure/runtime"
	"github.com/turtlepavlo/go-doom/internal/infrastructure/wad"
	"github.com/turtlepavlo/go-doom/internal/interfaces/ebitenplay"
)

func main() {
	var iwadPath string
	var runtimeTicks int
	var tickRate int
	var mapName string
	var playMode bool
	var windowWidth int
	var windowHeight int
	var zoom float64
	var topDownDebug bool

	flag.StringVar(&iwadPath, "iwad", "", "path to IWAD/PWAD file")
	flag.IntVar(&runtimeTicks, "runtime-ticks", 0, "run headless runtime loop for N ticks after loading WAD")
	flag.IntVar(&tickRate, "tick-rate", 35, "runtime loop tick rate")
	flag.StringVar(&mapName, "map", "", "parse selected map marker (for example E1M1 or MAP01)")
	flag.BoolVar(&playMode, "play", false, "run playable first-person mode")
	flag.IntVar(&windowWidth, "width", 1280, "window width in play mode")
	flag.IntVar(&windowHeight, "height", 720, "window height in play mode")
	flag.Float64Var(&zoom, "zoom", 1.0, "zoom multiplier in play mode")
	flag.BoolVar(&topDownDebug, "topdown", false, "use top-down debug renderer in play mode")
	flag.Parse()

	iwadPath = strings.TrimSpace(iwadPath)
	if iwadPath == "" {
		fmt.Fprintln(os.Stderr, "usage: doom -iwad <path-to-wad>")
		os.Exit(2)
	}

	loadCtx, cancelLoad := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelLoad()

	reader := wad.NewBinaryReader()
	converter := converters.NewWADConverter()
	useCase, err := loadiwad.New(reader, converter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bootstrap load IWAD use case: %v\n", err)
		os.Exit(1)
	}

	archive, err := useCase.Execute(loadCtx, iwadPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load WAD: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded %s with %d lumps and %d map markers\n", archive.Header.Kind, len(archive.Lumps), len(archive.Maps))
	for i, gameMap := range archive.Maps {
		if i >= 5 {
			fmt.Printf("... and %d more maps\n", len(archive.Maps)-5)
			break
		}
		fmt.Printf("  %s: %d lumps\n", gameMap.Name, len(gameMap.Lumps))
	}

	selectedMap := strings.TrimSpace(mapName)
	if playMode && selectedMap == "" {
		if len(archive.Maps) == 0 {
			fmt.Fprintln(os.Stderr, "play mode requires a WAD with map markers")
			os.Exit(1)
		}
		selectedMap = archive.Maps[0].Name
		fmt.Printf("Play mode default map: %s\n", selectedMap)
	}

	var parsedLevel domain.Level
	if selectedMap != "" {
		level, err := runLoadMap(loadCtx, iwadPath, selectedMap)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load map: %v\n", err)
			os.Exit(1)
		}
		parsedLevel = level
		fmt.Printf(
			"Parsed map %s: things=%d linedefs=%d sidedefs=%d vertexes=%d sectors=%d\n",
			level.Name,
			len(level.Things),
			len(level.Linedefs),
			len(level.Sidedefs),
			len(level.Vertexes),
			len(level.Sectors),
		)
	}

	if playMode {
		if parsedLevel.Name == "" {
			fmt.Fprintln(os.Stderr, "play mode requires a valid map")
			os.Exit(1)
		}
		if err := runPlayable(parsedLevel, tickRate, windowWidth, windowHeight, zoom, topDownDebug); err != nil {
			fmt.Fprintf(os.Stderr, "run playable mode: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if runtimeTicks > 0 {
		runtimeCtx, cancelRuntime := context.WithTimeout(
			context.Background(),
			estimateRuntimeTimeout(runtimeTicks, tickRate),
		)
		defer cancelRuntime()

		if err := runRuntime(runtimeCtx, runtimeTicks, tickRate); err != nil {
			fmt.Fprintf(os.Stderr, "run runtime loop: %v\n", err)
			os.Exit(1)
		}
	}
}

func runRuntime(ctx context.Context, maxTicks int, tickRate int) error {
	timer, err := runtimeinfra.NewFixedTimer(tickRate)
	if err != nil {
		return fmt.Errorf("create fixed timer: %w", err)
	}

	simulation, err := runtimeinfra.NewDomainSimulation(domain.NewEngine())
	if err != nil {
		return fmt.Errorf("create domain simulation: %w", err)
	}

	runner, err := rungame.New(
		runtimeinfra.NewNoopInput(),
		converters.NewInputConverter(),
		simulation,
		runtimeinfra.NewHeadlessRenderer(os.Stdout),
		timer,
	)
	if err != nil {
		return fmt.Errorf("create runtime use case: %w", err)
	}

	return runner.Run(ctx, maxTicks)
}

func runLoadMap(ctx context.Context, wadPath string, mapName string) (domain.Level, error) {
	useCase, err := loadmap.New(
		wad.NewMapReader(),
		converters.NewMapConverter(),
	)
	if err != nil {
		return domain.Level{}, fmt.Errorf("bootstrap load map use case: %w", err)
	}

	level, err := useCase.Execute(ctx, wadPath, mapName)
	if err != nil {
		return domain.Level{}, err
	}
	return level, nil
}

func runPlayable(level domain.Level, tickRate int, width int, height int, zoom float64, topDownDebug bool) error {
	startX, startY, ok := level.PlayerStart()
	if !ok {
		startX = 0
		startY = 0
	}

	engine := domain.NewEngineAt(startX, startY)
	simulation, err := runtimeinfra.NewDomainSimulation(engine)
	if err != nil {
		return fmt.Errorf("create domain simulation: %w", err)
	}

	controller, err := playmap.New(
		converters.NewInputConverter(),
		simulation,
	)
	if err != nil {
		return fmt.Errorf("create playmap use case: %w", err)
	}

	inputPoller := inputinfra.NewEbitenPoller()
	var renderer ebitenplay.FrameRenderer
	if topDownDebug {
		renderer = renderinfra.NewTopDownRenderer(level, width, height, zoom)
	} else {
		renderer = renderinfra.NewFirstPersonRenderer(level, width, height, zoom)
	}
	game := ebitenplay.New(controller, inputPoller, renderer, engine.Frame())

	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle(fmt.Sprintf("go-doom | %s | W/S move, A/D turn, Q/E strafe, Esc quit", level.Name))
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetTPS(tickRate)

	runErr := ebiten.RunGame(game)
	if runErr != nil && !errors.Is(runErr, ebitenplay.ErrExitRequested) {
		return runErr
	}
	return nil
}

func estimateRuntimeTimeout(maxTicks int, tickRate int) time.Duration {
	if maxTicks <= 0 || tickRate <= 0 {
		return 30 * time.Second
	}

	ticksDuration := time.Duration(maxTicks) * time.Second / time.Duration(tickRate)
	return ticksDuration + 3*time.Second
}
