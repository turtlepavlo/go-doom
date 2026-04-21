package render

import (
	"hash/fnv"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

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
