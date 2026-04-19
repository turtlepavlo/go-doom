package domain

import (
	"errors"
	"testing"
)

func TestNewLevel(t *testing.T) {
	level, err := NewLevel("E1M1", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if level.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", level.Name)
	}
}

func TestNewLevelEmptyName(t *testing.T) {
	_, err := NewLevel("", nil, nil, nil, nil, nil)
	if !errors.Is(err, ErrEmptyMapName) {
		t.Fatalf("expected ErrEmptyMapName, got %v", err)
	}
}
