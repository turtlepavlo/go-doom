package render

import (
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestLayout(t *testing.T) {
	level, err := domain.NewLevel("E1M1", nil, nil, nil, []domain.Vertex{
		{X: 0, Y: 0},
		{X: 128, Y: 128},
	}, nil)
	if err != nil {
		t.Fatalf("new level: %v", err)
	}

	renderer := NewTopDownRenderer(level, 800, 600, 1)
	w, h := renderer.Layout()
	if w != 800 || h != 600 {
		t.Fatalf("expected layout 800x600, got %dx%d", w, h)
	}
}

func TestFitScaleNonZero(t *testing.T) {
	level, err := domain.NewLevel("E1M1", nil, nil, nil, []domain.Vertex{
		{X: -64, Y: -64},
		{X: 64, Y: 64},
	}, nil)
	if err != nil {
		t.Fatalf("new level: %v", err)
	}

	scale := fitScale(level, 640, 480)
	if scale <= 0 {
		t.Fatalf("expected positive scale, got %f", scale)
	}
}
