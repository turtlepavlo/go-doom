package domain

import "errors"

var (
	ErrInvalidArchiveKind = errors.New("invalid archive kind")
	ErrNegativeOffset     = errors.New("negative offset")
	ErrNegativeSize       = errors.New("negative size")
	ErrNegativeCount      = errors.New("negative lump count")
	ErrEmptyLumpName      = errors.New("empty lump name")
	ErrLumpCountMismatch  = errors.New("lump count mismatch")
)
