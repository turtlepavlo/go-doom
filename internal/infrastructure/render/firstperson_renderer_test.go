package render

import (
	"math"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestFirstPersonRendererLayout(t *testing.T) {
	level, err := domain.NewLevel("E1M1", nil, []domain.Linedef{
		{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
	}, []domain.Sidedef{
		{Sector: 0},
	}, []domain.Vertex{
		{X: 0, Y: 0},
		{X: 64, Y: 0},
	}, []domain.Sector{
		{FloorHeight: 0, CeilingHeight: 128},
	})
	if err != nil {
		t.Fatalf("new level: %v", err)
	}

	renderer := NewFirstPersonRenderer(level, 1024, 576, 1)
	w, h := renderer.Layout()
	if w != 1024 || h != 576 {
		t.Fatalf("expected layout 1024x576, got %dx%d", w, h)
	}
}

func TestCollectSolidWalls(t *testing.T) {
	level, err := domain.NewLevel("E1M1", nil, []domain.Linedef{
		{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
		{StartVertex: 1, EndVertex: 2, RightSide: 1, LeftSide: 2},
	}, []domain.Sidedef{
		{Sector: 0},
		{Sector: 0},
		{Sector: 0},
	}, []domain.Vertex{
		{X: 0, Y: 0},
		{X: 64, Y: 0},
		{X: 64, Y: 64},
	}, []domain.Sector{
		{FloorHeight: 0, CeilingHeight: 128},
	})
	if err != nil {
		t.Fatalf("new level: %v", err)
	}

	walls := collectSolidWalls(level)
	if len(walls) != 1 {
		t.Fatalf("expected 1 one-sided wall, got %d", len(walls))
	}
}
