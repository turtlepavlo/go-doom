package loadmap

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilMapReader    = errors.New("nil map reader")
	ErrNilMapConverter = errors.New("nil map converter")
	ErrMapNameEmpty    = errors.New("map name is empty")
	ErrWADPathEmpty    = errors.New("wad path is empty")
)

type Reader interface {
	ReadMap(ctx context.Context, wadPath string, mapName string) (dto.RawMap, error)
}

type Converter interface {
	ToLevel(raw dto.RawMap) (domain.Level, error)
}

type UseCase struct {
	reader    Reader
	converter Converter
}

func New(reader Reader, converter Converter) (*UseCase, error) {
	switch {
	case reader == nil:
		return nil, ErrNilMapReader
	case converter == nil:
		return nil, ErrNilMapConverter
	}

	return &UseCase{
		reader:    reader,
		converter: converter,
	}, nil
}

func (u *UseCase) Execute(ctx context.Context, wadPath string, mapName string) (domain.Level, error) {
	if strings.TrimSpace(wadPath) == "" {
		return domain.Level{}, ErrWADPathEmpty
	}
	if strings.TrimSpace(mapName) == "" {
		return domain.Level{}, ErrMapNameEmpty
	}

	rawMap, err := u.reader.ReadMap(ctx, wadPath, mapName)
	if err != nil {
		return domain.Level{}, fmt.Errorf("read map: %w", err)
	}

	level, err := u.converter.ToLevel(rawMap)
	if err != nil {
		return domain.Level{}, fmt.Errorf("convert map: %w", err)
	}

	return level, nil
}
