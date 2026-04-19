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
	return &Engine{
		state: GameState{
			Running: true,
		},
	}
}

func (e *Engine) Step(commands []Command) Frame {
	if !e.state.Running {
		return e.snapshot()
	}

	for _, command := range commands {
		switch command {
		case CommandMoveForward:
			e.state.PlayerY--
		case CommandMoveBackward:
			e.state.PlayerY++
		case CommandStrafeLeft:
			e.state.PlayerX--
		case CommandStrafeRight:
			e.state.PlayerX++
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

func (e *Engine) snapshot() Frame {
	return Frame{
		Tick:    e.state.Tick,
		PlayerX: e.state.PlayerX,
		PlayerY: e.state.PlayerY,
		Running: e.state.Running,
	}
}
