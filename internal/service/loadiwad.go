package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func (s *GameService) LoadIWAD(ctx context.Context, path string) (domain.Archive, error) {
	if strings.TrimSpace(path) == "" {
		return domain.Archive{}, ErrEmptyIWADPath
	}

	raw, err := s.archiveReader.ReadArchive(ctx, path)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("read raw archive: %w", err)
	}

	archive, err := s.archiveConverter.Archive(raw)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("convert archive to domain: %w", err)
	}

	return archive, nil
}
