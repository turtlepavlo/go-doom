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
		{Code: "esc", Pressed: true},
		{Code: "left", Pressed: false},
	}

	commands := converter.ToCommands(raw)
	if len(commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(commands))
	}
	if commands[0] != domain.CommandMoveForward {
		t.Fatalf("expected move forward, got %s", commands[0])
	}
	if commands[1] != domain.CommandStrafeRight {
		t.Fatalf("expected strafe right, got %s", commands[1])
	}
	if commands[2] != domain.CommandQuit {
		t.Fatalf("expected quit, got %s", commands[2])
	}
}
