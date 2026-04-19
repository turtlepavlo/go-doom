package converters

import (
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type InputConverter struct{}

func NewInputConverter() *InputConverter {
	return &InputConverter{}
}

func (c *InputConverter) ToCommands(raw []dto.RawInput) []domain.Command {
	commands := make([]domain.Command, 0, len(raw))

	for _, input := range raw {
		if !input.Pressed {
			continue
		}

		code := strings.ToUpper(strings.TrimSpace(input.Code))
		switch code {
		case "W", "UP", "ARROWUP":
			commands = append(commands, domain.CommandMoveForward)
		case "S", "DOWN", "ARROWDOWN":
			commands = append(commands, domain.CommandMoveBackward)
		case "A", "LEFT", "ARROWLEFT":
			commands = append(commands, domain.CommandStrafeLeft)
		case "D", "RIGHT", "ARROWRIGHT":
			commands = append(commands, domain.CommandStrafeRight)
		case "ESC", "ESCAPE", "Q":
			commands = append(commands, domain.CommandQuit)
		}
	}

	return commands
}
