package rungame

import (
	"context"
	"errors"
	"fmt"
)

func (svc *Service) Run(ctx context.Context, maxTicks int) (runErr error) {
	if maxTicks < 0 {
		return ErrNegativeTicks
	}

	defer func() {
		closeErr := svc.timer.Close()
		if closeErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close timer: %w", closeErr))
		}
	}()

	ticksRun := 0
	for maxTicks == 0 || ticksRun < maxTicks {
		if err := svc.timer.Wait(ctx); err != nil {
			return fmt.Errorf("wait step timer: %w", err)
		}

		rawControls, err := svc.poller.Poll(ctx)
		if err != nil {
			return fmt.Errorf("poll controls: %w", err)
		}

		commands := svc.mapper.ToCommands(ctx, rawControls)
		frame, err := svc.simulation.Step(ctx, commands)
		if err != nil {
			return fmt.Errorf("simulate frame: %w", err)
		}

		if err := svc.renderer.Render(ctx, frame); err != nil {
			return fmt.Errorf("render frame: %w", err)
		}

		ticksRun++
		if !frame.Running {
			break
		}
	}

	return nil
}
