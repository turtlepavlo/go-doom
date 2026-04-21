package wad

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	thingRecordSize   = 10
	linedefRecordSize = 14
	sidedefRecordSize = 30
	vertexRecordSize  = 4
	sectorRecordSize  = 26
)

var (
	ErrInvalidThingsLumpSize   = errors.New("invalid THINGS lump size")
	ErrInvalidLinedefsLumpSize = errors.New("invalid LINEDEFS lump size")
	ErrInvalidSidedefsLumpSize = errors.New("invalid SIDEDEFS lump size")
	ErrInvalidVertexesLumpSize = errors.New("invalid VERTEXES lump size")
	ErrInvalidSectorsLumpSize  = errors.New("invalid SECTORS lump size")
)

type ToLevelConvert struct{}

func NewToLevelConvert() *ToLevelConvert { return &ToLevelConvert{} }

func (c *ToLevelConvert) Level(raw loadmap.RawMap) (domain.Level, error) {
	things, err := parseThings(raw.Things)
	if err != nil {
		return domain.Level{}, err
	}

	linedefs, err := parseLinedefs(raw.Linedefs)
	if err != nil {
		return domain.Level{}, err
	}

	sidedefs, err := parseSidedefs(raw.Sidedefs)
	if err != nil {
		return domain.Level{}, err
	}

	vertexes, err := parseVertexes(raw.Vertexes)
	if err != nil {
		return domain.Level{}, err
	}

	sectors, err := parseSectors(raw.Sectors)
	if err != nil {
		return domain.Level{}, err
	}

	level, err := buildLevel(raw.Name, things, linedefs, sidedefs, vertexes, sectors)
	if err != nil {
		return domain.Level{}, fmt.Errorf("build level: %w", err)
	}

	return level, nil
}

func parseThings(data []byte) ([]domain.Thing, error) {
	if len(data)%thingRecordSize != 0 {
		return nil, fmt.Errorf("%w: got %d bytes, expected multiple of %d", ErrInvalidThingsLumpSize, len(data), thingRecordSize)
	}

	out := make([]domain.Thing, 0, len(data)/thingRecordSize)
	for i := 0; i < len(data); i += thingRecordSize {
		out = append(out, domain.Thing{
			X:     int16(binary.LittleEndian.Uint16(data[i : i+2])),
			Y:     int16(binary.LittleEndian.Uint16(data[i+2 : i+4])),
			Angle: binary.LittleEndian.Uint16(data[i+4 : i+6]),
			Type:  binary.LittleEndian.Uint16(data[i+6 : i+8]),
			Flags: binary.LittleEndian.Uint16(data[i+8 : i+10]),
		})
	}

	return out, nil
}

func parseLinedefs(data []byte) ([]domain.Linedef, error) {
	if len(data)%linedefRecordSize != 0 {
		return nil, fmt.Errorf("%w: got %d bytes, expected multiple of %d", ErrInvalidLinedefsLumpSize, len(data), linedefRecordSize)
	}

	out := make([]domain.Linedef, 0, len(data)/linedefRecordSize)
	for i := 0; i < len(data); i += linedefRecordSize {
		out = append(out, domain.Linedef{
			StartVertex: binary.LittleEndian.Uint16(data[i : i+2]),
			EndVertex:   binary.LittleEndian.Uint16(data[i+2 : i+4]),
			Flags:       binary.LittleEndian.Uint16(data[i+4 : i+6]),
			SpecialType: binary.LittleEndian.Uint16(data[i+6 : i+8]),
			SectorTag:   binary.LittleEndian.Uint16(data[i+8 : i+10]),
			RightSide:   binary.LittleEndian.Uint16(data[i+10 : i+12]),
			LeftSide:    binary.LittleEndian.Uint16(data[i+12 : i+14]),
		})
	}

	return out, nil
}

func parseSidedefs(data []byte) ([]domain.Sidedef, error) {
	if len(data)%sidedefRecordSize != 0 {
		return nil, fmt.Errorf("%w: got %d bytes, expected multiple of %d", ErrInvalidSidedefsLumpSize, len(data), sidedefRecordSize)
	}

	out := make([]domain.Sidedef, 0, len(data)/sidedefRecordSize)
	for i := 0; i < len(data); i += sidedefRecordSize {
		out = append(out, domain.Sidedef{
			XOffset:       int16(binary.LittleEndian.Uint16(data[i : i+2])),
			YOffset:       int16(binary.LittleEndian.Uint16(data[i+2 : i+4])),
			UpperTexture:  parseTextureName(data[i+4 : i+12]),
			LowerTexture:  parseTextureName(data[i+12 : i+20]),
			MiddleTexture: parseTextureName(data[i+20 : i+28]),
			Sector:        binary.LittleEndian.Uint16(data[i+28 : i+30]),
		})
	}

	return out, nil
}

func parseVertexes(data []byte) ([]domain.Vertex, error) {
	if len(data)%vertexRecordSize != 0 {
		return nil, fmt.Errorf("%w: got %d bytes, expected multiple of %d", ErrInvalidVertexesLumpSize, len(data), vertexRecordSize)
	}

	out := make([]domain.Vertex, 0, len(data)/vertexRecordSize)
	for i := 0; i < len(data); i += vertexRecordSize {
		out = append(out, domain.Vertex{
			X: int16(binary.LittleEndian.Uint16(data[i : i+2])),
			Y: int16(binary.LittleEndian.Uint16(data[i+2 : i+4])),
		})
	}

	return out, nil
}

func parseSectors(data []byte) ([]domain.Sector, error) {
	if len(data)%sectorRecordSize != 0 {
		return nil, fmt.Errorf("%w: got %d bytes, expected multiple of %d", ErrInvalidSectorsLumpSize, len(data), sectorRecordSize)
	}

	out := make([]domain.Sector, 0, len(data)/sectorRecordSize)
	for i := 0; i < len(data); i += sectorRecordSize {
		out = append(out, domain.Sector{
			FloorHeight:    int16(binary.LittleEndian.Uint16(data[i : i+2])),
			CeilingHeight:  int16(binary.LittleEndian.Uint16(data[i+2 : i+4])),
			FloorTexture:   parseTextureName(data[i+4 : i+12]),
			CeilingTexture: parseTextureName(data[i+12 : i+20]),
			LightLevel:     binary.LittleEndian.Uint16(data[i+20 : i+22]),
			SpecialType:    binary.LittleEndian.Uint16(data[i+22 : i+24]),
			Tag:            binary.LittleEndian.Uint16(data[i+24 : i+26]),
		})
	}

	return out, nil
}

func parseTextureName(raw []byte) string {
	name := string(raw)
	name = strings.TrimRight(name, "\x00")
	name = strings.TrimSpace(name)
	return strings.ToUpper(name)
}
