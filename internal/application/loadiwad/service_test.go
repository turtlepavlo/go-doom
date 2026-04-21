package loadiwad

import (
	"context"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type readerStub struct {
	raw RawArchive
	err error
}

func (s *readerStub) ReadArchive(ctx context.Context, path string) (RawArchive, error) {
	if s.err != nil {
		return RawArchive{}, s.err
	}
	return s.raw, nil
}

type converterStub struct {
	archive domain.Archive
	err     error
}

func (s *converterStub) Archive(raw RawArchive) (domain.Archive, error) {
	if s.err != nil {
		return domain.Archive{}, s.err
	}
	return s.archive, nil
}

func TestService_Execute(t *testing.T) {
	expected := domain.Archive{
		Header: domain.Header{Kind: "IWAD", LumpCount: 1, DirectoryOffset: 12},
	}

	service, err := New(
		&readerStub{
			raw: RawArchive{
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

	got, err := service.Execute(context.Background(), "doom1.wad")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Header.Kind != "IWAD" {
		t.Fatalf("expected IWAD, got %q", got.Header.Kind)
	}
}

func TestService_Execute_EmptyPath(t *testing.T) {
	service, err := New(&readerStub{}, &converterStub{})
	if err != nil {
		t.Fatalf("unexpected bootstrap error: %v", err)
	}

	_, execErr := service.Execute(context.Background(), " ")
	if !errors.Is(execErr, ErrEmptyPath) {
		t.Fatalf("expected ErrEmptyPath, got %v", execErr)
	}
}
