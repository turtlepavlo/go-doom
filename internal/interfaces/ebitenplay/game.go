package ebitenplay

import (
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var ErrExitRequested = errors.New("exit requested")

type Ticker interface {
	Tick(raw []dto.RawInput) (domain.Frame, error)
}

type InputPoller interface {
	Poll() []dto.RawInput
}

type FrameRenderer interface {
	Draw(screen *ebiten.Image, frame domain.Frame)
	Layout() (int, int)
}

type Game struct {
	ticker    Ticker
	input     InputPoller
	renderer  FrameRenderer
	lastFrame domain.Frame
}

func New(
	ticker Ticker,
	input InputPoller,
	renderer FrameRenderer,
	initialFrame domain.Frame,
) *Game {
	return &Game{
		ticker:    ticker,
		input:     input,
		renderer:  renderer,
		lastFrame: initialFrame,
	}
}

func (g *Game) Update() error {
	raw := g.input.Poll()
	frame, err := g.ticker.Tick(raw)
	if err != nil {
		return err
	}

	g.lastFrame = frame
	if !frame.Running {
		return ErrExitRequested
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.renderer.Draw(screen, g.lastFrame)
}

func (g *Game) Layout(outsideWidth int, outsideHeight int) (int, int) {
	return g.renderer.Layout()
}
