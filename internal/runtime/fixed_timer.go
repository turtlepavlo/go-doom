package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrInvalidTickRate = errors.New("invalid tick rate")
	ErrTimerClosed     = errors.New("timer is closed")
)

type FixedTimer struct {
	mu     sync.Mutex
	ticker *time.Ticker
	closed bool
}

func NewFixedTimer(ticksPerSecond int) (*FixedTimer, error) {
	if ticksPerSecond <= 0 {
		return nil, ErrInvalidTickRate
	}
	interval := time.Second / time.Duration(ticksPerSecond)
	return &FixedTimer{
		ticker: time.NewTicker(interval),
	}, nil
}

func (t *FixedTimer) Wait(ctx context.Context) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return ErrTimerClosed
	}
	ticker := t.ticker
	t.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ticker.C:
		return nil
	}
}

func (t *FixedTimer) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.ticker.Stop()
	t.closed = true
	return nil
}
