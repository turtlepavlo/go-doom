package playmap

import (
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type inputConverterStub struct {
	commands []domain.Command
}

func (s *inputConverterStub) ToCommands(raw []dto.RawInput) []domain.Command {
	return s.commands
}

type simulationStub struct {
	frame domain.Frame
	err   error
}

func (s *simulationStub) Step(commands []domain.Command) (domain.Frame, error) {
	if s.err != nil {
		return domain.Frame{}, s.err
	}
	return s.frame, nil
}

func TestUseCaseTick(t *testing.T) {
	useCase, err := New(
		&inputConverterStub{
			commands: []domain.Command{domain.CommandMoveForward},
		},
		&simulationStub{
			frame: domain.Frame{Tick: 1, PlayerY: -16, Running: true},
		},
	)
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	frame, err := useCase.Tick([]dto.RawInput{{Code: "W", Pressed: true}})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if frame.Tick != 1 || frame.PlayerY != -16 {
		t.Fatalf("unexpected frame: %+v", frame)
	}
}

func TestUseCaseTickWrapsErrors(t *testing.T) {
	useCase, err := New(
		&inputConverterStub{},
		&simulationStub{err: errors.New("boom")},
	)
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	_, tickErr := useCase.Tick(nil)
	if tickErr == nil {
		t.Fatal("expected error, got nil")
	}
}
