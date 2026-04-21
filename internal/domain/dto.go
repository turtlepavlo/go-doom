package domain

// DTO-like command contracts used across runtime and input mapping.

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
