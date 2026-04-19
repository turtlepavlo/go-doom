package playmap

import (
	"errors"
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilInputConverter = errors.New("nil input converter")
	ErrNilSimulation     = errors.New("nil simulation")
)

type InputConverter interface {
	ToCommands(raw []dto.RawInput) []domain.Command
}

type Simulation interface {
	Step(commands []domain.Command) (domain.Frame, error)
}

type UseCase struct {
	inputConverter InputConverter
	simulation     Simulation
}

func New(inputConverter InputConverter, simulation Simulation) (*UseCase, error) {
	switch {
	case inputConverter == nil:
		return nil, ErrNilInputConverter
	case simulation == nil:
		return nil, ErrNilSimulation
	}

	return &UseCase{
		inputConverter: inputConverter,
		simulation:     simulation,
	}, nil
}

func (u *UseCase) Tick(raw []dto.RawInput) (domain.Frame, error) {
	commands := u.inputConverter.ToCommands(raw)
	frame, err := u.simulation.Step(commands)
	if err != nil {
		return domain.Frame{}, fmt.Errorf("simulate frame: %w", err)
	}
	return frame, nil
}
