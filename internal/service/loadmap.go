package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func (s *GameService) LoadMap(ctx context.Context, wadPath string, mapName string) (domain.Level, error) {
	if strings.TrimSpace(wadPath) == "" {
		return domain.Level{}, ErrEmptyWADPath
	}
	if strings.TrimSpace(mapName) == "" {
		return domain.Level{}, ErrEmptyMapName
	}

	rawMap, err := s.mapReader.ReadMap(ctx, wadPath, mapName)
	if err != nil {
		return domain.Level{}, fmt.Errorf("read map: %w", err)
	}

	level, err := s.mapConverter.Level(rawMap)
	if err != nil {
		return domain.Level{}, fmt.Errorf("convert map: %w", err)
	}

	return level, nil
}
