package converters

import (
	"fmt"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type WADConverter struct{}

func NewWADConverter() *WADConverter {
	return &WADConverter{}
}

func (c *WADConverter) ToDomain(raw dto.RawArchive) (domain.Archive, error) {
	header, err := domain.NewHeader(raw.Type, int(raw.LumpCount), int(raw.DirectoryOffset))
	if err != nil {
		return domain.Archive{}, fmt.Errorf("build header: %w", err)
	}

	lumps := make([]domain.Lump, 0, len(raw.Lumps))
	for i, rawLump := range raw.Lumps {
		lump, convErr := domain.NewLump(rawLump.Name, int(rawLump.Offset), int(rawLump.Size))
		if convErr != nil {
			return domain.Archive{}, fmt.Errorf("build lump at index %d: %w", i, convErr)
		}
		lumps = append(lumps, lump)
	}

	maps := domain.BuildMapIndex(lumps)

	archive, err := domain.NewArchive(header, lumps, maps)
	if err != nil {
		return domain.Archive{}, fmt.Errorf("build archive: %w", err)
	}

	return archive, nil
}
