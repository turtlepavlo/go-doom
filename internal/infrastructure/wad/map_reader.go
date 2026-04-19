package wad

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

var (
	ErrMapNotFound         = errors.New("map marker not found")
	ErrMissingThingsLump   = errors.New("THINGS lump is missing")
	ErrMissingLinedefsLump = errors.New("LINEDEFS lump is missing")
	ErrMissingSidedefsLump = errors.New("SIDEDEFS lump is missing")
	ErrMissingVertexesLump = errors.New("VERTEXES lump is missing")
	ErrMissingSectorsLump  = errors.New("SECTORS lump is missing")
)

type MapReader struct{}

func NewMapReader() *MapReader {
	return &MapReader{}
}

func (r *MapReader) ReadMap(ctx context.Context, wadPath string, mapName string) (dto.RawMap, error) {
	select {
	case <-ctx.Done():
		return dto.RawMap{}, ctx.Err()
	default:
	}

	fileData, err := os.ReadFile(wadPath)
	if err != nil {
		return dto.RawMap{}, fmt.Errorf("read file %q: %w", wadPath, err)
	}

	archive, err := parseBinary(fileData)
	if err != nil {
		return dto.RawMap{}, fmt.Errorf("parse archive %q: %w", wadPath, err)
	}

	startIdx := findMapStartIndex(archive.Lumps, mapName)
	if startIdx < 0 {
		return dto.RawMap{}, fmt.Errorf("%w: %s", ErrMapNotFound, mapName)
	}

	lumpData := make(map[string][]byte, 8)
	for i := startIdx + 1; i < len(archive.Lumps); i++ {
		name := strings.ToUpper(strings.TrimSpace(archive.Lumps[i].Name))
		if domain.IsMapMarker(name) {
			break
		}

		if _, exists := lumpData[name]; exists {
			continue
		}

		start := int(archive.Lumps[i].Offset)
		size := int(archive.Lumps[i].Size)
		data := make([]byte, size)
		copy(data, fileData[start:start+size])
		lumpData[name] = data
	}

	rawMap, err := buildRawMap(strings.ToUpper(strings.TrimSpace(mapName)), lumpData)
	if err != nil {
		return dto.RawMap{}, err
	}

	return rawMap, nil
}

func findMapStartIndex(lumps []dto.RawLump, mapName string) int {
	target := strings.ToUpper(strings.TrimSpace(mapName))
	for i := range lumps {
		if strings.ToUpper(strings.TrimSpace(lumps[i].Name)) == target {
			return i
		}
	}
	return -1
}

func buildRawMap(name string, lumpData map[string][]byte) (dto.RawMap, error) {
	things, ok := lumpData["THINGS"]
	if !ok {
		return dto.RawMap{}, ErrMissingThingsLump
	}
	linedefs, ok := lumpData["LINEDEFS"]
	if !ok {
		return dto.RawMap{}, ErrMissingLinedefsLump
	}
	sidedefs, ok := lumpData["SIDEDEFS"]
	if !ok {
		return dto.RawMap{}, ErrMissingSidedefsLump
	}
	vertexes, ok := lumpData["VERTEXES"]
	if !ok {
		return dto.RawMap{}, ErrMissingVertexesLump
	}
	sectors, ok := lumpData["SECTORS"]
	if !ok {
		return dto.RawMap{}, ErrMissingSectorsLump
	}

	return dto.RawMap{
		Name:     name,
		Things:   things,
		Linedefs: linedefs,
		Sidedefs: sidedefs,
		Vertexes: vertexes,
		Sectors:  sectors,
	}, nil
}
