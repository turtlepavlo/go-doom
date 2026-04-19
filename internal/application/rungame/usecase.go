package rungame

import (
	"context"
	"errors"
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilInputPoller = errors.New("nil input poller")
	ErrNilConverter   = errors.New("nil input converter")
	ErrNilSimulation  = errors.New("nil simulation")
	ErrNilRenderer    = errors.New("nil renderer")
	ErrNilTimer       = errors.New("nil timer")
	ErrNegativeTicks  = errors.New("negative max ticks")
)

type InputPoller interface {
	Poll(ctx context.Context) ([]dto.RawInput, error)
}

type InputConverter interface {
	ToCommands(raw []dto.RawInput) []domain.Command
}

type Simulation interface {
	Step(commands []domain.Command) (domain.Frame, error)
}

type Renderer interface {
	Render(ctx context.Context, frame domain.Frame) error
}

type StepTimer interface {
	Wait(ctx context.Context) error
	Close() error
}

type UseCase struct {
	input      InputPoller
	converter  InputConverter
	simulation Simulation
	renderer   Renderer
	timer      StepTimer
}

func New(input InputPoller, converter InputConverter, simulation Simulation, renderer Renderer, timer StepTimer) (*UseCase, error) {
	switch {
	case input == nil:
		return nil, ErrNilInputPoller
	case converter == nil:
		return nil, ErrNilConverter
	case simulation == nil:
		return nil, ErrNilSimulation
	case renderer == nil:
		return nil, ErrNilRenderer
	case timer == nil:
		return nil, ErrNilTimer
	}

	return &UseCase{
		input:      input,
		converter:  converter,
		simulation: simulation,
		renderer:   renderer,
		timer:      timer,
	}, nil
}

func (u *UseCase) Run(ctx context.Context, maxTicks int) (runErr error) {
	if maxTicks < 0 {
		return ErrNegativeTicks
	}

	defer func() {
		closeErr := u.timer.Close()
		if closeErr != nil {
			runErr = errors.Join(runErr, fmt.Errorf("close timer: %w", closeErr))
		}
	}()

	ticksRun := 0
	for maxTicks == 0 || ticksRun < maxTicks {
		if err := u.timer.Wait(ctx); err != nil {
			return fmt.Errorf("wait step timer: %w", err)
		}

		rawInput, err := u.input.Poll(ctx)
		if err != nil {
			return fmt.Errorf("poll input: %w", err)
		}

		commands := u.converter.ToCommands(rawInput)
		frame, err := u.simulation.Step(commands)
		if err != nil {
			return fmt.Errorf("simulate frame: %w", err)
		}

		if err := u.renderer.Render(ctx, frame); err != nil {
			return fmt.Errorf("render frame: %w", err)
		}

		ticksRun++
		if !frame.Running {
			break
		}
	}

	return nil
}
