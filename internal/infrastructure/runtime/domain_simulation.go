package runtime

import (
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

var ErrNilEngine = errors.New("nil domain engine")

type DomainSimulation struct {
	engine *domain.Engine
}

func NewDomainSimulation(engine *domain.Engine) (*DomainSimulation, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	return &DomainSimulation{engine: engine}, nil
}

func (s *DomainSimulation) Step(commands []domain.Command) (domain.Frame, error) {
	return s.engine.Step(commands), nil
}
