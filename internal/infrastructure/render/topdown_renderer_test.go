package render

import (
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestLayout(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Vertexes: []domain.Vertex{
			{X: 0, Y: 0},
			{X: 128, Y: 128},
		},
	}

	renderer := NewTopDownRenderer(level, 800, 600, 1)
	w, h := renderer.Layout()
	if w != 800 || h != 600 {
		t.Fatalf("expected layout 800x600, got %dx%d", w, h)
	}
}

func TestFitScaleNonZero(t *testing.T) {
	level := domain.Level{
		Name: "E1M1",
		Vertexes: []domain.Vertex{
			{X: -64, Y: -64},
			{X: 64, Y: 64},
		},
	}

	scale := fitScale(level, 640, 480)
	if scale <= 0 {
		t.Fatalf("expected positive scale, got %f", scale)
	}
}
