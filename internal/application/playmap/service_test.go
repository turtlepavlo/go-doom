package playmap

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

type commandMapperStub struct {
	commands []domain.Command
}

func (s *commandMapperStub) ToCommands(_ context.Context, _ []controls.RawControl) []domain.Command {
	return s.commands
}

type simulationStub struct {
	frame domain.Frame
	err   error
}

func (s *simulationStub) Step(_ context.Context, _ []domain.Command) (domain.Frame, error) {
	if s.err != nil {
		return domain.Frame{}, s.err
	}
	return s.frame, nil
}

func TestServiceTick(t *testing.T) {
	service, err := New(
		&commandMapperStub{
			commands: []domain.Command{domain.CommandMoveForward},
		},
		&simulationStub{
			frame: domain.Frame{Tick: 1, PlayerY: -16, Running: true},
		},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	frame, err := service.Tick(context.Background(), []controls.RawControl{{Code: "W", Pressed: true}})
	if err != nil {
		t.Fatalf("tick: %v", err)
	}
	if frame.Tick != 1 || frame.PlayerY != -16 {
		t.Fatalf("unexpected frame: %+v", frame)
	}
}

func TestServiceTickWrapsErrors(t *testing.T) {
	service, err := New(
		&commandMapperStub{},
		&simulationStub{err: errors.New("boom")},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, tickErr := service.Tick(context.Background(), nil)
	if tickErr == nil {
		t.Fatal("expected error, got nil")
	}
}
