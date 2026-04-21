package service

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

// --- archive stubs ---

type archiveReaderStub struct {
	raw loadiwad.RawArchive
	err error
}

func (s *archiveReaderStub) ReadArchive(_ context.Context, _ string) (loadiwad.RawArchive, error) {
	if s.err != nil {
		return loadiwad.RawArchive{}, s.err
	}
	return s.raw, nil
}

type archiveConverterStub struct {
	archive domain.Archive
	err     error
}

func (s *archiveConverterStub) Archive(_ loadiwad.RawArchive) (domain.Archive, error) {
	if s.err != nil {
		return domain.Archive{}, s.err
	}
	return s.archive, nil
}

// --- map stubs ---

type mapReaderStub struct {
	raw loadmap.RawMap
	err error
}

func (s *mapReaderStub) ReadMap(_ context.Context, _, _ string) (loadmap.RawMap, error) {
	if s.err != nil {
		return loadmap.RawMap{}, s.err
	}
	return s.raw, nil
}

type mapConverterStub struct {
	level domain.Level
	err   error
}

func (s *mapConverterStub) Level(_ loadmap.RawMap) (domain.Level, error) {
	if s.err != nil {
		return domain.Level{}, s.err
	}
	return s.level, nil
}

// --- runtime stubs ---

type commandMapperStub struct {
	commands []domain.Command
}

func (s *commandMapperStub) ToCommands(_ context.Context, _ []controls.RawControl) []domain.Command {
	return s.commands
}

type controlPollerStub struct{}

func (s *controlPollerStub) Poll(_ context.Context) ([]controls.RawControl, error) {
	return []controls.RawControl{{Code: "RIGHT", Pressed: true}}, nil
}

type simulationStub struct{ tick int }

func (s *simulationStub) Step(_ context.Context, _ []domain.Command) (domain.Frame, error) {
	s.tick++
	return domain.Frame{Tick: uint64(s.tick), PlayerX: int64(s.tick), Running: s.tick < 3}, nil
}

type rendererStub struct{ frames int }

func (s *rendererStub) Render(_ context.Context, _ domain.Frame) error {
	s.frames++
	return nil
}

type timerStub struct{}

func (s *timerStub) Wait(_ context.Context) error { return nil }
func (s *timerStub) Close() error                 { return nil }

// --- helper ---

func newTestService(t *testing.T) *GameService {
	t.Helper()
	svc, err := New(
		&archiveReaderStub{},
		&archiveConverterStub{},
		&mapReaderStub{},
		&mapConverterStub{},
		&commandMapperStub{},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

// --- LoadIWAD ---

func TestLoadIWAD_Execute(t *testing.T) {
	expected := domain.Archive{Header: domain.Header{Kind: "IWAD", LumpCount: 1, DirectoryOffset: 12}}

	svc, err := New(
		&archiveReaderStub{raw: loadiwad.RawArchive{Type: "IWAD", LumpCount: 1, DirectoryOffset: 12}},
		&archiveConverterStub{archive: expected},
		&mapReaderStub{},
		&mapConverterStub{},
		&commandMapperStub{},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	got, err := svc.LoadIWAD(context.Background(), "doom1.wad")
	if err != nil {
		t.Fatalf("load iwad: %v", err)
	}
	if got.Header.Kind != "IWAD" {
		t.Fatalf("expected IWAD, got %q", got.Header.Kind)
	}
}

func TestLoadIWAD_EmptyPath(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.LoadIWAD(context.Background(), " ")
	if !errors.Is(err, ErrEmptyIWADPath) {
		t.Fatalf("expected ErrEmptyIWADPath, got %v", err)
	}
}

// --- LoadMap ---

func TestLoadMap_Execute(t *testing.T) {
	expected := domain.Level{Name: "E1M1"}

	svc, err := New(
		&archiveReaderStub{},
		&archiveConverterStub{},
		&mapReaderStub{raw: loadmap.RawMap{Name: "E1M1"}},
		&mapConverterStub{level: expected},
		&commandMapperStub{},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	level, err := svc.LoadMap(context.Background(), "doom1.wad", "E1M1")
	if err != nil {
		t.Fatalf("load map: %v", err)
	}
	if level.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", level.Name)
	}
}

func TestLoadMap_EmptyName(t *testing.T) {
	svc := newTestService(t)
	_, err := svc.LoadMap(context.Background(), "doom1.wad", "")
	if !errors.Is(err, ErrEmptyMapName) {
		t.Fatalf("expected ErrEmptyMapName, got %v", err)
	}
}

// --- RunGame ---

func TestRunGame_StopsOnGameState(t *testing.T) {
	sim := &simulationStub{}
	renderer := &rendererStub{}

	svc, err := New(
		&archiveReaderStub{},
		&archiveConverterStub{},
		&mapReaderStub{},
		&mapConverterStub{},
		&commandMapperStub{commands: []domain.Command{domain.CommandStrafeRight}},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if err := svc.RunGame(context.Background(), &controlPollerStub{}, sim, renderer, &timerStub{}, 10); err != nil {
		t.Fatalf("run game: %v", err)
	}
	if sim.tick != 3 {
		t.Fatalf("expected 3 ticks, got %d", sim.tick)
	}
	if renderer.frames != 3 {
		t.Fatalf("expected 3 rendered frames, got %d", renderer.frames)
	}
}

// --- Tick ---

func TestTick(t *testing.T) {
	svc, err := New(
		&archiveReaderStub{},
		&archiveConverterStub{},
		&mapReaderStub{},
		&mapConverterStub{},
		&commandMapperStub{commands: []domain.Command{domain.CommandMoveForward}},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	sim := &simulationStub{}
	frame, err := svc.Tick(context.Background(), sim, []controls.RawControl{{Code: "W", Pressed: true}})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if frame.Tick != 1 {
		t.Fatalf("expected tick 1, got %d", frame.Tick)
	}
}

func TestTick_WrapsSimulationError(t *testing.T) {
	svc := newTestService(t)

	errSim := &errorSimStub{err: errors.New("boom")}
	_, err := svc.Tick(context.Background(), errSim, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

type errorSimStub struct{ err error }

func (s *errorSimStub) Step(_ context.Context, _ []domain.Command) (domain.Frame, error) {
	return domain.Frame{}, s.err
}
