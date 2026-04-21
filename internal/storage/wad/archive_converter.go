package wad

import (
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type ToArchiveConvert struct{}

func NewToArchiveConvert() *ToArchiveConvert { return &ToArchiveConvert{} }

func (c *ToArchiveConvert) Archive(raw loadiwad.RawArchive) (domain.Archive, error) {
	header, err := buildHeader(raw.Type, int64(raw.LumpCount), int64(raw.DirectoryOffset))
	if err != nil {
		return domain.Archive{}, fmt.Errorf("build header: %w", err)
	}

	lumps := make([]domain.Lump, 0, len(raw.Lumps))
	for i, rawLump := range raw.Lumps {
		lump, convErr := buildLump(rawLump.Name, int64(rawLump.Offset), int64(rawLump.Size))
		if convErr != nil {
			return domain.Archive{}, fmt.Errorf("build lump at index %d: %w", i, convErr)
		}
		lumps = append(lumps, lump)
	}

	maps := buildMapIndex(lumps)

	archive, err := buildArchive(header, lumps, maps)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("build archive: %w", err)
	}

	return archive, nil
}
