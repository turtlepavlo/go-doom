package render

import (
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

	walls       []worldWall
	depthBuffer []float64

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
		width:       width,
		height:      height,
		focal:       focal,
		viewportH:   viewportH,
		statusY:     viewportH,
		statusH:     statusBarHeight,
		walls:       collectRenderableWalls(level),
		depthBuffer: make([]float64, internalRenderWidth),
		backbuffer:  nil,
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

	depthBuffer := r.depthBuffer
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
			health:    int(enemy.Health),
			hurtTicks: int(enemy.HurtTicks),
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

	killSlots := intMin(10, int(frame.Kills))
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
