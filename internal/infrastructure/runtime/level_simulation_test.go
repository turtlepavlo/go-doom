package runtime

import (
	"math"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestNewLevelSimulationNilLevel(t *testing.T) {
	engine := domain.NewEngine()
	_, err := NewLevelSimulation(engine, nil)
	if err != ErrNilLevel {
		t.Fatalf("expected ErrNilLevel, got %v", err)
	}
}

func TestLevelSimulationCollisionBlocksForward(t *testing.T) {
	level := buildCollisionTestLevel(t)
	engine := domain.NewEngineAt(0, 0)

	sim, err := NewLevelSimulation(engine, &level)
	if err != nil {
		t.Fatalf("new level simulation: %v", err)
	}

	frame, err := sim.Step([]domain.Command{domain.CommandMoveForward})
	if err != nil {
		t.Fatalf("step: %v", err)
	}

	// movement along +X at angle 0 should hit the blocking line at x=32
	if frame.PlayerX != 0 || frame.PlayerY != 0 {
		t.Fatalf("expected blocked movement at origin, got (%d,%d)", frame.PlayerX, frame.PlayerY)
	}
}

func TestPointSegmentDistance(t *testing.T) {
	d := pointSegmentDistance(10, 10, 0, 0, 20, 0)
	if math.Abs(d-10) > 0.001 {
		t.Fatalf("expected distance ~10, got %f", d)
	}
}

func buildCollisionTestLevel(t *testing.T) domain.Level {
	t.Helper()

	level, err := domain.NewLevel(
		"E1M1",
		nil,
		[]domain.Linedef{
			// one-sided blocking vertical wall x=32
			{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
		},
		[]domain.Sidedef{
			{Sector: 0},
		},
		[]domain.Vertex{
			{X: 32, Y: -64},
			{X: 32, Y: 64},
		},
		[]domain.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
	)
	if err != nil {
		t.Fatalf("new level: %v", err)
	}
	return level
}
