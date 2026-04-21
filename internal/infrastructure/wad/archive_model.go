package wad

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	archiveKindIWAD = "IWAD"
	archiveKindPWAD = "PWAD"
)

var (
	ErrInvalidArchiveKind = errors.New("invalid archive kind")
	ErrNegativeOffset     = errors.New("negative offset")
	ErrNegativeSize       = errors.New("negative size")
	ErrNegativeCount      = errors.New("negative lump count")
	ErrEmptyLumpName      = errors.New("empty lump name")
	ErrLumpCountMismatch  = errors.New("lump count mismatch")
	ErrEmptyMapName       = errors.New("empty map name")

	episodeMapPattern = regexp.MustCompile(`^E[1-9]M[1-9]$`)
	numericMapPattern = regexp.MustCompile(`^MAP[0-9]{2}$`)
)

func buildHeader(kind string, lumpCount int, directoryOffset int) (domain.Header, error) {
	normalizedKind := strings.ToUpper(strings.TrimSpace(kind))
	if normalizedKind != archiveKindIWAD && normalizedKind != archiveKindPWAD {
		return domain.Header{}, fmt.Errorf("%w: %q", ErrInvalidArchiveKind, kind)
	}
	if lumpCount < 0 {
		return domain.Header{}, ErrNegativeCount
	}
	if directoryOffset < 0 {
		return domain.Header{}, ErrNegativeOffset
	}
	return domain.Header{
		Kind:            normalizedKind,
		LumpCount:       int64(lumpCount),
		DirectoryOffset: int64(directoryOffset),
	}, nil
}

func buildLump(name string, offset int, size int) (domain.Lump, error) {
	normalizedName := strings.ToUpper(strings.TrimSpace(name))
	if normalizedName == "" {
		return domain.Lump{}, ErrEmptyLumpName
	}
	if offset < 0 {
		return domain.Lump{}, ErrNegativeOffset
	}
	if size < 0 {
		return domain.Lump{}, ErrNegativeSize
	}
	return domain.Lump{
		Name:   normalizedName,
		Offset: int64(offset),
		Size:   int64(size),
	}, nil
}

func buildArchive(header domain.Header, lumps []domain.Lump, maps []domain.Map) (domain.Archive, error) {
	if header.LumpCount != int64(len(lumps)) {
		return domain.Archive{}, fmt.Errorf("%w: header=%d parsed=%d", ErrLumpCountMismatch, header.LumpCount, len(lumps))
	}
	return domain.Archive{
		Header: header,
		Lumps:  append([]domain.Lump(nil), lumps...),
		Maps:   append([]domain.Map(nil), maps...),
	}, nil
}

func buildMapIndex(lumps []domain.Lump) []domain.Map {
	maps := make([]domain.Map, 0)
	currentMapIndex := -1

	for _, lump := range lumps {
		if isMapMarker(lump.Name) {
			maps = append(maps, domain.Map{
				Name:  lump.Name,
				Lumps: make([]domain.Lump, 0, 10),
			})
			currentMapIndex = len(maps) - 1
			continue
		}

		if currentMapIndex >= 0 {
			maps[currentMapIndex].Lumps = append(maps[currentMapIndex].Lumps, lump)
		}
	}

	return maps
}

func isMapMarker(name string) bool {
	return episodeMapPattern.MatchString(name) || numericMapPattern.MatchString(name)
}
