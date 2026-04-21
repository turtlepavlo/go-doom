package wad

import (
	"testing"

	"github.com/turtlepavlo/go-doom/internal/application/loadiwad"
)

func TestArchiveConverterToDomain(t *testing.T) {
	converter := NewToArchiveConvert()

	raw := loadiwad.RawArchive{
		Type:            "IWAD",
		LumpCount:       4,
		DirectoryOffset: 12,
		Lumps: []loadiwad.RawLump{
			{Name: "PLAYPAL", Offset: 0, Size: 0},
			{Name: "E1M1", Offset: 0, Size: 0},
			{Name: "THINGS", Offset: 0, Size: 0},
			{Name: "LINEDEFS", Offset: 0, Size: 0},
		},
	}

	archive, err := converter.Archive(raw)
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

func TestArchiveConverterToDomainInvalidType(t *testing.T) {
	converter := NewToArchiveConvert()

	_, err := converter.Archive(loadiwad.RawArchive{
		Type:            "ABCD",
		LumpCount:       0,
		DirectoryOffset: 12,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
