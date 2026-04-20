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
	facing   float64
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
	phase     float64
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
		r.drawEnemyShadow(enemy, depthBuffer)
		r.drawProjectedEnemy(enemy, depthBuffer)
	}

	r.drawAtmospherePass(frame)
	r.drawCrosshair()
	r.drawWeapon(frame)
	r.drawStatusBar(frame)
	if frame.DamageFlashTicks > 0 {
		intensity := clamp(float64(frame.DamageFlashTicks)/6, 0.08, 0.45)
		overlayTint(r.backbuffer, 0, 0, internalRenderWidth, r.viewportH, color.RGBA{R: 164, G: 24, B: 24, A: 255}, intensity)
	}
	r.applyVignetteAndGrain(frame.Tick)

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
	horizonY := centerY - 5 + math.Sin(float64(frame.Tick)*0.015)*1.4

	for y := 0; y < r.viewportH; y++ {
		row := float64(y) - horizonY
		for x := 0; x < internalRenderWidth; x++ {
			if row < -0.5 {
				r.backbuffer.Set(x, y, sampleSkyTexel(float64(x), float64(y), frame.Angle, frame.Tick))
				continue
			}
			if row < 0.6 {
				r.backbuffer.Set(x, y, color.RGBA{R: 60, G: 64, B: 74, A: 255})
				continue
			}

			dist := (playerEyeHeight * r.focal) / row
			dist = clamp(dist, 8, 3200)
			cameraX := (float64(x) - centerX) / r.focal

			worldX := playerX + dist*(cosA-sinA*cameraX)
			worldY := playerY + dist*(sinA+cosA*cameraX)

			floorCol := sampleFloorTexel(worldX, worldY, dist, frame.Tick)
			fogAmt := smoothstep(780, 2600, dist)
			r.backbuffer.Set(x, y, mixColor(floorCol, color.RGBA{R: 82, G: 90, B: 106, A: 255}, 0.58*fogAmt))
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
		normalX := -(wall.by - wall.ay) / wallLen
		normalY := (wall.bx - wall.ax) / wallLen
		facing := math.Abs(normalX*cosA + normalY*sinA)

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
			facing:   facing,
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
		depthLighting := clamp(1-(depth/1850), 0.12, 1.08)
		sideLighting := 0.72 + wall.facing*0.36
		brightness := clamp(depthLighting*wall.light*sideLighting, 0.12, 1.25)
		fogAmt := smoothstep(760, 2450, depth)

		for y := topInt; y <= botInt; y++ {
			v := float64(y-topInt) * heightInv
			col := sampleWallTexel(wall.material, seed, baseR, baseG, baseB, u, v, brightness, x, y)
			col = mixColor(col, color.RGBA{R: 78, G: 84, B: 96, A: 255}, 0.62*fogAmt)
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
			phase:     float64(frame.Tick)*0.18 + (float64(enemy.X)+float64(enemy.Y))*0.008,
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
			col, alpha, ok := sampleEnemyTexel(enemy.kind, enemy.health, enemy.hurtTicks, u, v, enemy.phase, enemy.depth)
			if !ok {
				continue
			}
			blendPixel(r.backbuffer, x, y, col, alpha)
			columnPainted = true
		}

		if columnPainted {
			depthBuffer[x] = enemy.depth
		}
	}
}

