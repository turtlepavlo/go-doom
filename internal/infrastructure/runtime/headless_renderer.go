package runtime

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type HeadlessRenderer struct {
	mu             sync.Mutex
	writer         io.Writer
	framesRendered int
	lastFrame      domain.Frame
}

func NewHeadlessRenderer(writer io.Writer) *HeadlessRenderer {
	return &HeadlessRenderer{writer: writer}
}

func (r *HeadlessRenderer) Render(ctx context.Context, frame domain.Frame) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.framesRendered++
	r.lastFrame = frame
	if r.writer != nil {
		_, err := fmt.Fprintf(
			r.writer,
			"tick=%d pos=(%d,%d) running=%t\n",
			frame.Tick,
			frame.PlayerX,
			frame.PlayerY,
			frame.Running,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *HeadlessRenderer) Stats() (framesRendered int, lastFrame domain.Frame) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.framesRendered, r.lastFrame
}
