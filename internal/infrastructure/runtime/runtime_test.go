package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestNewDomainSimulation_NilEngine(t *testing.T) {
	_, err := NewDomainSimulation(nil)
	if !errors.Is(err, ErrNilEngine) {
		t.Fatalf("expected ErrNilEngine, got %v", err)
	}
}

func TestFixedTimer_CloseThenWait(t *testing.T) {
	timer, err := NewFixedTimer(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := timer.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}

	waitErr := timer.Wait(context.Background())
	if !errors.Is(waitErr, ErrTimerClosed) {
		t.Fatalf("expected ErrTimerClosed, got %v", waitErr)
	}
}

func TestHeadlessRenderer_Stats(t *testing.T) {
	renderer := NewHeadlessRenderer(nil)
	frame := domain.Frame{
		Tick:    5,
		PlayerX: 2,
		PlayerY: -1,
		Running: true,
	}

	if err := renderer.Render(context.Background(), frame); err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	frames, gotFrame := renderer.Stats()
	if frames != 1 {
		t.Fatalf("expected 1 rendered frame, got %d", frames)
	}
	if gotFrame.Tick != 5 || gotFrame.PlayerX != 2 || gotFrame.PlayerY != -1 {
		t.Fatalf("unexpected frame stats: %+v", gotFrame)
	}
}
