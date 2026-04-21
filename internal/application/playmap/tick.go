package playmap

import (
	"context"
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/domain"
	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

func (svc *Service) Tick(ctx context.Context, raw []controls.RawControl) (domain.Frame, error) {
	commands := svc.mapper.ToCommands(ctx, raw)
	frame, err := svc.simulation.Step(ctx, commands)
	if err != nil {
		return domain.Frame{}, fmt.Errorf("simulate frame: %w", err)
	}
	return frame, nil
}
