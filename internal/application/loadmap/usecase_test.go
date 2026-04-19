package loadmap

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type mapReaderStub struct {
	raw dto.RawMap
	err error
}

func (s *mapReaderStub) ReadMap(ctx context.Context, wadPath string, mapName string) (dto.RawMap, error) {
	if s.err != nil {
		return dto.RawMap{}, s.err
	}
	return s.raw, nil
}

type mapConverterStub struct {
	level domain.Level
	err   error
}

func (s *mapConverterStub) ToLevel(raw dto.RawMap) (domain.Level, error) {
	if s.err != nil {
		return domain.Level{}, s.err
	}
	return s.level, nil
}

func TestUseCaseExecute(t *testing.T) {
	expected, err := domain.NewLevel("E1M1", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("build level: %v", err)
	}

	useCase, err := New(
		&mapReaderStub{raw: dto.RawMap{Name: "E1M1"}},
		&mapConverterStub{level: expected},
	)
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	level, err := useCase.Execute(context.Background(), "doom1.wad", "E1M1")
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if level.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", level.Name)
	}
}

func TestUseCaseExecuteEmptyName(t *testing.T) {
	useCase, err := New(&mapReaderStub{}, &mapConverterStub{})
	if err != nil {
		t.Fatalf("new use case: %v", err)
	}

	_, execErr := useCase.Execute(context.Background(), "doom1.wad", "")
	if !errors.Is(execErr, ErrMapNameEmpty) {
		t.Fatalf("expected ErrMapNameEmpty, got %v", execErr)
	}
}