func (r *FirstPersonRenderer) drawEnemyShadow(enemy projectedEnemy, depthBuffer []float64) {
	shadowW := int(float64(enemy.width) * 0.64)
	shadowH := int(float64(enemy.height) * 0.12)
	if shadowW < 4 || shadowH < 2 {
		return
	}

	cx := (enemy.left + enemy.right) / 2
	baseY := intMin(r.viewportH-1, enemy.bottom+2)
	left := intMax(0, cx-shadowW/2)
	right := intMin(internalRenderWidth-1, cx+shadowW/2)
	top := intMax(0, baseY-shadowH/2)
	bottom := intMin(r.viewportH-1, baseY+shadowH/2)
	if right < left || bottom < top {
		return
	}

	for x := left; x <= right; x++ {
		if enemy.depth > depthBuffer[x]+120 {
			continue
		}
		nx := (float64(x-cx) / float64(intMax(1, shadowW/2)))
		for y := top; y <= bottom; y++ {
			ny := (float64(y-baseY) / float64(intMax(1, shadowH/2)))
			d := nx*nx + ny*ny
			if d > 1 {
				continue
			}
			alpha := (1 - d) * 0.32
			blendPixel(r.backbuffer, x, y, color.RGBA{R: 10, G: 8, B: 8, A: 255}, alpha)
		}
	}
}

func (r *FirstPersonRenderer) drawAtmospherePass(frame domain.Frame) {
	t := 0.42 + 0.1*math.Sin(float64(frame.Tick)*0.02)
	overlayTint(r.backbuffer, 0, 0, internalRenderWidth, r.viewportH/2, color.RGBA{R: 36, G: 54, B: 84, A: 255}, 0.08*t)
	overlayTint(r.backbuffer, 0, r.viewportH/2, internalRenderWidth, r.viewportH/2, color.RGBA{R: 84, G: 68, B: 52, A: 255}, 0.05)
}

