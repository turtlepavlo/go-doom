package loadmap

import (
	"context"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func (svc *Service) Execute(ctx context.Context, wadPath string, mapName string) (domain.Level, error) {
	if strings.TrimSpace(wadPath) == "" {
		return domain.Level{}, ErrWADPathEmpty
	}
	if strings.TrimSpace(mapName) == "" {
		return domain.Level{}, ErrMapNameEmpty
	}

	rawMap, err := svc.reader.ReadMap(ctx, wadPath, mapName)
	if err != nil {
		return domain.Level{}, fmt.Errorf("read map: %w", err)
	}

	level, err := svc.converter.Level(rawMap)
	if err != nil {
		return domain.Level{}, fmt.Errorf("convert map: %w", err)
	}

	return level, nil
}
