package input

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/application/dto"
)

type keyBinding struct {
	key  ebiten.Key
	code string
}

type EbitenPoller struct {
	bindings []keyBinding
}

func NewEbitenPoller() *EbitenPoller {
	return &EbitenPoller{
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
		},
	}
}

func (p *EbitenPoller) Poll() []dto.RawInput {
	out := make([]dto.RawInput, 0, len(p.bindings))
	for _, binding := range p.bindings {
		if !ebiten.IsKeyPressed(binding.key) {
			continue
		}

		out = append(out, dto.RawInput{
			Code:    binding.code,
			Pressed: true,
		})
	}
	return out
}
