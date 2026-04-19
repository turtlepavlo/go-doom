package domain

import "testing"

func TestEngineStep_AppliesCommands(t *testing.T) {
	engine := NewEngine()
	frame := engine.Step([]Command{
		CommandMoveForward,
		CommandStrafeRight,
	})

	if frame.Tick != 1 {
		t.Fatalf("expected tick=1, got %d", frame.Tick)
	}
	if frame.PlayerX != 1 || frame.PlayerY != -1 {
		t.Fatalf("expected player position (1,-1), got (%d,%d)", frame.PlayerX, frame.PlayerY)
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