func (r *FirstPersonRenderer) applyVignetteAndGrain(tick uint64) {
	cx := float64(internalRenderWidth) / 2
	cy := float64(r.viewportH) / 2
	maxDist := math.Hypot(cx, cy)

	for y := 0; y < r.viewportH; y++ {
		for x := 0; x < internalRenderWidth; x++ {
			dx := float64(x) - cx
			dy := float64(y) - cy
			dist := math.Hypot(dx, dy) / maxDist
			vignette := clamp((dist-0.58)/0.42, 0, 1)

			rv, gv, bv, _ := r.backbuffer.At(x, y).RGBA()
			red := float64(rv >> 8)
			green := float64(gv >> 8)
			blue := float64(bv >> 8)

			grain := (valueNoise2D(x+int(tick)%512, y+int(tick*3)%512, uint32(913+tick*17)) - 0.5) * 10
			darkness := 1 - vignette*0.4

			red = clamp((red+grain)*darkness, 0, 255)
			green = clamp((green+grain)*darkness, 0, 255)
			blue = clamp((blue+grain)*darkness, 0, 255)

			r.backbuffer.Set(x, y, color.RGBA{R: uint8(red), G: uint8(green), B: uint8(blue), A: 255})
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
	bobX := math.Sin(float64(frame.Tick)*0.18) * 2
	bobY := math.Sin(float64(frame.Tick)*0.24) * 2
	recoil := 0.0
	if frame.WeaponFlashTicks > 0 {
		recoil = float64(4 - frame.WeaponFlashTicks)
	}
	centerX := (internalRenderWidth / 2) + int(math.Round(bobX))
	baseY := r.viewportH - 8 + int(math.Round(bobY+recoil))

	// Hands
	drawEllipseFilled(r.backbuffer, centerX-28, baseY-4, 16, 10, color.RGBA{R: 148, G: 108, B: 82, A: 255})
	drawEllipseFilled(r.backbuffer, centerX+28, baseY-4, 16, 10, color.RGBA{R: 146, G: 106, B: 80, A: 255})
	drawEllipseFilled(r.backbuffer, centerX-28, baseY-1, 13, 7, color.RGBA{R: 122, G: 86, B: 64, A: 255})
	drawEllipseFilled(r.backbuffer, centerX+28, baseY-1, 13, 7, color.RGBA{R: 122, G: 86, B: 64, A: 255})

	// Receiver
	fillRectGradient(r.backbuffer, centerX-36, baseY-18, 72, 16, color.RGBA{R: 78, G: 74, B: 78, A: 255}, color.RGBA{R: 46, G: 44, B: 50, A: 255})
	fillRect(r.backbuffer, centerX-26, baseY-12, 52, 4, color.RGBA{R: 98, G: 98, B: 104, A: 255})

	// Barrel and pump
	fillRectGradient(r.backbuffer, centerX-16, baseY-34, 32, 18, color.RGBA{R: 116, G: 120, B: 126, A: 255}, color.RGBA{R: 70, G: 74, B: 82, A: 255})
	fillRectGradient(r.backbuffer, centerX-10, baseY-46, 20, 12, color.RGBA{R: 136, G: 138, B: 144, A: 255}, color.RGBA{R: 78, G: 82, B: 90, A: 255})
	fillRect(r.backbuffer, centerX-8, baseY-41, 16, 1, color.RGBA{R: 188, G: 190, B: 198, A: 255})

	// Grip
	fillRectGradient(r.backbuffer, centerX-9, baseY-8, 18, 20, color.RGBA{R: 72, G: 54, B: 40, A: 255}, color.RGBA{R: 44, G: 30, B: 22, A: 255})

	if frame.WeaponFlashTicks > 0 {
		flashStrength := clamp(float64(frame.WeaponFlashTicks)/3, 0.2, 1)
		drawEllipseFilled(
			r.backbuffer,
			centerX,
			baseY-48,
			int(18+flashStrength*10),
			int(7+flashStrength*4),
			color.RGBA{R: 255, G: 218, B: 132, A: 255},
		)
		overlayTint(
			r.backbuffer,
			centerX-36,
			baseY-62,
			72,
			42,
			color.RGBA{R: 255, G: 214, B: 120, A: 255},
			0.38*flashStrength,
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
	uFrac := fract(u)
	vFrac := fract(v)
	ux := uFrac * 64
	vy := vFrac * 128
	materialClass := wallMaterialClass(material)

	height := wallHeightSample(materialClass, ux, vy, seed)
	hx := wallHeightSample(materialClass, ux+1, vy, seed) - height
	hy := wallHeightSample(materialClass, ux, vy+1, seed) - height
	normalX, normalY, normalZ := normalizeVec3(-hx*1.8, -hy*1.8, 1)
	lightX, lightY, lightZ := normalizeVec3(-0.32, -0.18, 0.93)
	diffuse := clamp(normalX*lightX+normalY*lightY+normalZ*lightZ, 0.24, 1.25)

	pattern := 1.0
	switch materialClass {
	case "brick":
		mortarH := smoothPulse(vFrac, 0.09, 0.106, 0.16, 0.18)
		offset := 0.0
		if int(vy/12)%2 == 1 {
			offset = 8
		}
		mortarV := smoothPulse(fract((ux+offset)/16), 0.01, 0.03, 0.97, 0.99)
		pattern *= 0.75 + 0.25*(1-math.Max(mortarH, mortarV))
	case "metal":
		panelH := smoothPulse(fract(ux/16), 0.0, 0.05, 0.95, 1.0)
		panelV := smoothPulse(fract(vy/16), 0.0, 0.05, 0.95, 1.0)
		pattern *= 0.78 + 0.22*(1-math.Max(panelH, panelV))
	case "wood":
		grain := math.Sin(ux*0.22+vy*0.035) + 0.5*math.Sin(ux*0.57+vy*0.02)
		pattern *= 0.85 + 0.15*grain
	default:
		crack := smoothPulse(fract(ux/31), 0.0, 0.03, 0.97, 1.0)
		pattern *= 0.8 + 0.2*(1-crack)
	}

	microNoise := 0.84 + 0.28*valueNoise2D(int(ux)+x*5, int(vy)+y*3, seed)
	edgeOcclusion := clamp(math.Min(math.Min(uFrac, 1-uFrac), math.Min(vFrac, 1-vFrac))*7.5, 0.62, 1)
	spec := math.Pow(clamp(diffuse, 0, 1), 14) * 0.12
	shading := pattern * microNoise * brightness * diffuse * edgeOcclusion

	return color.RGBA{
		R: uint8(clamp(baseR*(shading+spec), 0, 255)),
		G: uint8(clamp(baseG*(shading+spec*0.95), 0, 255)),
		B: uint8(clamp(baseB*(shading+spec*0.9), 0, 255)),
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

func sampleFloorTexel(worldX float64, worldY float64, dist float64, tick uint64) color.RGBA {
	tile := 64.0
	tx := int(math.Floor(worldX / tile))
	ty := int(math.Floor(worldY / tile))
	localX := fract(worldX / tile)
	localY := fract(worldY / tile)

	noise := valueNoise2D(tx*19+int(localX*29), ty*23+int(localY*31), 91231)
	grime := valueNoise2D(tx*11+int(localX*61), ty*13+int(localY*67), 5123+uint32(tick))
	base := color.RGBA{R: 68, G: 52, B: 40, A: 255}
	if (tx+ty)&1 == 0 {
		base = color.RGBA{R: 76, G: 58, B: 44, A: 255}
	}
	grout := math.Max(
		smoothPulse(localX, 0.0, 0.028, 0.972, 1.0),
		smoothPulse(localY, 0.0, 0.028, 0.972, 1.0),
	)
	if grout > 0.2 {
		base = color.RGBA{R: 40, G: 30, B: 24, A: 255}
	}
	wetness := clamp((grime-0.75)*4, 0, 1)
	highlight := 0.06 + wetness*0.22
	fog := clamp(1-(dist/2350), 0.18, 1)
	mod := (0.82 + 0.24*noise) * fog
	return color.RGBA{
		R: uint8(clamp(float64(base.R)*(mod+highlight), 0, 255)),
		G: uint8(clamp(float64(base.G)*(mod+highlight*0.8), 0, 255)),
		B: uint8(clamp(float64(base.B)*(mod+highlight*0.55), 0, 255)),
		A: 255,
	}
}

func sampleEnemyTexel(kind string, health int, hurtTicks int, u float64, v float64, phase float64, depth float64) (color.RGBA, float64, bool) {
	cx := (u - 0.5) * 2
	cy := (v - 0.54) * 2

	sway := math.Sin(phase) * 0.06
	armSwing := math.Sin(phase+math.Pi/2) * 0.12
	legSwing := math.Sin(phase) * 0.16

	torso := ellipseSDF(cx-sway*0.3, cy+0.12, 0.62, 0.72)
	head := ellipseSDF(cx-sway*0.5, cy+0.74, 0.32, 0.28)
	leftArm := ellipseSDF(cx+0.52-armSwing*0.2, cy+0.16, 0.17, 0.43)
	rightArm := ellipseSDF(cx-0.52+armSwing*0.2, cy+0.16, 0.17, 0.43)
	leftLeg := ellipseSDF(cx+0.2+legSwing*0.2, cy-0.74, 0.19, 0.36)
	rightLeg := ellipseSDF(cx-0.2-legSwing*0.2, cy-0.74, 0.19, 0.36)

	model := math.Min(math.Min(torso, head), math.Min(math.Min(leftArm, rightArm), math.Min(leftLeg, rightLeg)))
	if strings.EqualFold(kind, "LOSTSOUL") {
		model = ellipseSDF(cx, cy+0.28, 0.54, 0.58)
	}
	if model > 0 {
		if strings.EqualFold(kind, "LOSTSOUL") {
			aura := ellipseSDF(cx, cy+0.25, 0.68, 0.72)
			if aura <= 0 {
				a := clamp(-aura*0.3, 0.04, 0.2)
				return color.RGBA{R: 255, G: 152, B: 76, A: 255}, a, true
			}
		}
		return color.RGBA{}, 0, false
	}

	base := enemyBaseColor(kind)
	if health < 40 {
		base = color.RGBA{
			R: clampU8(int(base.R) + 18),
			G: clampU8(int(base.G) - 12),
			B: clampU8(int(base.B) - 10),
			A: 255,
		}
	}
	if hurtTicks > 0 {
		base = color.RGBA{R: 220, G: 62, B: 60, A: 255}
	}

	nx := -cx * 0.72
	ny := (1 - v) * 0.65
	nz := 1.0
	nx, ny, nz = normalizeVec3(nx, ny, nz)
	lx, ly, lz := normalizeVec3(-0.34, -0.22, 0.92)
	lighting := clamp(nx*lx+ny*ly+nz*lz, 0.26, 1.22)

	ao := clamp(1+model*0.95, 0.42, 1)
	depthFade := clamp(1-(depth/2100), 0.48, 1)
	shade := lighting * ao * depthFade

	col := color.RGBA{
		R: uint8(clamp(float64(base.R)*shade, 0, 255)),
		G: uint8(clamp(float64(base.G)*shade, 0, 255)),
		B: uint8(clamp(float64(base.B)*shade, 0, 255)),
		A: 255,
	}

	// Eyes and mouth details.
	if head <= 0 {
		if math.Abs(cy+0.74) < 0.06 && (math.Abs(cx-0.11) < 0.045 || math.Abs(cx+0.11) < 0.045) {
			return color.RGBA{R: 252, G: 228, B: 126, A: 255}, 1, true
		}
		if math.Abs(cy+0.61) < 0.04 && math.Abs(cx) < 0.12 {
			return color.RGBA{R: 74, G: 24, B: 20, A: 255}, 0.95, true
		}
	}

	alpha := clamp(-model*3.2, 0.55, 1)
	return col, alpha, true
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

func sampleSkyTexel(screenX float64, screenY float64, angle float64, tick uint64) color.RGBA {
	horizon := float64(internalRenderHeight-statusBarHeight) * 0.5
	t := clamp(screenY/intMaxF(1, horizon), 0, 1)

	skyTop := color.RGBA{R: 20, G: 30, B: 54, A: 255}
	skyBottom := color.RGBA{R: 90, G: 110, B: 146, A: 255}
	col := mixColor(skyTop, skyBottom, t*0.9)

	uvx := fract(screenX/float64(internalRenderWidth) + angle/(2*math.Pi) + float64(tick)*0.00006)
	uvy := screenY / horizon
	cloudNoise := fbmNoise2D(uvx*190, uvy*145+float64(tick)*0.003, 7391)
	cloud := clamp((cloudNoise-0.56)*2.2, 0, 1)
	if cloud > 0 {
		col = mixColor(col, color.RGBA{R: 196, G: 202, B: 218, A: 255}, cloud*0.45)
	}

	starNoise := valueNoise2D(int(uvx*640), int(uvy*640), 12211+uint32(tick/8))
	if t < 0.25 && starNoise > 0.992 {
		col = mixColor(col, color.RGBA{R: 255, G: 244, B: 196, A: 255}, 0.8)
	}

	return col
}

func wallHeightSample(materialClass string, ux float64, vy float64, seed uint32) float64 {
	x := int(math.Floor(ux))
	y := int(math.Floor(vy))

	switch materialClass {
	case "brick":
		mortar := 0.0
		if y%12 == 0 {
			mortar += 0.45
		}
		offset := 0
		if (y/12)%2 == 1 {
			offset = 8
		}
		if (x+offset)%16 == 0 {
			mortar += 0.45
		}
		return clamp(0.6+0.35*valueNoise2D(x*3, y*2, seed)-mortar, 0, 1)
	case "metal":
		panel := 0.0
		if x%16 == 0 || y%16 == 0 {
			panel += 0.22
		}
		rivet := 0.0
		if (x%16 == 3 || x%16 == 12) && (y%16 == 3 || y%16 == 12) {
			rivet = 0.35
		}
		return clamp(0.5+0.3*valueNoise2D(x, y, seed)-panel+rivet, 0, 1)
	case "wood":
		grain := 0.5 + 0.5*math.Sin(ux*0.21+vy*0.03+math.Sin(ux*0.02))
		return clamp(0.38+0.55*grain+0.2*valueNoise2D(x*2, y, seed), 0, 1)
	default:
		crack := 0.0
		if x%31 == 0 || y%23 == 0 {
			crack = 0.28
		}
		return clamp(0.46+0.44*valueNoise2D(x*2, y*3, seed)-crack, 0, 1)
	}
}

func smoothPulse(x float64, a float64, b float64, c float64, d float64) float64 {
	return smoothstep(a, b, x) - smoothstep(c, d, x)
}

func smoothstep(edge0 float64, edge1 float64, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func mixColor(a color.RGBA, b color.RGBA, amount float64) color.RGBA {
	amount = clamp(amount, 0, 1)
	return color.RGBA{
		R: uint8(clamp(float64(a.R)*(1-amount)+float64(b.R)*amount, 0, 255)),
		G: uint8(clamp(float64(a.G)*(1-amount)+float64(b.G)*amount, 0, 255)),
		B: uint8(clamp(float64(a.B)*(1-amount)+float64(b.B)*amount, 0, 255)),
		A: 255,
	}
}

func fbmNoise2D(x float64, y float64, seed uint32) float64 {
	amp := 0.5
	freq := 1.0
	sum := 0.0
	for i := 0; i < 4; i++ {
		n := valueNoise2D(int(x*freq), int(y*freq), seed+uint32(i*977))
		sum += n * amp
		amp *= 0.5
		freq *= 2
	}
	return clamp(sum/0.9375, 0, 1)
}

func normalizeVec3(x float64, y float64, z float64) (float64, float64, float64) {
	length := math.Sqrt(x*x + y*y + z*z)
	if length < 0.00001 {
		return 0, 0, 1
	}
	return x / length, y / length, z / length
}

func ellipseSDF(x float64, y float64, rx float64, ry float64) float64 {
	if rx <= 0 || ry <= 0 {
		return 1
	}
	return math.Sqrt((x*x)/(rx*rx)+(y*y)/(ry*ry)) - 1
}

func blendPixel(img *ebiten.Image, x int, y int, col color.RGBA, alpha float64) {
	alpha = clamp(alpha, 0, 1)
	if alpha <= 0 {
		return
	}
	rv, gv, bv, _ := img.At(x, y).RGBA()
	base := color.RGBA{
		R: uint8(rv >> 8),
		G: uint8(gv >> 8),
		B: uint8(bv >> 8),
		A: 255,
	}
	img.Set(x, y, mixColor(base, col, alpha))
}

func drawEllipseFilled(img *ebiten.Image, cx int, cy int, rx int, ry int, col color.RGBA) {
	if rx <= 0 || ry <= 0 {
		return
	}
	x0 := intMax(0, cx-rx)
	x1 := intMin(internalRenderWidth-1, cx+rx)
	y0 := intMax(0, cy-ry)
	y1 := intMin(internalRenderHeight-1, cy+ry)
	rrx := float64(rx * rx)
	rry := float64(ry * ry)
	for y := y0; y <= y1; y++ {
		dy := float64(y - cy)
		for x := x0; x <= x1; x++ {
			dx := float64(x - cx)
			d := (dx*dx)/rrx + (dy*dy)/rry
			if d > 1 {
				continue
			}
			alpha := clamp(1-(d*0.85), 0.4, 1)
			blendPixel(img, x, y, col, alpha)
		}
	}
}

func fillRectGradient(img *ebiten.Image, x int, y int, w int, h int, top color.RGBA, bottom color.RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	x0 := intMax(0, x)
	y0 := intMax(0, y)
	x1 := intMin(internalRenderWidth, x+w)
	y1 := intMin(internalRenderHeight, y+h)
	height := float64(intMax(1, y1-y0))
	for yy := y0; yy < y1; yy++ {
		t := float64(yy-y0) / height
		rowCol := mixColor(top, bottom, t)
		for xx := x0; xx < x1; xx++ {
			img.Set(xx, yy, rowCol)
		}
	}
}

func intMaxF(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
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
