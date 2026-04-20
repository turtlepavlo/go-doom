package runtime

import (
	"errors"
	"math"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	playerCollisionRadius = 18.0
	minPassageHeight      = 56.0
)

var ErrNilLevel = errors.New("nil level")

type blockingSegment struct {
	ax float64
	ay float64
	bx float64
	by float64
}

type LevelSimulation struct {
	engine   *domain.Engine
	segments []blockingSegment
}

func NewLevelSimulation(engine *domain.Engine, level *domain.Level) (*LevelSimulation, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	if level == nil {
		return nil, ErrNilLevel
	}

	return &LevelSimulation{
		engine:   engine,
		segments: collectBlockingSegments(*level),
	}, nil
}

func (s *LevelSimulation) Step(commands []domain.Command) (domain.Frame, error) {
	prev := s.engine.State()
	frame := s.engine.Step(commands)
	if !frame.Running {
		return frame, nil
	}

	if !s.collides(float64(frame.PlayerX), float64(frame.PlayerY)) {
		return frame, nil
	}

	nextX := float64(frame.PlayerX)
	nextY := float64(frame.PlayerY)
	prevX := float64(prev.PlayerX)
	prevY := float64(prev.PlayerY)

	switch {
	case !s.collides(nextX, prevY):
		s.engine.SetPlayerPosition(int(math.Round(nextX)), int(math.Round(prevY)))
	case !s.collides(prevX, nextY):
		s.engine.SetPlayerPosition(int(math.Round(prevX)), int(math.Round(nextY)))
	default:
		s.engine.SetPlayerPosition(prev.PlayerX, prev.PlayerY)
	}

	return s.engine.Frame(), nil
}

func (s *LevelSimulation) collides(playerX float64, playerY float64) bool {
	for _, segment := range s.segments {
		if pointSegmentDistance(playerX, playerY, segment.ax, segment.ay, segment.bx, segment.by) <= playerCollisionRadius {
			return true
		}
	}
	return false
}

func collectBlockingSegments(level domain.Level) []blockingSegment {
	segments := make([]blockingSegment, 0, len(level.Linedefs))

	for _, line := range level.Linedefs {
		aIdx := int(line.StartVertex)
		bIdx := int(line.EndVertex)
		if aIdx < 0 || bIdx < 0 || aIdx >= len(level.Vertexes) || bIdx >= len(level.Vertexes) {
			continue
		}

		if !isBlockingLine(level, line) {
			continue
		}

		a := level.Vertexes[aIdx]
		b := level.Vertexes[bIdx]
		segments = append(segments, blockingSegment{
			ax: float64(a.X),
			ay: float64(a.Y),
			bx: float64(b.X),
			by: float64(b.Y),
		})
	}

	return segments
}

func isBlockingLine(level domain.Level, line domain.Linedef) bool {
	const noSide = math.MaxUint16

	rightIdx := int(line.RightSide)
	leftIdx := int(line.LeftSide)
	if line.RightSide == noSide || rightIdx < 0 || rightIdx >= len(level.Sidedefs) {
		return true
	}
	if line.LeftSide == noSide || leftIdx < 0 || leftIdx >= len(level.Sidedefs) {
		return true
	}

	rightSectorIdx := int(level.Sidedefs[rightIdx].Sector)
	leftSectorIdx := int(level.Sidedefs[leftIdx].Sector)
	if rightSectorIdx < 0 || rightSectorIdx >= len(level.Sectors) {
		return true
	}
	if leftSectorIdx < 0 || leftSectorIdx >= len(level.Sectors) {
		return true
	}

	rightSector := level.Sectors[rightSectorIdx]
	leftSector := level.Sectors[leftSectorIdx]

	openTop := math.Min(float64(rightSector.CeilingHeight), float64(leftSector.CeilingHeight))
	openBottom := math.Max(float64(rightSector.FloorHeight), float64(leftSector.FloorHeight))
	return (openTop - openBottom) < minPassageHeight
}

func pointSegmentDistance(px float64, py float64, ax float64, ay float64, bx float64, by float64) float64 {
	abx := bx - ax
	aby := by - ay
	apx := px - ax
	apy := py - ay

	abLenSq := abx*abx + aby*aby
	if abLenSq == 0 {
		dx := px - ax
		dy := py - ay
		return math.Sqrt(dx*dx + dy*dy)
	}

	t := (apx*abx + apy*aby) / abLenSq
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	closestX := ax + t*abx
	closestY := ay + t*aby
	dx := px - closestX
	dy := py - closestY
	return math.Sqrt(dx*dx + dy*dy)
}
