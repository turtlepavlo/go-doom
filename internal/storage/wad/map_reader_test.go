package wad

import (
	"context"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type payloadEntry struct {
	name string
	data []byte
}

func TestMapReaderReadMap(t *testing.T) {
	archive := buildWADWithPayloads("IWAD", []payloadEntry{
		{name: "PLAYPAL", data: []byte{1, 2, 3}},
		{name: "E1M1", data: []byte{}},
		{name: "THINGS", data: make([]byte, 10)},
		{name: "LINEDEFS", data: make([]byte, 14)},
		{name: "SIDEDEFS", data: make([]byte, 30)},
		{name: "VERTEXES", data: make([]byte, 8)},
		{name: "SECTORS", data: make([]byte, 26)},
		{name: "E1M2", data: []byte{}},
		{name: "THINGS", data: make([]byte, 10)},
		{name: "LINEDEFS", data: make([]byte, 14)},
		{name: "SIDEDEFS", data: make([]byte, 30)},
		{name: "VERTEXES", data: make([]byte, 8)},
		{name: "SECTORS", data: make([]byte, 26)},
	})

	wadPath := writeTempWAD(t, archive)
	reader := NewMapReader()

	rawMap, err := reader.ReadMap(context.Background(), wadPath, "E1M1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rawMap.Name != "E1M1" {
		t.Fatalf("expected E1M1, got %s", rawMap.Name)
	}
	if len(rawMap.Things) != 10 {
		t.Fatalf("expected THINGS size 10, got %d", len(rawMap.Things))
	}
	if len(rawMap.Linedefs) != 14 {
		t.Fatalf("expected LINEDEFS size 14, got %d", len(rawMap.Linedefs))
	}
	if len(rawMap.Sidedefs) != 30 {
		t.Fatalf("expected SIDEDEFS size 30, got %d", len(rawMap.Sidedefs))
	}
	if len(rawMap.Vertexes) != 8 {
		t.Fatalf("expected VERTEXES size 8, got %d", len(rawMap.Vertexes))
	}
	if len(rawMap.Sectors) != 26 {
		t.Fatalf("expected SECTORS size 26, got %d", len(rawMap.Sectors))
	}
}

func TestMapReaderReadMapNotFound(t *testing.T) {
	archive := buildWADWithPayloads("IWAD", []payloadEntry{
		{name: "E1M1", data: []byte{}},
	})
	wadPath := writeTempWAD(t, archive)

	reader := NewMapReader()
	_, err := reader.ReadMap(context.Background(), wadPath, "E1M9")
	if !errors.Is(err, ErrMapNotFound) {
		t.Fatalf("expected ErrMapNotFound, got %v", err)
	}
}

func TestMapReaderReadMapMissingRequiredLump(t *testing.T) {
	archive := buildWADWithPayloads("IWAD", []payloadEntry{
		{name: "E1M1", data: []byte{}},
		{name: "THINGS", data: make([]byte, 10)},
		{name: "LINEDEFS", data: make([]byte, 14)},
		{name: "SIDEDEFS", data: make([]byte, 30)},
		{name: "VERTEXES", data: make([]byte, 8)},
	})
	wadPath := writeTempWAD(t, archive)

	reader := NewMapReader()
	_, err := reader.ReadMap(context.Background(), wadPath, "E1M1")
	if !errors.Is(err, ErrMissingSectorsLump) {
		t.Fatalf("expected ErrMissingSectorsLump, got %v", err)
	}
}

func writeTempWAD(t *testing.T, bytes []byte) string {
	t.Helper()

	dir := t.TempDir()
	wadPath := filepath.Join(dir, "test.wad")
	if err := os.WriteFile(wadPath, bytes, 0o644); err != nil {
		t.Fatalf("write wad: %v", err)
	}
	return wadPath
}

func buildWADWithPayloads(wadType string, entries []payloadEntry) []byte {
	payloadOffset := headerSize
	payloadSize := 0
	for _, entry := range entries {
		payloadSize += len(entry.data)
	}

	directoryOffset := payloadOffset + payloadSize
	totalSize := directoryOffset + len(entries)*directoryEntrySize
	data := make([]byte, totalSize)

	copy(data[0:4], []byte(wadType))
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(entries)))
	binary.LittleEndian.PutUint32(data[8:12], uint32(directoryOffset))

	cursor := payloadOffset
	for i, entry := range entries {
		entryOffset := cursor
		if len(entry.data) > 0 {
			copy(data[cursor:cursor+len(entry.data)], entry.data)
			cursor += len(entry.data)
		}

		dirPos := directoryOffset + i*directoryEntrySize
		binary.LittleEndian.PutUint32(data[dirPos:dirPos+4], uint32(entryOffset))
		binary.LittleEndian.PutUint32(data[dirPos+4:dirPos+8], uint32(len(entry.data)))

		nameField := make([]byte, lumpNameFieldLength)
		copy(nameField, []byte(entry.name))
		copy(data[dirPos+8:dirPos+8+lumpNameFieldLength], nameField)
	}

	return data
}
