package converters

import (
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

func TestInputConverter_ToCommands(t *testing.T) {
	converter := NewInputConverter()
	raw := []dto.RawInput{
		{Code: "w", Pressed: true},
		{Code: "ARROWRIGHT", Pressed: true},
		{Code: "q", Pressed: true},
		{Code: "mouse1", Pressed: true},
		{Code: "esc", Pressed: true},
		{Code: "left", Pressed: false},
	}

	commands := converter.ToCommands(raw)
	if len(commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(commands))
	}
	if commands[0] != domain.CommandMoveForward {
		t.Fatalf("expected move forward, got %s", commands[0])
	}
	if commands[1] != domain.CommandTurnRight {
		t.Fatalf("expected turn right, got %s", commands[1])
	}
	if commands[2] != domain.CommandStrafeLeft {
		t.Fatalf("expected strafe left, got %s", commands[2])
	}
	if commands[3] != domain.CommandFire {
		t.Fatalf("expected fire, got %s", commands[3])
	}
	if commands[4] != domain.CommandQuit {
		t.Fatalf("expected quit, got %s", commands[4])
	}
}
