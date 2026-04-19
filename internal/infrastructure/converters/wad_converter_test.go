package converters

import (
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
)

func TestWADConverter_ToDomain(t *testing.T) {
	converter := NewWADConverter()

	raw := dto.RawArchive{
		Type:            "IWAD",
		LumpCount:       4,
		DirectoryOffset: 12,
		Lumps: []dto.RawLump{
			{Name: "PLAYPAL", Offset: 0, Size: 0},
			{Name: "E1M1", Offset: 0, Size: 0},
			{Name: "THINGS", Offset: 0, Size: 0},
			{Name: "LINEDEFS", Offset: 0, Size: 0},
		},
	}

	archive, err := converter.ToDomain(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if archive.Header.Kind != "IWAD" {
		t.Fatalf("expected IWAD kind, got %q", archive.Header.Kind)
	}
	if len(archive.Maps) != 1 {
		t.Fatalf("expected one map, got %d", len(archive.Maps))
	}
	if archive.Maps[0].Name != "E1M1" {
		t.Fatalf("expected map E1M1, got %q", archive.Maps[0].Name)
	}
	if len(archive.Maps[0].Lumps) != 2 {
		t.Fatalf("expected 2 map lumps, got %d", len(archive.Maps[0].Lumps))
	}
}

func TestWADConverter_ToDomain_InvalidType(t *testing.T) {
	converter := NewWADConverter()

	_, err := converter.ToDomain(dto.RawArchive{
		Type:            "ABCD",
		LumpCount:       0,
		DirectoryOffset: 12,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
