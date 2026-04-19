package domain

import (
	"fmt"
	"strings"
)

type ArchiveKind string

const (
	ArchiveKindIWAD ArchiveKind = "IWAD"
	ArchiveKindPWAD ArchiveKind = "PWAD"
)

type Header struct {
	Kind            ArchiveKind
	LumpCount       int
	DirectoryOffset int
}

type Lump struct {
	Name   string
	Offset int
	Size   int
}

type Map struct {
	Name  string
	Lumps []Lump
}

type Archive struct {
	Header Header
	Lumps  []Lump
	Maps   []Map
}

func NewHeader(kind string, lumpCount int, directoryOffset int) (Header, error) {
	normalizedKind := ArchiveKind(strings.ToUpper(strings.TrimSpace(kind)))
	if normalizedKind != ArchiveKindIWAD && normalizedKind != ArchiveKindPWAD {
		return Header{}, fmt.Errorf("%w: %q", ErrInvalidArchiveKind, kind)
	}
	if lumpCount < 0 {
		return Header{}, ErrNegativeCount
	}
	if directoryOffset < 0 {
		return Header{}, ErrNegativeOffset
	}
	return Header{
		Kind:            normalizedKind,
		LumpCount:       lumpCount,
		DirectoryOffset: directoryOffset,
	}, nil
}

func NewLump(name string, offset int, size int) (Lump, error) {
	normalizedName := strings.ToUpper(strings.TrimSpace(name))
	if normalizedName == "" {
		return Lump{}, ErrEmptyLumpName
	}
	if offset < 0 {
		return Lump{}, ErrNegativeOffset
	}
	if size < 0 {
		return Lump{}, ErrNegativeSize
	}
	return Lump{
		Name:   normalizedName,
		Offset: offset,
		Size:   size,
	}, nil
}

func NewArchive(header Header, lumps []Lump, maps []Map) (Archive, error) {
	if header.LumpCount != len(lumps) {
		return Archive{}, fmt.Errorf("%w: header=%d parsed=%d", ErrLumpCountMismatch, header.LumpCount, len(lumps))
	}
	return Archive{
		Header: header,
		Lumps:  append([]Lump(nil), lumps...),
		Maps:   append([]Map(nil), maps...),
	}, nil
}
