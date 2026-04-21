package wad

import (
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

func buildLevel(
	name string,
	things []domain.Thing,
	linedefs []domain.Linedef,
	sidedefs []domain.Sidedef,
	vertexes []domain.Vertex,
	sectors []domain.Sector,
) (domain.Level, error) {
	if strings.TrimSpace(name) == "" {
		return domain.Level{}, ErrEmptyMapName
	}

	return domain.Level{
		Name:     name,
		Things:   append([]domain.Thing(nil), things...),
		Linedefs: append([]domain.Linedef(nil), linedefs...),
		Sidedefs: append([]domain.Sidedef(nil), sidedefs...),
		Vertexes: append([]domain.Vertex(nil), vertexes...),
		Sectors:  append([]domain.Sector(nil), sectors...),
	}, nil
}
