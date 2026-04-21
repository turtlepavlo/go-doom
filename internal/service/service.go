package service

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

var (
	ErrNilArchiveReader    = errors.New("nil archive reader")
	ErrNilArchiveConverter = errors.New("nil archive converter")
	ErrNilMapReader        = errors.New("nil map reader")
	ErrNilMapConverter     = errors.New("nil map converter")
	ErrNilCommandMapper    = errors.New("nil command mapper")

	ErrEmptyIWADPath = errors.New("empty IWAD path")
	ErrEmptyWADPath  = errors.New("empty WAD path")
	ErrEmptyMapName  = errors.New("empty map name")
	ErrNegativeTicks = errors.New("negative max ticks")
)

// Storage interfaces ??? implemented by internal/storage/wad.

type ArchiveReader interface {
	ReadArchive(ctx context.Context, path string) (loadiwad.RawArchive, error)
}

type ArchiveConverter interface {
	Archive(raw loadiwad.RawArchive) (domain.Archive, error)
}

type MapReader interface {
	ReadMap(ctx context.Context, wadPath string, mapName string) (loadmap.RawMap, error)
}

type MapConverter interface {
	Level(raw loadmap.RawMap) (domain.Level, error)
}

// Runtime interfaces ??? passed per-call since simulation depends on level data.

type CommandMapper interface {
	ToCommands(ctx context.Context, raw []controls.RawControl) []domain.Command
}

type Simulation interface {
	Step(ctx context.Context, commands []domain.Command) (domain.Frame, error)
}

type ControlPoller interface {
	Poll(ctx context.Context) ([]controls.RawControl, error)
}

type HeadlessRenderer interface {
	Render(ctx context.Context, frame domain.Frame) error
}

type StepTimer interface {
	Wait(ctx context.Context) error
	Close() error
}

// GameService coordinates WAD asset loading and game session execution.
type GameService struct {
	archiveReader    ArchiveReader
	archiveConverter ArchiveConverter
	mapReader        MapReader
	mapConverter     MapConverter
	commandMapper    CommandMapper
}

func New(
	archiveReader ArchiveReader,
	archiveConverter ArchiveConverter,
	mapReader MapReader,
	mapConverter MapConverter,
	commandMapper CommandMapper,
) (*GameService, error) {
	switch {
	case archiveReader == nil:
		return nil, ErrNilArchiveReader
	case archiveConverter == nil:
		return nil, ErrNilArchiveConverter
	case mapReader == nil:
		return nil, ErrNilMapReader
	case mapConverter == nil:
		return nil, ErrNilMapConverter
	case commandMapper == nil:
		return nil, ErrNilCommandMapper
	}

	return &GameService{
		archiveReader:    archiveReader,
		archiveConverter: archiveConverter,
		mapReader:        mapReader,
		mapConverter:     mapConverter,
		commandMapper:    commandMapper,
	}, nil
}
