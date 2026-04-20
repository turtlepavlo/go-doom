package domain

import (
	"errors"
	"math"
	"testing"
)

func TestNewLevel(t *testing.T) {
	level, err := NewLevel("E1M1", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if level.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", level.Name)
	}
}

func TestNewLevelEmptyName(t *testing.T) {
	_, err := NewLevel("", nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrEmptyMapName) {
		t.Fatalf("expected ErrEmptyMapName, got %v", err)
	}
}

func TestPlayerStart(t *testing.T) {
	level, err := NewLevel("E1M1", []Thing{
		{X: 40, Y: 10, Type: 3004},
		{X: 128, Y: -64, Type: 1, Angle: 90},
	}, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("new level: %v", err)
	}

	x, y, ok := level.PlayerStart()
	if !ok {
		t.Fatal("expected player start")
	}
	if x != 128 || y != -64 {
		t.Fatalf("expected player start at (128,-64), got (%d,%d)", x, y)
	}

	spawn, ok := level.PlayerSpawn()
	if !ok {
		t.Fatal("expected player spawn")
	}
	if spawn.X != 128 || spawn.Y != -64 {
		t.Fatalf("unexpected spawn position %+v", spawn)
	}
	if math.Abs(spawn.Angle-math.Pi/2) > 0.001 {
		t.Fatalf("expected angle around PI/2, got %f", spawn.Angle)
	}
}
