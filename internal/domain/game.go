package domain

import "math"

type Command string

const (
	CommandMoveForward  Command = "MOVE_FORWARD"
	CommandMoveBackward Command = "MOVE_BACKWARD"
	CommandStrafeLeft   Command = "STRAFE_LEFT"
	CommandStrafeRight  Command = "STRAFE_RIGHT"
	CommandTurnLeft     Command = "TURN_LEFT"
	CommandTurnRight    Command = "TURN_RIGHT"
	CommandFire         Command = "FIRE"
	CommandQuit         Command = "QUIT"
)

type GameState struct {
	Tick    uint64
	PlayerX int
	PlayerY int
	Angle   float64
	Running bool
}

type Frame struct {
	Tick             uint64
	PlayerX          int
	PlayerY          int
	Angle            float64
	Running          bool
	Health           int
	Ammo             int
	EnemyCount       int
	EnemyAlive       int
	Kills            int
	ShotsFired       int
	ShotHits         int
	WeaponCooldown   int
	WeaponFlashTicks int
	DamageFlashTicks int
	Enemies          []EnemySnapshot
}

type EnemySnapshot struct {
	X         int
	Y         int
	TypeID    uint16
	Kind      string
	Health    int
	HurtTicks int
	Alive     bool
}

type Engine struct {
	state GameState
}

func NewEngine() *Engine {
	return NewEngineAt(0, 0)
}

func NewEngineAt(playerX int, playerY int) *Engine {
	return NewEnginePose(playerX, playerY, 0)
}

func NewEnginePose(playerX int, playerY int, angle float64) *Engine {
	return &Engine{
		state: GameState{
			PlayerX: playerX,
			PlayerY: playerY,
			Angle:   normalizeAngle(angle),
			Running: true,
		},
	}
}

func (e *Engine) Step(commands []Command) Frame {
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
		case CommandMoveForward:
			moveForward = true
		case CommandMoveBackward:
			moveBackward = true
		case CommandStrafeLeft:
			strafeLeft = true
		case CommandStrafeRight:
			strafeRight = true
		case CommandTurnLeft:
			turnLeft = true
		case CommandTurnRight:
			turnRight = true
		case CommandQuit:
			e.state.Running = false
		}
	}

	if turnLeft {
		e.state.Angle -= turnStep
	}
	if turnRight {
		e.state.Angle += turnStep
	}
	e.state.Angle = normalizeAngle(e.state.Angle)

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

	e.state.PlayerX = int(math.Round(float64(e.state.PlayerX) + dx))
	e.state.PlayerY = int(math.Round(float64(e.state.PlayerY) + dy))

	e.state.Tick++
	return e.snapshot()
}

func (e *Engine) State() GameState {
	return e.state
}

func (e *Engine) Frame() Frame {
	return e.snapshot()
}

func (e *Engine) SetPlayerPosition(playerX int, playerY int) {
	e.state.PlayerX = playerX
	e.state.PlayerY = playerY
}

func (e *Engine) SetAngle(angle float64) {
	e.state.Angle = normalizeAngle(angle)
}

func (e *Engine) Stop() {
	e.state.Running = false
}

func (e *Engine) snapshot() Frame {
	return Frame{
		Tick:    e.state.Tick,
		PlayerX: e.state.PlayerX,
		PlayerY: e.state.PlayerY,
		Angle:   e.state.Angle,
		Running: e.state.Running,
	}
}

func normalizeAngle(angle float64) float64 {
	const twoPi = 2 * math.Pi
	for angle > math.Pi {
		angle -= twoPi
	}
	for angle < -math.Pi {
		angle += twoPi
	}
	return angle
}
