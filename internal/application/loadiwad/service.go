package loadiwad

import (
	"context"
	"errors"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilReader    = errors.New("nil reader")
	ErrNilConverter = errors.New("nil converter")
	ErrEmptyPath    = errors.New("empty path")
)

type Reader interface {
	ReadArchive(ctx context.Context, path string) (RawArchive, error)
}

type Converter interface {
	Archive(raw RawArchive) (domain.Archive, error)
}

type Service struct {
	reader    Reader
	converter Converter
}

func New(reader Reader, converter Converter) (*Service, error) {
	if reader == nil {
		return nil, ErrNilReader
	}
	if converter == nil {
		return nil, ErrNilConverter
	}
	return &Service{
		reader:    reader,
		converter: converter,
	}, nil
}
