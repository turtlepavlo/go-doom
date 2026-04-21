package ebitenplay

import (
	"context"
	"errors"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

var ErrExitRequested = errors.New("exit requested")

type Ticker interface {
	Tick(ctx context.Context, raw []controls.RawControl) (domain.Frame, error)
}

type ControlPoller interface {
	Poll(ctx context.Context) ([]controls.RawControl, error)
}

type FrameRenderer interface {
	Draw(screen *ebiten.Image, frame domain.Frame)
	Layout() (int, int)
}

type Game struct {
	ctx       context.Context
	ticker    Ticker
	controls  ControlPoller
	renderer  FrameRenderer
	lastFrame domain.Frame
}

func New(
	ctx context.Context,
	ticker Ticker,
	controls ControlPoller,
	renderer FrameRenderer,
	initialFrame domain.Frame,
) *Game {
	return &Game{
		ctx:       ctx,
		ticker:    ticker,
		controls:  controls,
		renderer:  renderer,
		lastFrame: initialFrame,
	}
}

func (g *Game) Update() error {
	if g.ctx != nil {
		select {
		case <-g.ctx.Done():
			return ErrExitRequested
		default:
		}
	}

	raw, err := g.controls.Poll(g.ctx)
	if err != nil {
		return err
	}

	frame, err := g.ticker.Tick(g.ctx, raw)
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
