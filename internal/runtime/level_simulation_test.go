package runtime

import (
	"context"
	"math"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestNewLevelSimulationNilLevel(t *testing.T) {
	engine := NewEngine()
	_, err := NewLevelSimulation(engine, nil)
	if err != ErrNilLevel {
		t.Fatalf("expected ErrNilLevel, got %v", err)
	}
}

func TestLevelSimulationCollisionBlocksForward(t *testing.T) {
	level := buildCollisionTestLevel(t)
	engine := NewEngineAt(0, 0)

	sim, err := NewLevelSimulation(engine, &level)
	if err != nil {
		t.Fatalf("new level simulation: %v", err)
	}

	frame, err := sim.Step(context.Background(), []domain.Command{domain.CommandMoveForward})
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

func TestLevelSimulationFireCanHitEnemy(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Things: []domain.Thing{
			{X: 120, Y: 0, Type: 3004},
		},
	}

	engine := NewEnginePose(0, 0, 0)
	sim, err := NewLevelSimulation(engine, &level)
	if err != nil {
		t.Fatalf("new level simulation: %v", err)
	}

	frame, err := sim.Step(context.Background(), []domain.Command{domain.CommandFire})
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	if frame.ShotsFired != 1 || frame.ShotHits != 1 {
		t.Fatalf("expected one successful shot, got shots=%d hits=%d", frame.ShotsFired, frame.ShotHits)
	}
	if frame.Kills != 1 || frame.EnemyAlive != 0 {
		t.Fatalf("expected enemy kill, got kills=%d alive=%d", frame.Kills, frame.EnemyAlive)
	}
	if frame.Ammo != 59 {
		t.Fatalf("expected ammo=59, got %d", frame.Ammo)
	}
}

func TestLevelSimulationEnemyCanDamagePlayer(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Things: []domain.Thing{
			{X: 40, Y: 0, Type: 3002},
		},
	}

	engine := NewEnginePose(0, 0, 0)
	sim, err := NewLevelSimulation(engine, &level)
	if err != nil {
		t.Fatalf("new level simulation: %v", err)
	}

	frame, err := sim.Step(context.Background(), nil)
	if err != nil {
		t.Fatalf("step: %v", err)
	}
	if frame.Health >= 100 {
		t.Fatalf("expected health to drop below 100, got %d", frame.Health)
	}
}

func TestIsBlockingLine(t *testing.T) {
	sector := domain.Sector{FloorHeight: 0, CeilingHeight: 128}
	side := domain.Sidedef{Sector: 0}
	level := domain.Level{
		Sectors:  []domain.Sector{sector},
		Sidedefs: []domain.Sidedef{side},
	}

	t.Run("blocking flag", func(t *testing.T) {
		line := domain.Linedef{Flags: linedefFlagBlocking, RightSide: 0, LeftSide: 0}
		if !isBlockingLine(level, line) {
			t.Error("expected line with blocking flag to be blocking")
		}
	})

	t.Run("one sided", func(t *testing.T) {
		line := domain.Linedef{RightSide: 0, LeftSide: math.MaxUint16}
		if !isBlockingLine(level, line) {
			t.Error("expected one-sided line to be blocking")
		}
	})

	t.Run("low ceiling", func(t *testing.T) {
		lowSector := domain.Sector{FloorHeight: 0, CeilingHeight: 32}
		levelWithLow := domain.Level{
			Sectors:  []domain.Sector{sector, lowSector},
			Sidedefs: []domain.Sidedef{{Sector: 0}, {Sector: 1}},
		}
		line := domain.Linedef{RightSide: 0, LeftSide: 1}
		if !isBlockingLine(levelWithLow, line) {
			t.Error("expected line with low ceiling to be blocking")
		}
	})

	t.Run("passable", func(t *testing.T) {
		highSector := domain.Sector{FloorHeight: 0, CeilingHeight: 128}
		levelPassable := domain.Level{
			Sectors:  []domain.Sector{sector, highSector},
			Sidedefs: []domain.Sidedef{{Sector: 0}, {Sector: 1}},
		}
		line := domain.Linedef{RightSide: 0, LeftSide: 1}
		if isBlockingLine(levelPassable, line) {
			t.Error("expected two-sided line with enough height to be passable")
		}
	})
}

func buildCollisionTestLevel(t *testing.T) domain.Level {
	t.Helper()

	return domain.Level{
		Name: "E1M1",
		Linedefs: []domain.Linedef{
			// one-sided blocking vertical wall x=32
			{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
		},
		Sidedefs: []domain.Sidedef{
			{Sector: 0},
		},
		Vertexes: []domain.Vertex{
			{X: 32, Y: -64},
			{X: 32, Y: 64},
		},
		Sectors: []domain.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
	}
}
