package render

import (
	"math"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestFirstPersonRendererLayout(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Linedefs: []domain.Linedef{
			{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
		},
		Sidedefs: []domain.Sidedef{
			{Sector: 0},
		},
		Vertexes: []domain.Vertex{
			{X: 0, Y: 0},
			{X: 64, Y: 0},
		},
		Sectors: []domain.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
	}

	renderer := NewFirstPersonRenderer(level, 1024, 576, 1)
	w, h := renderer.Layout()
	if w != 1024 || h != 576 {
		t.Fatalf("expected layout 1024x576, got %dx%d", w, h)
	}
}

func TestCollectRenderableWalls(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Linedefs: []domain.Linedef{
			{StartVertex: 0, EndVertex: 1, RightSide: math.MaxUint16, LeftSide: 0},
			{StartVertex: 1, EndVertex: 2, RightSide: 1, LeftSide: 2},
		},
		Sidedefs: []domain.Sidedef{
			{Sector: 0},
			{Sector: 0},
			{Sector: 0},
		},
		Vertexes: []domain.Vertex{
			{X: 0, Y: 0},
			{X: 64, Y: 0},
			{X: 64, Y: 64},
		},
		Sectors: []domain.Sector{
			{FloorHeight: 0, CeilingHeight: 128},
		},
	}

	walls := collectRenderableWalls(level)
	if len(walls) != 1 {
		t.Fatalf("expected 1 renderable wall, got %d", len(walls))
	}
}
