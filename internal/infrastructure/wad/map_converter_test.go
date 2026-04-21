package wad

import (
	"encoding/binary"
	"errors"
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/loadmap"
)

func TestMapConverterToLevel(t *testing.T) {
	converter := NewToLevelConvert()
	raw := loadmap.RawMap{
		Name:     "E1M1",
		Things:   buildThing(100, -50, 90, 1, 7),
		Linedefs: buildLinedef(0, 1, 1, 0, 0, 0, 65535),
		Sidedefs: buildSidedef(0, 0, "STARTAN3", "-", "STARTAN2", 0),
		Vertexes: buildVertex(0, 0, 128, 0),
		Sectors:  buildSector(0, 128, "FLOOR4_8", "CEIL5_2", 160, 0, 1),
	}

	level, err := converter.Level(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if level.Name != "E1M1" {
		t.Fatalf("expected map E1M1, got %s", level.Name)
	}
	if len(level.Things) != 1 {
		t.Fatalf("expected 1 thing, got %d", len(level.Things))
	}
	if len(level.Vertexes) != 2 {
		t.Fatalf("expected 2 vertexes, got %d", len(level.Vertexes))
	}
	if len(level.Sectors) != 1 {
		t.Fatalf("expected 1 sector, got %d", len(level.Sectors))
	}
}

func TestMapConverterToLevelInvalidLumpSize(t *testing.T) {
	converter := NewToLevelConvert()
	raw := loadmap.RawMap{
		Name:     "E1M1",
		Things:   []byte{1},
		Linedefs: []byte{},
		Sidedefs: []byte{},
		Vertexes: []byte{},
		Sectors:  []byte{},
	}

	_, err := converter.Level(raw)
	if !errors.Is(err, ErrInvalidThingsLumpSize) {
		t.Fatalf("expected ErrInvalidThingsLumpSize, got %v", err)
	}
}

func buildThing(x, y int16, angle, thingType, flags uint16) []byte {
	data := make([]byte, 10)
	binary.LittleEndian.PutUint16(data[0:2], uint16(x))
	binary.LittleEndian.PutUint16(data[2:4], uint16(y))
	binary.LittleEndian.PutUint16(data[4:6], angle)
	binary.LittleEndian.PutUint16(data[6:8], thingType)
	binary.LittleEndian.PutUint16(data[8:10], flags)
	return data
}

func buildLinedef(start, end, lineFlags, special, tag, right, left uint16) []byte {
	data := make([]byte, 14)
	binary.LittleEndian.PutUint16(data[0:2], start)
	binary.LittleEndian.PutUint16(data[2:4], end)
	binary.LittleEndian.PutUint16(data[4:6], lineFlags)
	binary.LittleEndian.PutUint16(data[6:8], special)
	binary.LittleEndian.PutUint16(data[8:10], tag)
	binary.LittleEndian.PutUint16(data[10:12], right)
	binary.LittleEndian.PutUint16(data[12:14], left)
	return data
}

func buildSidedef(xOffset, yOffset int16, upper, lower, middle string, sector uint16) []byte {
	data := make([]byte, 30)
	binary.LittleEndian.PutUint16(data[0:2], uint16(xOffset))
	binary.LittleEndian.PutUint16(data[2:4], uint16(yOffset))
	copyTexture(data[4:12], upper)
	copyTexture(data[12:20], lower)
	copyTexture(data[20:28], middle)
	binary.LittleEndian.PutUint16(data[28:30], sector)
	return data
}

func buildVertex(x1, y1, x2, y2 int16) []byte {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data[0:2], uint16(x1))
	binary.LittleEndian.PutUint16(data[2:4], uint16(y1))
	binary.LittleEndian.PutUint16(data[4:6], uint16(x2))
	binary.LittleEndian.PutUint16(data[6:8], uint16(y2))
	return data
}

func buildSector(floor, ceiling int16, floorTex, ceilingTex string, light, special, tag uint16) []byte {
	data := make([]byte, 26)
	binary.LittleEndian.PutUint16(data[0:2], uint16(floor))
	binary.LittleEndian.PutUint16(data[2:4], uint16(ceiling))
	copyTexture(data[4:12], floorTex)
	copyTexture(data[12:20], ceilingTex)
	binary.LittleEndian.PutUint16(data[20:22], light)
	binary.LittleEndian.PutUint16(data[22:24], special)
	binary.LittleEndian.PutUint16(data[24:26], tag)
	return data
}

func copyTexture(dst []byte, texture string) {
	for i := range dst {
		dst[i] = 0
	}
	copy(dst, []byte(texture))
}
