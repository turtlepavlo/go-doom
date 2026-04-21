package runtime

import (
	"context"
	"math"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type Engine struct {
	state domain.GameState
}

func NewEngine() *Engine {
	return NewEngineAt(0, 0)
}

func NewEngineAt(playerX int64, playerY int64) *Engine {
	return NewEnginePose(playerX, playerY, 0)
}

func NewEnginePose(playerX int64, playerY int64, angle float64) *Engine {
	return &Engine{
		state: domain.GameState{
			PlayerX: playerX,
			PlayerY: playerY,
			Angle:   normalizeEngineAngle(angle),
			Running: true,
		},
	}
}

func (e *Engine) Step(_ context.Context, commands []domain.Command) domain.Frame {
	if !e.state.Running {
		return e.snapshot()
	}

	const (
		moveStep = 14.0
		turnStep = 0.11
	)

	var (
		moveForward  bool
		moveBackward bool
		strafeLeft   bool
		strafeRight  bool
		turnLeft     bool
		turnRight    bool
	)

	for _, command := range commands {
		switch command {
		case domain.CommandMoveForward:
			moveForward = true
		case domain.CommandMoveBackward:
			moveBackward = true
		case domain.CommandStrafeLeft:
			strafeLeft = true
		case domain.CommandStrafeRight:
			strafeRight = true
		case domain.CommandTurnLeft:
			turnLeft = true
		case domain.CommandTurnRight:
			turnRight = true
		case domain.CommandQuit:
			e.state.Running = false
		}
	}

	if turnLeft {
		e.state.Angle -= turnStep
	}
	if turnRight {
		e.state.Angle += turnStep
	}
	e.state.Angle = normalizeEngineAngle(e.state.Angle)

	var dx float64
	var dy float64

	cosA := math.Cos(e.state.Angle)
	sinA := math.Sin(e.state.Angle)

	if moveForward {
		dx += cosA * moveStep
		dy += sinA * moveStep
	}
	if moveBackward {
		dx -= cosA * moveStep
		dy -= sinA * moveStep
	}
	if strafeLeft {
		dx += sinA * moveStep
		dy -= cosA * moveStep
	}
	if strafeRight {
		dx -= sinA * moveStep
		dy += cosA * moveStep
	}

	e.state.PlayerX = int64(math.Round(float64(e.state.PlayerX) + dx))
	e.state.PlayerY = int64(math.Round(float64(e.state.PlayerY) + dy))

	e.state.Tick++
	return e.snapshot()
}

func (e *Engine) State() domain.GameState {
	return e.state
}

func (e *Engine) Frame() domain.Frame {
	return e.snapshot()
}

func (e *Engine) SetPlayerPosition(playerX int64, playerY int64) {
	e.state.PlayerX = playerX
	e.state.PlayerY = playerY
}

func (e *Engine) Stop() {
	e.state.Running = false
}

func (e *Engine) snapshot() domain.Frame {
	return domain.Frame{
		Tick:    e.state.Tick,
		PlayerX: e.state.PlayerX,
		PlayerY: e.state.PlayerY,
		Angle:   e.state.Angle,
		Running: e.state.Running,
	}
}

func normalizeEngineAngle(angle float64) float64 {
	const twoPi = 2 * math.Pi
	for angle > math.Pi {
		angle -= twoPi
	}
	for angle < -math.Pi {
		angle += twoPi
	}
	return angle
}
