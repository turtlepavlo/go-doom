package wad

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
)

const (
	headerSize          = 12
	directoryEntrySize  = 16
	lumpNameFieldLength = 8
)

var (
	ErrWADTooSmall          = errors.New("wad file too small")
	ErrInvalidDirectory     = errors.New("invalid directory bounds")
	ErrLumpOutOfBounds      = errors.New("lump payload out of file bounds")
	ErrDirectoryEntryBroken = errors.New("directory entry is truncated")
)

type BinaryReader struct{}

func NewBinaryReader() *BinaryReader {
	return &BinaryReader{}
}

func (r *BinaryReader) ReadArchive(ctx context.Context, path string) (dto.RawArchive, error) {
	select {
	case <-ctx.Done():
		return dto.RawArchive{}, ctx.Err()
	default:
	}

	fileData, err := os.ReadFile(path)
	if err != nil {
		return dto.RawArchive{}, fmt.Errorf("read file %q: %w", path, err)
	}

	archive, err := parseBinary(fileData)
	if err != nil {
		return dto.RawArchive{}, fmt.Errorf("parse %q: %w", path, err)
	}

	return archive, nil
}

func parseBinary(fileData []byte) (dto.RawArchive, error) {
	if len(fileData) < headerSize {
		return dto.RawArchive{}, ErrWADTooSmall
	}

	wadType := string(fileData[0:4])
	lumpCount := binary.LittleEndian.Uint32(fileData[4:8])
	directoryOffset := binary.LittleEndian.Uint32(fileData[8:12])

	directorySize := uint64(lumpCount) * directoryEntrySize
	directoryEnd := uint64(directoryOffset) + directorySize
	if directoryEnd > uint64(len(fileData)) {
		return dto.RawArchive{}, fmt.Errorf("%w: offset=%d size=%d file=%d", ErrInvalidDirectory, directoryOffset, directorySize, len(fileData))
	}

	lumps := make([]dto.RawLump, 0, lumpCount)
	for i := uint32(0); i < lumpCount; i++ {
		entryStart := uint64(directoryOffset) + uint64(i)*directoryEntrySize
		entryEnd := entryStart + directoryEntrySize
		if entryEnd > uint64(len(fileData)) {
			return dto.RawArchive{}, fmt.Errorf("%w at index %d", ErrDirectoryEntryBroken, i)
		}

		pos := int(entryStart)
		lumpOffset := binary.LittleEndian.Uint32(fileData[pos : pos+4])
		lumpSize := binary.LittleEndian.Uint32(fileData[pos+4 : pos+8])
		lumpName := parseLumpName(fileData[pos+8 : pos+8+lumpNameFieldLength])

		payloadEnd := uint64(lumpOffset) + uint64(lumpSize)
		if payloadEnd > uint64(len(fileData)) {
			return dto.RawArchive{}, fmt.Errorf("%w: name=%s end=%d file=%d", ErrLumpOutOfBounds, lumpName, payloadEnd, len(fileData))
		}

		lumps = append(lumps, dto.RawLump{
			Name:   lumpName,
			Offset: lumpOffset,
			Size:   lumpSize,
		})
	}

	return dto.RawArchive{
		Type:            wadType,
		LumpCount:       lumpCount,
		DirectoryOffset: directoryOffset,
		Lumps:           lumps,
	}, nil
}

func parseLumpName(raw []byte) string {
	name := string(raw)
	name = strings.TrimRight(name, "\x00")
	return strings.TrimSpace(name)
}
