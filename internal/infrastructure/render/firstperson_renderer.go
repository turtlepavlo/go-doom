package render

import (
	"hash/fnv"
	"image/color"
	"math"
	"slices"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	nearPlaneDistance = 2.0
	playerEyeHeight   = 41.0
	baseFOVRadians    = math.Pi / 2

	internalRenderWidth  = 320
	internalRenderHeight = 200
	statusBarHeight      = 32
)

type worldWall struct {
	ax float64
	ay float64
	bx float64
	by float64

	bottomZ float64
	topZ    float64

	lightLevel float64
	material   string
}

type projectedWall struct {
	x1 float64
	x2 float64

	top1 float64
	top2 float64
	bot1 float64
	bot2 float64

	z1 float64
	z2 float64

	material string
	light    float64
	depth    float64
}

type FirstPersonRenderer struct {
	width     int
	height    int
	focal     float64
	viewportH int
	statusY   int
	statusH   int

	walls []worldWall

	backbuffer *ebiten.Image
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

	viewportH := internalRenderHeight - statusBarHeight
	focal := float64(internalRenderWidth) / (2 * math.Tan(fov/2))
	if focal <= 0 || math.IsInf(focal, 0) || math.IsNaN(focal) {
		focal = float64(internalRenderWidth) / 2
	}

	return &FirstPersonRenderer{
		width:     width,
		height:    height,
		focal:     focal,
		viewportH: viewportH,
		statusY:   viewportH,
		statusH:   statusBarHeight,
		walls:     collectRenderableWalls(level),
	}
}

func (r *FirstPersonRenderer) Draw(screen *ebiten.Image, frame domain.Frame) {
	r.ensureBackbuffer()
	r.drawBackground()

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

	depthBuffer := make([]float64, internalRenderWidth)
	for i := range depthBuffer {
		depthBuffer[i] = math.Inf(1)
	}

	for _, wall := range projected {
		r.drawProjectedWall(wall, depthBuffer)
	}

	r.drawWeapon(frame)
	r.drawStatusBar(frame)

	screen.Fill(color.RGBA{R: 0, G: 0, B: 0, A: 255})
	opts := &ebiten.DrawImageOptions{}
	opts.GeoM.Scale(
		float64(r.width)/float64(internalRenderWidth),
		float64(r.height)/float64(internalRenderHeight),
	)
	screen.DrawImage(r.backbuffer, opts)
}

func (r *FirstPersonRenderer) Layout() (int, int) {
	return r.width, r.height
}

func (r *FirstPersonRenderer) ensureBackbuffer() {
	if r.backbuffer == nil {
		r.backbuffer = ebiten.NewImage(internalRenderWidth, internalRenderHeight)
	}
}

func (r *FirstPersonRenderer) drawBackground() {
	half := r.viewportH / 2
	for y := 0; y < r.viewportH; y++ {
		var col color.RGBA
		if y < half {
			t := float64(y) / float64(half)
			col = color.RGBA{
				R: uint8(24 + 26*t),
				G: uint8(24 + 20*t),
				B: uint8(40 + 30*t),
				A: 255,
			}
		} else {
			t := float64(y-half) / float64(half)
			col = color.RGBA{
				R: uint8(40 + 36*t),
				G: uint8(28 + 26*t),
				B: uint8(22 + 18*t),
				A: 255,
			}
		}
		for x := 0; x < internalRenderWidth; x++ {
			r.backbuffer.Set(x, y, col)
		}
	}
}

func (r *FirstPersonRenderer) projectWalls(frame domain.Frame) []projectedWall {
	out := make([]projectedWall, 0, len(r.walls))
	cosA := math.Cos(frame.Angle)
	sinA := math.Sin(frame.Angle)
	centerX := float64(internalRenderWidth) / 2
	centerY := float64(r.viewportH) / 2

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
		if (sx1 < 0 && sx2 < 0) || (sx1 >= internalRenderWidth && sx2 >= internalRenderWidth) {
			continue
		}

		top1 := centerY - ((wall.topZ-playerEyeHeight)/az)*r.focal
		top2 := centerY - ((wall.topZ-playerEyeHeight)/bz)*r.focal
		bot1 := centerY - ((wall.bottomZ-playerEyeHeight)/az)*r.focal
		bot2 := centerY - ((wall.bottomZ-playerEyeHeight)/bz)*r.focal

		out = append(out, projectedWall{
			x1: sx1,
			x2: sx2,

			top1: top1,
			top2: top2,
			bot1: bot1,
			bot2: bot2,

			z1: az,
			z2: bz,

			material: wall.material,
			light:    wall.lightLevel,
			depth:    (az + bz) / 2,
		})
	}

	return out
}

