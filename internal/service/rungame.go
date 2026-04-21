package service

import (
	"context"
	"errors"
	"fmt"
)

func (s *GameService) RunGame(
	ctx context.Context,
	poller ControlPoller,
	sim Simulation,
	renderer HeadlessRenderer,
	timer StepTimer,
	maxTicks int,
) (runErr error) {
	if maxTicks < 0 {
		return ErrNegativeTicks
	}

	defer func() {
		if err := timer.Close(); err != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close timer: %w", err))
		}
	}()

	ticksRun := 0
	for maxTicks == 0 || ticksRun < maxTicks {
		if err := timer.Wait(ctx); err != nil {
			return fmt.Errorf("wait timer: %w", err)
		}

		rawControls, err := poller.Poll(ctx)
		if err != nil {
			return fmt.Errorf("poll controls: %w", err)
		}

		commands := s.commandMapper.ToCommands(ctx, rawControls)
		frame, err := sim.Step(ctx, commands)
		if err != nil {
			return fmt.Errorf("simulate frame: %w", err)
		}

		if err := renderer.Render(ctx, frame); err != nil {
			return fmt.Errorf("render frame: %w", err)
		}

		ticksRun++
		if !frame.Running {
			break
		}
	}

	return nil
}
