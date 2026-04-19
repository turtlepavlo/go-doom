package domain

type Command string

const (
	CommandMoveForward  Command = "MOVE_FORWARD"
	CommandMoveBackward Command = "MOVE_BACKWARD"
	CommandStrafeLeft   Command = "STRAFE_LEFT"
	CommandStrafeRight  Command = "STRAFE_RIGHT"
	CommandQuit         Command = "QUIT"
)

type GameState struct {
	Tick    uint64
	PlayerX int
	PlayerY int
	Running bool
}

type Frame struct {
	Tick    uint64
	PlayerX int
	PlayerY int
	Running bool
}

type Engine struct {
	state GameState
}

func NewEngine() *Engine {
	return NewEngineAt(0, 0)
}

func NewEngineAt(playerX int, playerY int) *Engine {
	return &Engine{
		state: GameState{
			PlayerX: playerX,
			PlayerY: playerY,
			Running: true,
		},
	}
}

func (e *Engine) Step(commands []Command) Frame {
	if !e.state.Running {
		return e.snapshot()
	}

	const moveStep = 16

	for _, command := range commands {
		switch command {
		case CommandMoveForward:
			e.state.PlayerY -= moveStep
		case CommandMoveBackward:
			e.state.PlayerY += moveStep
		case CommandStrafeLeft:
			e.state.PlayerX -= moveStep
		case CommandStrafeRight:
			e.state.PlayerX += moveStep
		case CommandQuit:
			e.state.Running = false
		}
	}

	e.state.Tick++
	return e.snapshot()
}

func (e *Engine) State() GameState {
	return e.state
}

func (e *Engine) Frame() Frame {
	return e.snapshot()
}

func (e *Engine) snapshot() Frame {
	return Frame{
		Tick:    e.state.Tick,
		PlayerX: e.state.PlayerX,
		PlayerY: e.state.PlayerY,
		Running: e.state.Running,
	}
}