func (r *FirstPersonRenderer) drawProjectedWall(wall projectedWall, depthBuffer []float64) {
	x1 := wall.x1
	x2 := wall.x2
	top1 := wall.top1
	top2 := wall.top2
	bot1 := wall.bot1
	bot2 := wall.bot2
	z1 := wall.z1
	z2 := wall.z2

	if x1 > x2 {
		x1, x2 = x2, x1
		top1, top2 = top2, top1
		bot1, bot2 = bot2, bot1
		z1, z2 = z2, z1
	}

	if x2-x1 < 0.0001 {
		return
	}

	left := intMax(0, int(math.Ceil(x1)))
	right := intMin(internalRenderWidth-1, int(math.Floor(x2)))
	if right < left {
		return
	}

	seed := materialSeed(wall.material)
	baseR, baseG, baseB := materialColor(seed)

	for x := left; x <= right; x++ {
		t := (float64(x) - x1) / (x2 - x1)
		top := lerp(top1, top2, t)
		bot := lerp(bot1, bot2, t)
		if bot < top {
			top, bot = bot, top
		}

		if bot < 0 || top >= float64(r.viewportH) {
			continue
		}

		topInt := int(clamp(math.Round(top), 0, float64(r.viewportH-1)))
		botInt := int(clamp(math.Round(bot), 0, float64(r.viewportH-1)))
		if botInt < topInt {
			continue
		}

		depth := lerp(z1, z2, t)
		if depth >= depthBuffer[x] {
			continue
		}
		depthBuffer[x] = depth

		shadeDistance := clamp(1-(depth/1700), 0.12, 1)
		shadeLight := clamp(wall.light, 0.25, 1.25)

		for y := topInt; y <= botInt; y++ {
			rowTone := 0.88 + 0.12*math.Sin(float64(seed)+float64(x)*0.23+float64(y)*0.06)
			brightness := shadeDistance * shadeLight * rowTone
			col := color.RGBA{
				R: uint8(clamp(baseR*brightness, 0, 255)),
				G: uint8(clamp(baseG*brightness, 0, 255)),
				B: uint8(clamp(baseB*brightness, 0, 255)),
				A: 255,
			}
			r.backbuffer.Set(x, y, col)
		}
	}
}

func (r *FirstPersonRenderer) drawWeapon(frame domain.Frame) {
	bob := math.Sin(float64(frame.Tick)*0.22) * 2
	baseY := r.viewportH - 10 + int(math.Round(bob))
	centerX := internalRenderWidth / 2

	metal := color.RGBA{R: 118, G: 112, B: 108, A: 255}
	darkMetal := color.RGBA{R: 82, G: 76, B: 72, A: 255}
	grip := color.RGBA{R: 60, G: 46, B: 38, A: 255}

	fillRect(r.backbuffer, centerX-26, baseY-10, 52, 9, darkMetal)
	fillRect(r.backbuffer, centerX-18, baseY-18, 36, 8, metal)
	fillRect(r.backbuffer, centerX-10, baseY-28, 20, 10, metal)
	fillRect(r.backbuffer, centerX-6, baseY-8, 12, 16, grip)
}

func (r *FirstPersonRenderer) drawStatusBar(frame domain.Frame) {
	base := color.RGBA{R: 74, G: 54, B: 40, A: 255}
	edge := color.RGBA{R: 110, G: 86, B: 68, A: 255}
	dark := color.RGBA{R: 36, G: 26, B: 18, A: 255}

	fillRect(r.backbuffer, 0, r.statusY, internalRenderWidth, r.statusH, base)
	fillRect(r.backbuffer, 0, r.statusY, internalRenderWidth, 2, edge)
	fillRect(r.backbuffer, 0, internalRenderHeight-2, internalRenderWidth, 2, dark)

	healthPanelX := 8
	ammoPanelX := internalRenderWidth - 8 - 96
	facePanelX := (internalRenderWidth / 2) - 26

	fillRect(r.backbuffer, healthPanelX, r.statusY+6, 96, 20, dark)
	fillRect(r.backbuffer, ammoPanelX, r.statusY+6, 96, 20, dark)
	fillRect(r.backbuffer, facePanelX, r.statusY+4, 52, 24, color.RGBA{R: 55, G: 40, B: 34, A: 255})

	health := 100
	ammo := 50
	healthBars := health / 10
	ammoBars := ammo / 5
	for i := 0; i < healthBars; i++ {
		fillRect(r.backbuffer, healthPanelX+4+i*9, r.statusY+11, 6, 10, color.RGBA{R: 180, G: 56, B: 46, A: 255})
	}
	for i := 0; i < ammoBars && i < 19; i++ {
		fillRect(r.backbuffer, ammoPanelX+4+i*5, r.statusY+11, 3, 10, color.RGBA{R: 220, G: 188, B: 82, A: 255})
	}

	faceCol := color.RGBA{R: 194, G: 146, B: 112, A: 255}
	fillRect(r.backbuffer, facePanelX+14, r.statusY+8, 24, 16, faceCol)
	fillRect(r.backbuffer, facePanelX+19, r.statusY+13, 3, 3, dark)
	fillRect(r.backbuffer, facePanelX+30, r.statusY+13, 3, 3, dark)
	fillRect(r.backbuffer, facePanelX+22, r.statusY+19, 8, 2, dark)
}

