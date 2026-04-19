package wad

import (
	"encoding/binary"
	"testing"
)

type testEntry struct {
	offset uint32
	size   uint32
	name   string
}

func TestParseBinary_ValidArchive(t *testing.T) {
	data := buildWAD("IWAD", []testEntry{
		{offset: 0, size: 0, name: "PLAYPAL"},
		{offset: 0, size: 0, name: "E1M1"},
		{offset: 0, size: 0, name: "THINGS"},
	})

	archive, err := parseBinary(data)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if archive.Type != "IWAD" {
		t.Fatalf("expected IWAD, got %q", archive.Type)
	}
	if archive.LumpCount != 3 {
		t.Fatalf("expected 3 lumps, got %d", archive.LumpCount)
	}
	if got := archive.Lumps[1].Name; got != "E1M1" {
		t.Fatalf("expected map marker E1M1, got %q", got)
	}
}

func TestParseBinary_InvalidDirectoryBounds(t *testing.T) {
	data := make([]byte, headerSize)
	copy(data[0:4], []byte("IWAD"))
	binary.LittleEndian.PutUint32(data[4:8], 1)
	binary.LittleEndian.PutUint32(data[8:12], 128)

	_, err := parseBinary(data)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func buildWAD(wadType string, entries []testEntry) []byte {
	directoryOffset := uint32(headerSize)
	directorySize := uint32(len(entries) * directoryEntrySize)
	total := int(directoryOffset + directorySize)
	data := make([]byte, total)

	copy(data[0:4], []byte(wadType))
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(entries)))
	binary.LittleEndian.PutUint32(data[8:12], directoryOffset)

	for i, entry := range entries {
		pos := int(directoryOffset) + i*directoryEntrySize
		binary.LittleEndian.PutUint32(data[pos:pos+4], entry.offset)
		binary.LittleEndian.PutUint32(data[pos+4:pos+8], entry.size)

		nameField := make([]byte, lumpNameFieldLength)
		copy(nameField, []byte(entry.name))
		copy(data[pos+8:pos+8+lumpNameFieldLength], nameField)
	}

	return data
}
