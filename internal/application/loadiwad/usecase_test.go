package loadiwad

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type readerStub struct {
	raw dto.RawArchive
	err error
}

func (s *readerStub) ReadArchive(ctx context.Context, path string) (dto.RawArchive, error) {
	if s.err != nil {
		return dto.RawArchive{}, s.err
	}
	return s.raw, nil
}

type converterStub struct {
	archive domain.Archive
	err     error
}

func (s *converterStub) ToDomain(raw dto.RawArchive) (domain.Archive, error) {
	if s.err != nil {
		return domain.Archive{}, s.err
	}
	return s.archive, nil
}

func TestUseCase_Execute(t *testing.T) {
	expected := domain.Archive{
		Header: domain.Header{Kind: domain.ArchiveKindIWAD, LumpCount: 1, DirectoryOffset: 12},
	}

	useCase, err := New(
		&readerStub{
			raw: dto.RawArchive{
				Type:            "IWAD",
				LumpCount:       1,
				DirectoryOffset: 12,
			},
		},
		&converterStub{archive: expected},
	)
	if err != nil {
		t.Fatalf("unexpected bootstrap error: %v", err)
	}

	got, err := useCase.Execute(context.Background(), "doom1.wad")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Header.Kind != domain.ArchiveKindIWAD {
		t.Fatalf("expected IWAD, got %q", got.Header.Kind)
	}
}

func TestUseCase_Execute_EmptyPath(t *testing.T) {
	useCase, err := New(&readerStub{}, &converterStub{})
	if err != nil {
		t.Fatalf("unexpected bootstrap error: %v", err)
	}

	_, execErr := useCase.Execute(context.Background(), " ")
	if !errors.Is(execErr, ErrEmptyPath) {
		t.Fatalf("expected ErrEmptyPath, got %v", execErr)
	}
}
