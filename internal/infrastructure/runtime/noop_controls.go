package runtime

import (
	"context"

	"github.com/turtlepavlo/go-doom/internal/transport/controls"
)

type NoopControlPoller struct{}

func NewNoopControlPoller() *NoopControlPoller {
	return &NoopControlPoller{}
}

func (n *NoopControlPoller) Poll(_ context.Context) ([]controls.RawControl, error) {
	return []controls.RawControl{}, nil
}
