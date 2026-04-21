package loadiwad

import (
	"context"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func (svc *Service) Execute(ctx context.Context, path string) (domain.Archive, error) {
	if strings.TrimSpace(path) == "" {
		return domain.Archive{}, ErrEmptyPath
	}

	raw, err := svc.reader.ReadArchive(ctx, path)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("read raw archive: %w", err)
	}

	archive, err := svc.converter.Archive(raw)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("convert raw archive to domain: %w", err)
	}

	return archive, nil
}
