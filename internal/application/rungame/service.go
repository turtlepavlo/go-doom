package rungame

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

var (
	ErrNilControlPoller = errors.New("nil control poller")
	ErrNilCommandMapper = errors.New("nil command mapper")
	ErrNilSimulation    = errors.New("nil simulation")
	ErrNilRenderer      = errors.New("nil renderer")
	ErrNilTimer         = errors.New("nil timer")
	ErrNegativeTicks    = errors.New("negative max ticks")
)

type ControlPoller interface {
	Poll(ctx context.Context) ([]controls.RawControl, error)
}

type CommandMapper interface {
	ToCommands(ctx context.Context, raw []controls.RawControl) []domain.Command
}

type Simulation interface {
	Step(ctx context.Context, commands []domain.Command) (domain.Frame, error)
}

type Renderer interface {
	Render(ctx context.Context, frame domain.Frame) error
}

type StepTimer interface {
	Wait(ctx context.Context) error
	Close() error
}

type Service struct {
	poller     ControlPoller
	mapper     CommandMapper
	simulation Simulation
	renderer   Renderer
	timer      StepTimer
}

func New(
	poller ControlPoller,
	mapper CommandMapper,
	simulation Simulation,
	renderer Renderer,
	timer StepTimer,
) (*Service, error) {
	switch {
	case poller == nil:
		return nil, ErrNilControlPoller
	case mapper == nil:
		return nil, ErrNilCommandMapper
	case simulation == nil:
		return nil, ErrNilSimulation
	case renderer == nil:
		return nil, ErrNilRenderer
	case timer == nil:
		return nil, ErrNilTimer
	}

	return &Service{
		poller:     poller,
		mapper:     mapper,
		simulation: simulation,
		renderer:   renderer,
		timer:      timer,
	}, nil
}
