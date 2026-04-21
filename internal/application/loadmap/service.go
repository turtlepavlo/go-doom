package loadmap

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilMapReader    = errors.New("nil map reader")
	ErrNilMapConverter = errors.New("nil map converter")
	ErrMapNameEmpty    = errors.New("map name is empty")
	ErrWADPathEmpty    = errors.New("wad path is empty")
)

type Reader interface {
	ReadMap(ctx context.Context, wadPath string, mapName string) (RawMap, error)
}

type Converter interface {
	Level(raw RawMap) (domain.Level, error)
}

type Service struct {
	reader    Reader
	converter Converter
}

func New(reader Reader, converter Converter) (*Service, error) {
	switch {
	case reader == nil:
		return nil, ErrNilMapReader
	case converter == nil:
		return nil, ErrNilMapConverter
	}

	return &Service{
		reader:    reader,
		converter: converter,
	}, nil
}
