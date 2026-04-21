package loadmap

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type mapReaderStub struct {
	raw RawMap
	err error
}

func (s *mapReaderStub) ReadMap(ctx context.Context, wadPath string, mapName string) (RawMap, error) {
	if s.err != nil {
		return RawMap{}, s.err
	}
	return s.raw, nil
}

type mapConverterStub struct {
	level domain.Level
	err   error
}

func (s *mapConverterStub) Level(raw RawMap) (domain.Level, error) {
	if s.err != nil {
		return domain.Level{}, s.err
	}
	return s.level, nil
}

func TestServiceExecute(t *testing.T) {
	expected := domain.Level{Name: "E1M1"}

	service, err := New(
		&mapReaderStub{raw: RawMap{Name: "E1M1"}},
		&mapConverterStub{level: expected},
	)
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	level, err := service.Execute(context.Background(), "doom1.wad", "E1M1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if level.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", level.Name)
	}
}

func TestServiceExecuteEmptyName(t *testing.T) {
	service, err := New(&mapReaderStub{}, &mapConverterStub{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, execErr := service.Execute(context.Background(), "doom1.wad", "")
	if !errors.Is(execErr, ErrMapNameEmpty) {
		t.Fatalf("expected ErrMapNameEmpty, got %v", execErr)
	}
}
