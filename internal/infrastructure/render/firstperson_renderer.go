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

	u1 float64
	u2 float64

	material string
	light    float64
	depth    float64
}

type projectedEnemy struct {
	left   int
	right  int
	top    int
	bottom int
	width  int
	height int

	depth float64

	kind      string
	hurtTicks int
	health    int
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
		width:      width,
		height:     height,
		focal:      focal,
		viewportH:  viewportH,
		statusY:    viewportH,
		statusH:    statusBarHeight,
		walls:      collectRenderableWalls(level),
		backbuffer: nil,
	}
}

func (r *FirstPersonRenderer) Draw(screen *ebiten.Image, frame domain.Frame) {
	r.ensureBackbuffer()
	r.drawBackground(frame)

	projectedWalls := r.projectWalls(frame)
	slices.SortFunc(projectedWalls, func(a projectedWall, b projectedWall) int {
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

	for _, wall := range projectedWalls {
		r.drawProjectedWall(wall, depthBuffer)
	}

	enemies := r.projectEnemies(frame)
	slices.SortFunc(enemies, func(a projectedEnemy, b projectedEnemy) int {
		if a.depth > b.depth {
			return -1
		}
		if a.depth < b.depth {
			return 1
		}
		return 0
	})
	for _, enemy := range enemies {
		r.drawProjectedEnemy(enemy, depthBuffer)
	}

	r.drawCrosshair()
	r.drawWeapon(frame)
	r.drawStatusBar(frame)
	if frame.DamageFlashTicks > 0 {
		intensity := clamp(float64(frame.DamageFlashTicks)/6, 0.08, 0.45)
		overlayTint(r.backbuffer, 0, 0, internalRenderWidth, r.viewportH, color.RGBA{R: 164, G: 24, B: 24, A: 255}, intensity)
	}

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

func (r *FirstPersonRenderer) drawBackground(frame domain.Frame) {
	centerX := float64(internalRenderWidth) / 2
	centerY := float64(r.viewportH) / 2
	cosA := math.Cos(frame.Angle)
	sinA := math.Sin(frame.Angle)
	playerX := float64(frame.PlayerX)
	playerY := float64(frame.PlayerY)

	for y := 0; y < r.viewportH; y++ {
		row := float64(y) - centerY
		for x := 0; x < internalRenderWidth; x++ {
			if math.Abs(row) < 0.75 {
				r.backbuffer.Set(x, y, color.RGBA{R: 38, G: 44, B: 58, A: 255})
				continue
			}

			dist := (playerEyeHeight * r.focal) / math.Abs(row)
			dist = clamp(dist, 8, 2800)
			cameraX := (float64(x) - centerX) / r.focal

			worldX := playerX + dist*(cosA-sinA*cameraX)
			worldY := playerY + dist*(sinA+cosA*cameraX)

			if row > 0 {
				r.backbuffer.Set(x, y, sampleFloorTexel(worldX, worldY, dist))
			} else {
				r.backbuffer.Set(x, y, sampleCeilingTexel(worldX, worldY, dist))
			}
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
		worldAX, worldAY := wall.ax, wall.ay
		worldBX, worldBY := wall.bx, wall.by

		ax, az := toCamera(worldAX, worldAY, float64(frame.PlayerX), float64(frame.PlayerY), cosA, sinA)
		bx, bz := toCamera(worldBX, worldBY, float64(frame.PlayerX), float64(frame.PlayerY), cosA, sinA)

		if az <= nearPlaneDistance && bz <= nearPlaneDistance {
			continue
		}

		if az <= nearPlaneDistance {
			t := (nearPlaneDistance - az) / (bz - az)
			worldAX = worldAX + (worldBX-worldAX)*t
			worldAY = worldAY + (worldBY-worldAY)*t
			ax = ax + (bx-ax)*t
			az = nearPlaneDistance
		}
		if bz <= nearPlaneDistance {
			t := (nearPlaneDistance - bz) / (az - bz)
			worldBX = worldBX + (worldAX-worldBX)*t
			worldBY = worldBY + (worldAY-worldBY)*t
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

		wallLen := math.Hypot(wall.bx-wall.ax, wall.by-wall.ay)
		if wallLen < 0.0001 {
			continue
		}
		u1 := math.Hypot(worldAX-wall.ax, worldAY-wall.ay) / 64
		u2 := math.Hypot(worldBX-wall.ax, worldBY-wall.ay) / 64

		out = append(out, projectedWall{
			x1: sx1,
			x2: sx2,

			top1: top1,
			top2: top2,
			bot1: bot1,
			bot2: bot2,

			z1: az,
			z2: bz,

			u1: u1,
			u2: u2,

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
	u1 := wall.u1
	u2 := wall.u2

	if x1 > x2 {
		x1, x2 = x2, x1
		top1, top2 = top2, top1
		bot1, bot2 = bot2, bot1
		z1, z2 = z2, z1
		u1, u2 = u2, u1
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

		u := lerp(u1, u2, t)
		heightInv := 1.0 / float64(intMax(1, botInt-topInt+1))
		brightness := clamp((1-(depth/1750))*wall.light, 0.15, 1.15)

		for y := topInt; y <= botInt; y++ {
			v := float64(y-topInt) * heightInv
			col := sampleWallTexel(wall.material, seed, baseR, baseG, baseB, u, v, brightness, x, y)
			r.backbuffer.Set(x, y, col)
		}

		depthBuffer[x] = depth
	}
}

func (r *FirstPersonRenderer) projectEnemies(frame domain.Frame) []projectedEnemy {
	out := make([]projectedEnemy, 0, len(frame.Enemies))
	cosA := math.Cos(frame.Angle)
	sinA := math.Sin(frame.Angle)
	centerX := float64(internalRenderWidth) / 2
	centerY := float64(r.viewportH) / 2

	for _, enemy := range frame.Enemies {
		if !enemy.Alive {
			continue
		}

		x, z := toCamera(
			float64(enemy.X),
			float64(enemy.Y),
			float64(frame.PlayerX),
			float64(frame.PlayerY),
			cosA,
			sinA,
		)
		if z <= nearPlaneDistance || z > 1900 {
			continue
		}

		screenX := centerX + (x/z)*r.focal
		scale := r.focal / z
		spriteH := int(math.Round(scale * 82))
		spriteW := int(math.Round(scale * 48))
		if spriteH < 6 || spriteW < 4 {
			continue
		}

		bottomY := int(math.Round(centerY + (playerEyeHeight/z)*r.focal))
		topY := bottomY - spriteH
		leftX := int(math.Round(screenX)) - (spriteW / 2)
		rightX := leftX + spriteW - 1

		if rightX < 0 || leftX >= internalRenderWidth || topY >= r.viewportH || bottomY < 0 {
			continue
		}

		out = append(out, projectedEnemy{
			left:      leftX,
			right:     rightX,
			top:       topY,
			bottom:    bottomY,
			width:     spriteW,
			height:    spriteH,
			depth:     z,
			kind:      enemy.Kind,
			health:    enemy.Health,
			hurtTicks: enemy.HurtTicks,
		})
	}

	return out
}

func (r *FirstPersonRenderer) drawProjectedEnemy(enemy projectedEnemy, depthBuffer []float64) {
	left := intMax(0, enemy.left)
	right := intMin(internalRenderWidth-1, enemy.right)
	top := intMax(0, enemy.top)
	bottom := intMin(r.viewportH-1, enemy.bottom)
	if right < left || bottom < top {
		return
	}

	widthDenom := float64(intMax(1, enemy.width-1))
	heightDenom := float64(intMax(1, enemy.height-1))

	for x := left; x <= right; x++ {
		if enemy.depth >= depthBuffer[x] {
			continue
		}

		u := float64(x-enemy.left) / widthDenom
		columnPainted := false
		for y := top; y <= bottom; y++ {
			v := float64(y-enemy.top) / heightDenom
			col, ok := sampleEnemyTexel(enemy.kind, enemy.health, enemy.hurtTicks, u, v)
			if !ok {
				continue
			}
			r.backbuffer.Set(x, y, col)
			columnPainted = true
		}

		if columnPainted {
			depthBuffer[x] = enemy.depth
		}
	}
}

func (r *FirstPersonRenderer) drawCrosshair() {
	cx := internalRenderWidth / 2
	cy := r.viewportH / 2
	col := color.RGBA{R: 222, G: 188, B: 86, A: 255}

	fillRect(r.backbuffer, cx-6, cy, 13, 1, col)
	fillRect(r.backbuffer, cx, cy-6, 1, 13, col)
	fillRect(r.backbuffer, cx-1, cy-1, 3, 3, color.RGBA{R: 255, G: 238, B: 164, A: 255})
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

	if frame.WeaponFlashTicks > 0 {
		flashStrength := clamp(float64(frame.WeaponFlashTicks)/3, 0.2, 1)
		overlayTint(
			r.backbuffer,
			centerX-24,
			baseY-36,
			48,
			20,
			color.RGBA{R: 255, G: 214, B: 130, A: 255},
			0.5*flashStrength,
		)
	}
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

	health := clamp(float64(frame.Health), 0, 100)
	ammo := clamp(float64(frame.Ammo), 0, 99)
	healthBars := int(math.Round(health / 10))
	ammoBars := int(math.Round(ammo / 5))
	for i := 0; i < healthBars && i < 10; i++ {
		fillRect(r.backbuffer, healthPanelX+4+i*9, r.statusY+11, 6, 10, color.RGBA{R: 180, G: 56, B: 46, A: 255})
	}
	for i := 0; i < ammoBars && i < 19; i++ {
		fillRect(r.backbuffer, ammoPanelX+4+i*5, r.statusY+11, 3, 10, color.RGBA{R: 220, G: 188, B: 82, A: 255})
	}

	faceCol := color.RGBA{R: 194, G: 146, B: 112, A: 255}
	if frame.Health <= 0 {
		faceCol = color.RGBA{R: 84, G: 74, B: 74, A: 255}
	} else if frame.DamageFlashTicks > 0 {
		faceCol = color.RGBA{R: 212, G: 116, B: 96, A: 255}
	}
	fillRect(r.backbuffer, facePanelX+14, r.statusY+8, 24, 16, faceCol)
	fillRect(r.backbuffer, facePanelX+19, r.statusY+13, 3, 3, dark)
	fillRect(r.backbuffer, facePanelX+30, r.statusY+13, 3, 3, dark)
	fillRect(r.backbuffer, facePanelX+22, r.statusY+19, 8, 2, dark)

	killSlots := intMin(10, frame.Kills)
	for i := 0; i < killSlots; i++ {
		fillRect(r.backbuffer, facePanelX-38+i*3, r.statusY+11, 2, 10, color.RGBA{R: 168, G: 32, B: 28, A: 255})
	}
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
		{120, 140, 156},
	}
	choice := palette[int(seed)%len(palette)]
	return choice[0], choice[1], choice[2]
}

func sampleWallTexel(material string, seed uint32, baseR float64, baseG float64, baseB float64, u float64, v float64, brightness float64, x int, y int) color.RGBA {
	ux := int(math.Floor(fract(u) * 64))
	vy := int(math.Floor(fract(v) * 128))
	pattern := 1.0

	switch wallMaterialClass(material) {
	case "brick":
		if vy%12 == 0 {
			pattern *= 0.62
		}
		offset := 0
		if ((vy / 12) % 2) == 1 {
			offset = 8
		}
		if (ux+offset)%16 == 0 {
			pattern *= 0.58
		}
	case "metal":
		if ux%16 == 0 || vy%16 == 0 {
			pattern *= 0.72
		}
		if (ux%16 == 3 || ux%16 == 12) && (vy%16 == 3 || vy%16 == 12) {
			pattern *= 1.22
		}
	case "wood":
		wave := math.Sin(float64(ux)*0.35 + float64(vy)*0.07)
		pattern *= 0.82 + 0.2*wave
	default:
		if ux%32 == 0 || vy%20 == 0 {
			pattern *= 0.8
		}
	}

	noise := 0.86 + 0.22*valueNoise2D(ux+x*3, vy+y*2, seed)
	shading := pattern * noise * brightness
	return color.RGBA{
		R: uint8(clamp(baseR*shading, 0, 255)),
		G: uint8(clamp(baseG*shading, 0, 255)),
		B: uint8(clamp(baseB*shading, 0, 255)),
		A: 255,
	}
}

func wallMaterialClass(material string) string {
	upper := strings.ToUpper(material)
	switch {
	case strings.Contains(upper, "BRI"), strings.Contains(upper, "STONE"), strings.Contains(upper, "BST"), strings.Contains(upper, "ROCK"):
		return "brick"
	case strings.Contains(upper, "MET"), strings.Contains(upper, "TEK"), strings.Contains(upper, "COMP"), strings.Contains(upper, "SUP"), strings.Contains(upper, "PAN"):
		return "metal"
	case strings.Contains(upper, "WOOD"), strings.Contains(upper, "WOD"), strings.Contains(upper, "LOG"):
		return "wood"
	default:
		return "concrete"
	}
}

func sampleFloorTexel(worldX float64, worldY float64, dist float64) color.RGBA {
	tile := 64.0
	tx := int(math.Floor(worldX / tile))
	ty := int(math.Floor(worldY / tile))
	localX := fract(worldX / tile)
	localY := fract(worldY / tile)

	noise := valueNoise2D(tx*17+int(localX*23), ty*19+int(localY*31), 91231)
	base := color.RGBA{R: 62, G: 46, B: 36, A: 255}
	if (tx+ty)&1 == 0 {
		base = color.RGBA{R: 70, G: 54, B: 42, A: 255}
	}
	if localX < 0.03 || localY < 0.03 {
		base = color.RGBA{R: 38, G: 26, B: 20, A: 255}
	}

	fog := clamp(1-(dist/2100), 0.2, 1)
	mod := (0.88 + 0.2*noise) * fog
	return color.RGBA{
		R: uint8(clamp(float64(base.R)*mod, 0, 255)),
		G: uint8(clamp(float64(base.G)*mod, 0, 255)),
		B: uint8(clamp(float64(base.B)*mod, 0, 255)),
		A: 255,
	}
}

func sampleCeilingTexel(worldX float64, worldY float64, dist float64) color.RGBA {
	tile := 96.0
	tx := int(math.Floor(worldX / tile))
	ty := int(math.Floor(worldY / tile))
	localX := fract(worldX / tile)
	localY := fract(worldY / tile)

	base := color.RGBA{R: 34, G: 36, B: 52, A: 255}
	if (tx+ty)&1 == 0 {
		base = color.RGBA{R: 40, G: 42, B: 60, A: 255}
	}
	if localX < 0.02 || localY < 0.02 {
		base = color.RGBA{R: 20, G: 22, B: 34, A: 255}
	}
	noise := valueNoise2D(tx*13+int(localX*27), ty*11+int(localY*21), 42193)
	fog := clamp(1-(dist/2300), 0.2, 1)
	mod := (0.85 + 0.24*noise) * fog
	return color.RGBA{
		R: uint8(clamp(float64(base.R)*mod, 0, 255)),
		G: uint8(clamp(float64(base.G)*mod, 0, 255)),
		B: uint8(clamp(float64(base.B)*mod, 0, 255)),
		A: 255,
	}
}

func sampleEnemyTexel(kind string, health int, hurtTicks int, u float64, v float64) (color.RGBA, bool) {
	cx := (u - 0.5) * 2
	cy := (v - 0.52) * 2

	body := (cx*cx)/(0.85*0.85)+((cy-0.24)*(cy-0.24))/(0.78*0.78) <= 1
	head := (cx*cx)/(0.42*0.42)+((cy+0.48)*(cy+0.48))/(0.32*0.32) <= 1
	leftLeg := ((cx+0.26)*(cx+0.26))/(0.22*0.22)+((cy-0.9)*(cy-0.9))/(0.22*0.22) <= 1
	rightLeg := ((cx-0.26)*(cx-0.26))/(0.22*0.22)+((cy-0.9)*(cy-0.9))/(0.22*0.22) <= 1
	if !body && !head && !leftLeg && !rightLeg {
		return color.RGBA{}, false
	}

	base := enemyBaseColor(kind)
	shade := 0.74 + 0.26*(1-v)
	if hurtTicks > 0 {
		base = color.RGBA{R: 224, G: 58, B: 58, A: 255}
		shade = 0.9 + 0.1*(1-v)
	} else if health < 40 {
		base = color.RGBA{R: clampU8(int(base.R) + 20), G: clampU8(int(base.G) - 15), B: clampU8(int(base.B) - 15), A: 255}
	}

	r := clamp(float64(base.R)*shade, 0, 255)
	g := clamp(float64(base.G)*shade, 0, 255)
	b := clamp(float64(base.B)*shade, 0, 255)
	col := color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}

	// Eyes
	if math.Abs(cy+0.55) < 0.06 && (math.Abs(cx-0.12) < 0.05 || math.Abs(cx+0.12) < 0.05) {
		return color.RGBA{R: 255, G: 226, B: 124, A: 255}, true
	}

	return col, true
}

func enemyBaseColor(kind string) color.RGBA {
	switch strings.ToUpper(kind) {
	case "ZOMBIE":
		return color.RGBA{R: 116, G: 128, B: 108, A: 255}
	case "IMP":
		return color.RGBA{R: 148, G: 96, B: 76, A: 255}
	case "DEMON":
		return color.RGBA{R: 138, G: 78, B: 74, A: 255}
	case "LOSTSOUL":
		return color.RGBA{R: 180, G: 166, B: 132, A: 255}
	default:
		return color.RGBA{R: 124, G: 92, B: 88, A: 255}
	}
}

func valueNoise2D(x int, y int, seed uint32) float64 {
	n := uint32(x*374761393+y*668265263) ^ seed
	n = (n ^ (n >> 13)) * 1274126177
	n = n ^ (n >> 16)
	return float64(n&0xFF) / 255.0
}

func fract(v float64) float64 {
	return v - math.Floor(v)
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

func overlayTint(img *ebiten.Image, x int, y int, w int, h int, tint color.RGBA, amount float64) {
	amount = clamp(amount, 0, 1)
	if amount <= 0 {
		return
	}
	x0 := intMax(0, x)
	y0 := intMax(0, y)
	x1 := intMin(internalRenderWidth, x+w)
	y1 := intMin(internalRenderHeight, y+h)
	for yy := y0; yy < y1; yy++ {
		for xx := x0; xx < x1; xx++ {
			rv, gv, bv, _ := img.At(xx, yy).RGBA()
			currentR := float64(rv >> 8)
			currentG := float64(gv >> 8)
			currentB := float64(bv >> 8)
			img.Set(xx, yy, color.RGBA{
				R: uint8(clamp(currentR*(1-amount)+float64(tint.R)*amount, 0, 255)),
				G: uint8(clamp(currentG*(1-amount)+float64(tint.G)*amount, 0, 255)),
				B: uint8(clamp(currentB*(1-amount)+float64(tint.B)*amount, 0, 255)),
				A: 255,
			})
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

func clampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
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
