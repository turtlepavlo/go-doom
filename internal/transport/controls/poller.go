package controls

import (
	"context"

	"github.com/hajimehoshi/ebiten/v2"
)

type keyBinding struct {
	key  ebiten.Key
	code string
}

type ControlPoller struct {
	bindings []keyBinding
}

func NewControlPoller() *ControlPoller {
	return &ControlPoller{
		bindings: []keyBinding{
			{key: ebiten.KeyW, code: "W"},
			{key: ebiten.KeyS, code: "S"},
			{key: ebiten.KeyA, code: "A"},
			{key: ebiten.KeyD, code: "D"},
			{key: ebiten.KeyArrowUp, code: "ARROWUP"},
			{key: ebiten.KeyArrowDown, code: "ARROWDOWN"},
			{key: ebiten.KeyArrowLeft, code: "ARROWLEFT"},
			{key: ebiten.KeyArrowRight, code: "ARROWRIGHT"},
			{key: ebiten.KeyEscape, code: "ESC"},
			{key: ebiten.KeyQ, code: "Q"},
			{key: ebiten.KeyE, code: "E"},
			{key: ebiten.KeySpace, code: "FIRE"},
			{key: ebiten.KeyControlLeft, code: "FIRE"},
			{key: ebiten.KeyControlRight, code: "FIRE"},
		},
	}
}

func (p *ControlPoller) Poll(_ context.Context) ([]RawControl, error) {
	out := make([]RawControl, 0, len(p.bindings))
	for _, binding := range p.bindings {
		if !ebiten.IsKeyPressed(binding.key) {
			continue
		}

		out = append(out, RawControl{
			Code:    binding.code,
			Pressed: true,
		})
	}
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		out = append(out, RawControl{
			Code:    "MOUSE1",
			Pressed: true,
		})
	}
	return out, nil
}