func collectRenderableWalls(level domain.Level) []worldWall {
	walls := make([]worldWall, 0, len(level.Linedefs)*2)

	for _, line := range level.Linedefs {
		aIdx := int(line.StartVertex)
		bIdx := int(line.EndVertex)
		if aIdx < 0 || bIdx < 0 || aIdx >= len(level.Vertexes) || bIdx >= len(level.Vertexes) {
			continue
		}

		av := level.Vertexes[aIdx]
		bv := level.Vertexes[bIdx]

		rightSide, hasRight := sidedefAt(level, line.RightSide)
		leftSide, hasLeft := sidedefAt(level, line.LeftSide)
		rightSector, rightOK := sectorBySidedef(level, rightSide, hasRight)
		leftSector, leftOK := sectorBySidedef(level, leftSide, hasLeft)

		switch {
		case hasRight && hasLeft && rightOK && leftOK:
			upperTop := math.Max(float64(rightSector.CeilingHeight), float64(leftSector.CeilingHeight))
			upperBottom := math.Min(float64(rightSector.CeilingHeight), float64(leftSector.CeilingHeight))
			if upperTop-upperBottom > 0.01 {
				walls = append(walls, buildWallSlice(
					av,
					bv,
					upperBottom,
					upperTop,
					(rightLight(rightSector)+rightLight(leftSector))/2,
					chooseTexture(rightSide.UpperTexture, leftSide.UpperTexture),
				))
			}

			lowerTop := math.Max(float64(rightSector.FloorHeight), float64(leftSector.FloorHeight))
			lowerBottom := math.Min(float64(rightSector.FloorHeight), float64(leftSector.FloorHeight))
			if lowerTop-lowerBottom > 0.01 {
				walls = append(walls, buildWallSlice(
					av,
					bv,
					lowerBottom,
					lowerTop,
					(rightLight(rightSector)+rightLight(leftSector))/2,
					chooseTexture(rightSide.LowerTexture, leftSide.LowerTexture),
				))
			}
		case hasRight && rightOK:
			walls = append(walls, buildWallSlice(
				av,
				bv,
				float64(rightSector.FloorHeight),
				float64(rightSector.CeilingHeight),
				rightLight(rightSector),
				rightSide.MiddleTexture,
			))
		case hasLeft && leftOK:
			walls = append(walls, buildWallSlice(
				av,
				bv,
				float64(leftSector.FloorHeight),
				float64(leftSector.CeilingHeight),
				rightLight(leftSector),
				leftSide.MiddleTexture,
			))
		}
	}

	return walls
}

func buildWallSlice(a domain.Vertex, b domain.Vertex, bottom float64, top float64, light float64, material string) worldWall {
	return worldWall{
		ax: float64(a.X),
		ay: float64(a.Y),
		bx: float64(b.X),
		by: float64(b.Y),

		bottomZ: bottom,
		topZ:    top,

		lightLevel: light,
		material:   normalizeMaterialName(material),
	}
}

func sidedefAt(level domain.Level, idx uint16) (domain.Sidedef, bool) {
	const noSide = math.MaxUint16
	if idx == noSide {
		return domain.Sidedef{}, false
	}
	i := int(idx)
	if i < 0 || i >= len(level.Sidedefs) {
		return domain.Sidedef{}, false
	}
	return level.Sidedefs[i], true
}

func sectorBySidedef(level domain.Level, side domain.Sidedef, ok bool) (domain.Sector, bool) {
	if !ok {
		return domain.Sector{}, false
	}
	i := int(side.Sector)
	if i < 0 || i >= len(level.Sectors) {
		return domain.Sector{}, false
	}
	return level.Sectors[i], true
}

func chooseTexture(primary string, fallback string) string {
	name := normalizeMaterialName(primary)
	if name != "" && name != "-" {
		return name
	}
	name = normalizeMaterialName(fallback)
	if name != "" {
		return name
	}
	return "WALL"
}

func normalizeMaterialName(name string) string {
	return strings.ToUpper(strings.TrimSpace(strings.TrimRight(name, "\x00")))
}

func rightLight(sector domain.Sector) float64 {
	return 0.3 + (float64(sector.LightLevel)/255.0)*0.9
}

func materialSeed(material string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(material))
	return h.Sum32()
}

func materialColor(seed uint32) (r float64, g float64, b float64) {
	palette := [][3]float64{
		{184, 138, 108},
		{142, 118, 96},
		{122, 96, 78},
		{170, 156, 138},
		{128, 112, 108},
	}
	choice := palette[int(seed)%len(palette)]
	return choice[0], choice[1], choice[2]
}

func fillRect(img *ebiten.Image, x int, y int, w int, h int, col color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	x0 := intMax(0, x)
	y0 := intMax(0, y)
	x1 := intMin(internalRenderWidth, x+w)
	y1 := intMin(internalRenderHeight, y+h)
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			img.Set(xx, yy, col)
		}
	}
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
