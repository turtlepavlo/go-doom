package loadiwad

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrNilReader    = errors.New("nil reader")
	ErrNilConverter = errors.New("nil converter")
	ErrEmptyPath    = errors.New("empty path")
)

type Reader interface {
	ReadArchive(ctx context.Context, path string) (dto.RawArchive, error)
}

type Converter interface {
	ToDomain(raw dto.RawArchive) (domain.Archive, error)
}

type UseCase struct {
	reader    Reader
	converter Converter
}

func New(reader Reader, converter Converter) (*UseCase, error) {
	if reader == nil {
		return nil, ErrNilReader
	}
	if converter == nil {
		return nil, ErrNilConverter
	}
	return &UseCase{
		reader:    reader,
		converter: converter,
	}, nil
}

func (u *UseCase) Execute(ctx context.Context, path string) (domain.Archive, error) {
	if strings.TrimSpace(path) == "" {
		return domain.Archive{}, ErrEmptyPath
	}

	raw, err := u.reader.ReadArchive(ctx, path)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("read raw archive: %w", err)
	}

	archive, err := u.converter.ToDomain(raw)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("convert raw archive to domain: %w", err)
	}

	return archive, nil
}
