package playmap

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

var (
	ErrNilCommandMapper = errors.New("nil command mapper")
	ErrNilSimulation    = errors.New("nil simulation")
)

type CommandMapper interface {
	ToCommands(ctx context.Context, raw []controls.RawControl) []domain.Command
}

type Simulation interface {
	Step(ctx context.Context, commands []domain.Command) (domain.Frame, error)
}

type Service struct {
	mapper     CommandMapper
	simulation Simulation
}

func New(mapper CommandMapper, simulation Simulation) (*Service, error) {
	switch {
	case mapper == nil:
		return nil, ErrNilCommandMapper
	case simulation == nil:
		return nil, ErrNilSimulation
	}

	return &Service{
		mapper:     mapper,
		simulation: simulation,
	}, nil
}
