package domain

import "testing"

func TestEngineStep_AppliesCommands(t *testing.T) {
	engine := NewEngineAt(0, 0)
	frame := engine.Step([]Command{
		CommandMoveForward,
	})

	if frame.Tick != 1 {
		t.Fatalf("expected tick=1, got %d", frame.Tick)
	}
	if frame.PlayerX != 14 || frame.PlayerY != 0 {
		t.Fatalf("expected player position (14,0), got (%d,%d)", frame.PlayerX, frame.PlayerY)
	}
	if !frame.Running {
		t.Fatal("expected running=true")
	}
}

func TestEngineStep_Quit(t *testing.T) {
	engine := NewEngine()
	frame := engine.Step([]Command{CommandQuit})

	if frame.Running {
		t.Fatal("expected running=false after quit command")
	}
}

func TestNewEngineAt(t *testing.T) {
	engine := NewEngineAt(128, -64)
	state := engine.State()
	if state.PlayerX != 128 || state.PlayerY != -64 {
		t.Fatalf("expected start position (128,-64), got (%d,%d)", state.PlayerX, state.PlayerY)
	}
}

func TestFrame(t *testing.T) {
	engine := NewEngineAt(32, 16)
	frame := engine.Frame()
	if frame.PlayerX != 32 || frame.PlayerY != 16 || !frame.Running {
		t.Fatalf("unexpected frame snapshot: %+v", frame)
	}
}

func TestEngineStep_TurnChangesAngle(t *testing.T) {
	engine := NewEngine()
	before := engine.Frame().Angle
	after := engine.Step([]Command{CommandTurnRight}).Angle
	if after <= before {
		t.Fatalf("expected angle to increase, before=%f after=%f", before, after)
	}
}
