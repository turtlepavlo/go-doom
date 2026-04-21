package controls

import (
	"context"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type CommandMapper struct{}

func NewCommandMapper() *CommandMapper {
	return &CommandMapper{}
}

func (c *CommandMapper) ToCommands(_ context.Context, raw []RawControl) []domain.Command {
	commands := make([]domain.Command, 0, len(raw))

	for _, control := range raw {
		if !control.Pressed {
			continue
		}

		code := strings.ToUpper(strings.TrimSpace(control.Code))
		switch code {
		case "W", "UP", "ARROWUP":
			commands = append(commands, domain.CommandMoveForward)
		case "S", "DOWN", "ARROWDOWN":
			commands = append(commands, domain.CommandMoveBackward)
		case "A", "LEFT", "ARROWLEFT":
			commands = append(commands, domain.CommandTurnLeft)
		case "D", "RIGHT", "ARROWRIGHT":
			commands = append(commands, domain.CommandTurnRight)
		case "Q":
			commands = append(commands, domain.CommandStrafeLeft)
		case "E":
			commands = append(commands, domain.CommandStrafeRight)
		case "SPACE", "CTRL", "CONTROL", "F", "FIRE", "MOUSE1":
			commands = append(commands, domain.CommandFire)
		case "ESC", "ESCAPE":
			commands = append(commands, domain.CommandQuit)
		}
	}

	return commands
}
