package render

import (
	"image/color"
	"math"
	"slices"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	nearPlaneDistance = 2.0
	playerEyeHeight   = 41.0
	baseFOVRadians    = math.Pi / 2
)

type worldWall struct {
	ax float64
	ay float64
	bx float64
	by float64

	floorHeight   float64
	ceilingHeight float64
}

type projectedWall struct {
	x1 float64
	x2 float64

	top1 float64
	top2 float64
	bot1 float64
	bot2 float64

	depth float64
}

type FirstPersonRenderer struct {
	width  int
	height int
	focal  float64

	walls []worldWall
}

func NewFirstPersonRenderer(level domain.Level, width int, height int, zoom float64) *FirstPersonRenderer {
	fov := baseFOVRadians
	if zoom > 0 {
		fov = baseFOVRadians / zoom
	}
	if fov < 0.35 {
		fov = 0.35
	}
	if fov > 2.5 {
		fov = 2.5
	}

	focal := float64(width) / (2 * math.Tan(fov/2))
	if focal <= 0 || math.IsInf(focal, 0) || math.IsNaN(focal) {
		focal = float64(width) / 2
	}

	return &FirstPersonRenderer{
		width:  width,
		height: height,
		focal:  focal,
		walls:  collectSolidWalls(level),
	}
}

func (r *FirstPersonRenderer) Draw(screen *ebiten.Image, frame domain.Frame) {
	screen.Fill(color.RGBA{R: 8, G: 8, B: 10, A: 255})

	horizon := float32(r.height) / 2
	vector.DrawFilledRect(
		screen,
		0,
		0,
		float32(r.width),
		horizon,
		color.RGBA{R: 26, G: 28, B: 36, A: 255},
		false,
	)
	vector.DrawFilledRect(
		screen,
		0,
		horizon,
		float32(r.width),
		float32(r.height)-horizon,
		color.RGBA{R: 36, G: 30, B: 24, A: 255},
		false,
	)

	projected := r.projectWalls(frame)
	slices.SortFunc(projected, func(a projectedWall, b projectedWall) int {
		if a.depth > b.depth {
			return -1
		}
		if a.depth < b.depth {
			return 1
		}
		return 0
	})

	for _, wall := range projected {
		r.drawProjectedWall(screen, wall)
	}

	cx := float32(r.width) / 2
	cy := float32(r.height) / 2
	vector.StrokeLine(screen, cx-8, cy, cx+8, cy, 1, color.RGBA{R: 245, G: 205, B: 90, A: 255}, false)
	vector.StrokeLine(screen, cx, cy-8, cx, cy+8, 1, color.RGBA{R: 245, G: 205, B: 90, A: 255}, false)
}

func (r *FirstPersonRenderer) Layout() (int, int) {
	return r.width, r.height
}

func (r *FirstPersonRenderer) projectWalls(frame domain.Frame) []projectedWall {
	out := make([]projectedWall, 0, len(r.walls))
	cosA := math.Cos(frame.Angle)
	sinA := math.Sin(frame.Angle)
	centerX := float64(r.width) / 2
	centerY := float64(r.height) / 2

	for _, wall := range r.walls {
		ax, az := toCamera(wall.ax, wall.ay, float64(frame.PlayerX), float64(frame.PlayerY), cosA, sinA)
		bx, bz := toCamera(wall.bx, wall.by, float64(frame.PlayerX), float64(frame.PlayerY), cosA, sinA)

		if az <= nearPlaneDistance && bz <= nearPlaneDistance {
			continue
		}

		if az <= nearPlaneDistance {
			t := (nearPlaneDistance - az) / (bz - az)
			ax = ax + (bx-ax)*t
			az = nearPlaneDistance
		}
		if bz <= nearPlaneDistance {
			t := (nearPlaneDistance - bz) / (az - bz)
			bx = bx + (ax-bx)*t
			bz = nearPlaneDistance
		}

		sx1 := centerX + (ax/az)*r.focal
		sx2 := centerX + (bx/bz)*r.focal
		if (sx1 < 0 && sx2 < 0) || (sx1 >= float64(r.width) && sx2 >= float64(r.width)) {
			continue
		}

		top1 := centerY - ((wall.ceilingHeight-playerEyeHeight)/az)*r.focal
		top2 := centerY - ((wall.ceilingHeight-playerEyeHeight)/bz)*r.focal
		bot1 := centerY - ((wall.floorHeight-playerEyeHeight)/az)*r.focal
		bot2 := centerY - ((wall.floorHeight-playerEyeHeight)/bz)*r.focal

		out = append(out, projectedWall{
			x1: sx1,
			x2: sx2,

			top1: top1,
			top2: top2,
			bot1: bot1,
			bot2: bot2,

			depth: (az + bz) / 2,
		})
	}

	return out
}

