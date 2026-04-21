package rungame

import (
	"context"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

type testControlsPoller struct{}

func (t *testControlsPoller) Poll(ctx context.Context) ([]controls.RawControl, error) {
	return []controls.RawControl{{Code: "RIGHT", Pressed: true}}, nil
}

type testCommandMapper struct{}

func (t *testCommandMapper) ToCommands(_ context.Context, _ []controls.RawControl) []domain.Command {
	return []domain.Command{domain.CommandStrafeRight}
}

type testSimulation struct {
	tick int
}

func (t *testSimulation) Step(_ context.Context, _ []domain.Command) (domain.Frame, error) {
	t.tick++
	return domain.Frame{
		Tick:    uint64(t.tick),
		PlayerX: int64(t.tick),
		PlayerY: 0,
		Running: t.tick < 3,
	}, nil
}

type testRenderer struct {
	frames int
}

func (t *testRenderer) Render(ctx context.Context, frame domain.Frame) error {
	t.frames++
	return nil
}

type testTimer struct {
	waits int
}

func (t *testTimer) Wait(ctx context.Context) error {
	t.waits++
	return nil
}

func (t *testTimer) Close() error {
	return nil
}

func TestService_Run_StopsOnGameState(t *testing.T) {
	sim := &testSimulation{}
	renderer := &testRenderer{}
	timer := &testTimer{}

	service, err := New(&testControlsPoller{}, &testCommandMapper{}, sim, renderer, timer)
	if err != nil {
		t.Fatalf("unexpected bootstrap error: %v", err)
	}

	if err := service.Run(context.Background(), 10); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if sim.tick != 3 {
		t.Fatalf("expected 3 simulated ticks, got %d", sim.tick)
	}
	if renderer.frames != 3 {
		t.Fatalf("expected 3 rendered frames, got %d", renderer.frames)
	}
}
