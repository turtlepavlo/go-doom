package service

import (
	"context"
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

func (s *GameService) Tick(ctx context.Context, sim Simulation, raw []controls.RawControl) (domain.Frame, error) {
	commands := s.commandMapper.ToCommands(ctx, raw)
	frame, err := sim.Step(ctx, commands)
	if err != nil {
		return domain.Frame{}, fmt.Errorf("simulate frame: %w", err)
	}
	return frame, nil
}