func (r *FirstPersonRenderer) drawProjectedWall(screen *ebiten.Image, wall projectedWall) {
	x1 := wall.x1
	x2 := wall.x2
	top1 := wall.top1
	top2 := wall.top2
	bot1 := wall.bot1
	bot2 := wall.bot2

	if x1 > x2 {
		x1, x2 = x2, x1
		top1, top2 = top2, top1
		bot1, bot2 = bot2, bot1
	}

	if x2-x1 < 0.0001 {
		return
	}

	left := intMax(0, int(math.Ceil(x1)))
	right := intMin(r.width-1, int(math.Floor(x2)))
	if right < left {
		return
	}

	shade := clamp(1-(wall.depth/2200.0), 0.18, 1.0)
	wallColor := color.RGBA{
		R: uint8(170 * shade),
		G: uint8(150 * shade),
		B: uint8(140 * shade),
		A: 255,
	}

	for x := left; x <= right; x++ {
		t := (float64(x) - x1) / (x2 - x1)
		top := lerp(top1, top2, t)
		bot := lerp(bot1, bot2, t)
		if bot < top {
			top, bot = bot, top
		}

		if bot < 0 || top >= float64(r.height) {
			continue
		}

		top = clamp(top, 0, float64(r.height-1))
		bot = clamp(bot, 0, float64(r.height-1))
		vector.StrokeLine(
			screen,
			float32(x),
			float32(top),
			float32(x),
			float32(bot),
			1,
			wallColor,
			false,
		)
	}
}

func collectSolidWalls(level domain.Level) []worldWall {
	walls := make([]worldWall, 0, len(level.Linedefs))

	for _, line := range level.Linedefs {
		aIdx := int(line.StartVertex)
		bIdx := int(line.EndVertex)
		if aIdx < 0 || bIdx < 0 || aIdx >= len(level.Vertexes) || bIdx >= len(level.Vertexes) {
			continue
		}

		sector, ok := wallSector(level, line)
		if !ok {
			continue
		}

		av := level.Vertexes[aIdx]
		bv := level.Vertexes[bIdx]
		walls = append(walls, worldWall{
			ax:            float64(av.X),
			ay:            float64(av.Y),
			bx:            float64(bv.X),
			by:            float64(bv.Y),
			floorHeight:   float64(sector.FloorHeight),
			ceilingHeight: float64(sector.CeilingHeight),
		})
	}

	return walls
}

func wallSector(level domain.Level, line domain.Linedef) (domain.Sector, bool) {
	const noSide = math.MaxUint16

	oneSided := line.RightSide == noSide || line.LeftSide == noSide
	if !oneSided {
		return domain.Sector{}, false
	}

	sideIdx := int(line.RightSide)
	if line.RightSide == noSide {
		sideIdx = int(line.LeftSide)
	}
	if sideIdx < 0 || sideIdx >= len(level.Sidedefs) {
		return domain.Sector{}, false
	}

	sectorIdx := int(level.Sidedefs[sideIdx].Sector)
	if sectorIdx < 0 || sectorIdx >= len(level.Sectors) {
		return domain.Sector{}, false
	}

	return level.Sectors[sectorIdx], true
}

func toCamera(worldX float64, worldY float64, playerX float64, playerY float64, cosA float64, sinA float64) (x float64, z float64) {
	dx := worldX - playerX
	dy := worldY - playerY

	z = dx*cosA + dy*sinA
	x = -dx*sinA + dy*cosA
	return x, z
}

func lerp(a float64, b float64, t float64) float64 {
	return a + (b-a)*t
}

func clamp(v float64, minV float64, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func intMin(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func intMax(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
