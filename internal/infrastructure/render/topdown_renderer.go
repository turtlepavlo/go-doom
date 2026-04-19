package render

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

type TopDownRenderer struct {
	level  domain.Level
	width  int
	height int
	scale  float64
}

func NewTopDownRenderer(level domain.Level, width int, height int, zoom float64) *TopDownRenderer {
	scale := fitScale(level, width, height)
	if zoom > 0 {
		scale *= zoom
	}
	if scale <= 0 {
		scale = 1
	}

	return &TopDownRenderer{
		level:  level,
		width:  width,
		height: height,
		scale:  scale,
	}
}

func (r *TopDownRenderer) Draw(screen *ebiten.Image, frame domain.Frame) {
	screen.Fill(color.RGBA{R: 12, G: 14, B: 18, A: 255})

	for _, linedef := range r.level.Linedefs {
		start := int(linedef.StartVertex)
		end := int(linedef.EndVertex)
		if start < 0 || end < 0 || start >= len(r.level.Vertexes) || end >= len(r.level.Vertexes) {
			continue
		}

		v1 := r.level.Vertexes[start]
		v2 := r.level.Vertexes[end]
		x1, y1 := r.worldToScreen(float64(v1.X), float64(v1.Y), frame)
		x2, y2 := r.worldToScreen(float64(v2.X), float64(v2.Y), frame)
		vector.StrokeLine(screen, x1, y1, x2, y2, 1, color.RGBA{R: 80, G: 220, B: 150, A: 255}, false)
	}

	px, py := r.worldToScreen(float64(frame.PlayerX), float64(frame.PlayerY), frame)
	vector.DrawFilledCircle(screen, px, py, 4, color.RGBA{R: 255, G: 210, B: 70, A: 255}, false)
	vector.StrokeLine(screen, px-8, py, px+8, py, 1, color.RGBA{R: 255, G: 210, B: 70, A: 255}, false)
	vector.StrokeLine(screen, px, py-8, px, py+8, 1, color.RGBA{R: 255, G: 210, B: 70, A: 255}, false)
}

func (r *TopDownRenderer) worldToScreen(worldX float64, worldY float64, frame domain.Frame) (float32, float32) {
	dx := worldX - float64(frame.PlayerX)
	dy := worldY - float64(frame.PlayerY)

	screenX := float64(r.width)/2 + dx*r.scale
	screenY := float64(r.height)/2 - dy*r.scale
	return float32(screenX), float32(screenY)
}

func (r *TopDownRenderer) Layout() (int, int) {
	return r.width, r.height
}

func fitScale(level domain.Level, width int, height int) float64 {
	if len(level.Vertexes) == 0 || width <= 0 || height <= 0 {
		return 1
	}

	minX := float64(level.Vertexes[0].X)
	maxX := minX
	minY := float64(level.Vertexes[0].Y)
	maxY := minY

	for _, vertex := range level.Vertexes[1:] {
		x := float64(vertex.X)
		y := float64(vertex.Y)
		minX = math.Min(minX, x)
		maxX = math.Max(maxX, x)
		minY = math.Min(minY, y)
		maxY = math.Max(maxY, y)
	}

	worldW := maxX - minX
	worldH := maxY - minY
	if worldW <= 0 || worldH <= 0 {
		return 1
	}

	margin := 0.85
	scaleX := float64(width) / worldW
	scaleY := float64(height) / worldH
	return math.Min(scaleX, scaleY) * margin
}
