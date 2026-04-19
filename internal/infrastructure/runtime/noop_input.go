package runtime

import (
	"context"

	"github.com/turtlepavlo/go-doom/internal/application/dto"
)

type NoopInput struct{}

func NewNoopInput() *NoopInput {
	return &NoopInput{}
}

func (n *NoopInput) Poll(ctx context.Context) ([]dto.RawInput, error) {
	return []dto.RawInput{}, nil
}
