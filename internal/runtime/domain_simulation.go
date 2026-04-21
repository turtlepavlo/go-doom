package runtime

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

var ErrNilEngine = errors.New("nil domain engine")

type DomainSimulation struct {
	engine *Engine
}

func NewDomainSimulation(engine *Engine) (*DomainSimulation, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	return &DomainSimulation{engine: engine}, nil
}

func (s *DomainSimulation) Step(ctx context.Context, commands []domain.Command) (domain.Frame, error) {
	return s.engine.Step(ctx, commands), nil
}
