package runtime

import (
	"context"
	"errors"
	"math"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

const (
	playerCollisionRadius = 18.0
	minPassageHeight      = 56.0
	linedefFlagBlocking   = 1 << 0

	playerStartHealth = 100
	playerStartAmmo   = 60

	weaponFireCooldownTicks = 7
	weaponFlashTicksOnFire  = 3
	damageFlashTicksOnHit   = 4
	hitscanRange            = 1400.0
)

var ErrNilLevel = errors.New("nil level")

type enemyProfile struct {
	kind           string
	maxHealth      int64
	radius         float64
	speed          float64
	attackRange    float64
	attackMinDmg   int64
	attackMaxDmg   int64
	attackCooldown int64
}

type enemyState struct {
	x float64
	y float64

	typeID uint16
	kind   string
	health int64
	radius float64
	speed  float64

	attackRange    float64
	attackMinDmg   int64
	attackMaxDmg   int64
	attackCooldown int64

	cooldown  int64
	hurtTicks int64
}

type blockingSegment struct {
	ax float64
	ay float64
	bx float64
	by float64
}

type LevelSimulation struct {
	engine   *Engine
	segments []blockingSegment
	enemies  []enemyState

	enemySnapshots []domain.EnemySnapshot

	playerHealth int64
	ammo         int64

	weaponCooldown   int64
	weaponFlashTicks int64
	damageFlashTicks int64

	kills      int64
	shotsFired int64
	shotHits   int64
}

func NewLevelSimulation(engine *Engine, level *domain.Level) (*LevelSimulation, error) {
	if engine == nil {
		return nil, ErrNilEngine
	}
	if level == nil {
		return nil, ErrNilLevel
	}

	segments := collectBlockingSegments(*level)
	return &LevelSimulation{
		engine:         engine,
		segments:       segments,
		enemies:        collectEnemyStates(*level),
		enemySnapshots: make([]domain.EnemySnapshot, 0, len(level.Things)),
		playerHealth:   playerStartHealth,
		ammo:           playerStartAmmo,
	}, nil
}

func (s *LevelSimulation) Frame() domain.Frame {
	return s.decorateFrame(s.engine.Frame())
}

func (s *LevelSimulation) Step(ctx context.Context, commands []domain.Command) (domain.Frame, error) {
	prev := s.engine.State()
	frame := s.engine.Step(ctx, commands)
	if !frame.Running {
		return s.decorateFrame(frame), nil
	}

	if s.collides(float64(frame.PlayerX), float64(frame.PlayerY)) {
		nextX := float64(frame.PlayerX)
		nextY := float64(frame.PlayerY)
		prevX := float64(prev.PlayerX)
		prevY := float64(prev.PlayerY)

		switch {
		case !s.collides(nextX, prevY):
			s.engine.SetPlayerPosition(int64(math.Round(nextX)), int64(math.Round(prevY)))
		case !s.collides(prevX, nextY):
			s.engine.SetPlayerPosition(int64(math.Round(prevX)), int64(math.Round(nextY)))
		default:
			s.engine.SetPlayerPosition(prev.PlayerX, prev.PlayerY)
		}
		frame = s.engine.Frame()
	}

	s.tickCombatTimers()
	if hasCommand(commands, domain.CommandFire) {
		s.tryFire(frame)
	}
	s.stepEnemies(frame)

	if s.playerHealth <= 0 {
		s.playerHealth = 0
		s.engine.Stop()
		frame = s.engine.Frame()
	}

	return s.decorateFrame(frame), nil
}

func (s *LevelSimulation) decorateFrame(frame domain.Frame) domain.Frame {
	var alive int64
	s.enemySnapshots = s.enemySnapshots[:0]
	for _, enemy := range s.enemies {
		if enemy.health > 0 {
			alive++
		}
		s.enemySnapshots = append(s.enemySnapshots, domain.EnemySnapshot{
			X:         int64(math.Round(enemy.x)),
			Y:         int64(math.Round(enemy.y)),
			TypeID:    enemy.typeID,
			Kind:      enemy.kind,
			Health:    maxInt64(enemy.health, 0),
			HurtTicks: enemy.hurtTicks,
			Alive:     enemy.health > 0,
		})
	}

	frame.Health = maxInt64(s.playerHealth, 0)
	frame.Ammo = maxInt64(s.ammo, 0)
	frame.EnemyCount = int64(len(s.enemies))
	frame.EnemyAlive = alive
	frame.Kills = s.kills
	frame.ShotsFired = s.shotsFired
	frame.ShotHits = s.shotHits
	frame.WeaponCooldown = s.weaponCooldown
	frame.WeaponFlashTicks = s.weaponFlashTicks
	frame.DamageFlashTicks = s.damageFlashTicks
	frame.Enemies = s.enemySnapshots
	return frame
}

func (s *LevelSimulation) tickCombatTimers() {
	if s.weaponCooldown > 0 {
		s.weaponCooldown--
	}
	if s.weaponFlashTicks > 0 {
		s.weaponFlashTicks--
	}
	if s.damageFlashTicks > 0 {
		s.damageFlashTicks--
	}
}

func (s *LevelSimulation) tryFire(frame domain.Frame) {
	if s.ammo <= 0 || s.weaponCooldown > 0 {
		return
	}

	s.ammo--
	s.shotsFired++
	s.weaponCooldown = weaponFireCooldownTicks
	s.weaponFlashTicks = weaponFlashTicksOnFire

	targetIdx, targetDistance := s.pickShootTarget(frame)
	if targetIdx < 0 {
		return
	}

	damage := shotgunLikeDamage(frame.Tick, targetDistance)
	enemy := &s.enemies[targetIdx]
	enemy.health -= damage
	enemy.hurtTicks = 4
	s.shotHits++
	if enemy.health <= 0 {
		enemy.health = 0
		enemy.hurtTicks = 6
		s.kills++
	}
}

func (s *LevelSimulation) pickShootTarget(frame domain.Frame) (idx int, distance float64) {
	idx = -1
	bestScore := math.Inf(1)
	px := float64(frame.PlayerX)
	py := float64(frame.PlayerY)

	for i := range s.enemies {
		enemy := s.enemies[i]
		if enemy.health <= 0 {
			continue
		}

		dx := enemy.x - px
		dy := enemy.y - py
		dist := math.Hypot(dx, dy)
		if dist <= 0.001 || dist > hitscanRange {
			continue
		}

		aimDelta := normalizeAngle(math.Atan2(dy, dx) - frame.Angle)
		aimAllowance := 0.06 + enemy.radius/math.Max(1, dist)
		if math.Abs(aimDelta) > aimAllowance {
			continue
		}

		if s.segmentBlocked(px, py, enemy.x, enemy.y) {
			continue
		}

		centerPenalty := math.Abs(aimDelta) * 240
		score := dist + centerPenalty
		if score < bestScore {
			bestScore = score
			distance = dist
			idx = i
		}
	}

	return idx, distance
}

func (s *LevelSimulation) stepEnemies(frame domain.Frame) {
	px := float64(frame.PlayerX)
	py := float64(frame.PlayerY)

	for i := range s.enemies {
		enemy := &s.enemies[i]
		if enemy.health <= 0 {
			continue
		}

		if enemy.cooldown > 0 {
			enemy.cooldown--
		}
		if enemy.hurtTicks > 0 {
			enemy.hurtTicks--
		}

		dx := px - enemy.x
		dy := py - enemy.y
		dist := math.Hypot(dx, dy)
		if dist < 0.001 {
			dist = 0.001
		}

		hasLOS := !s.segmentBlocked(enemy.x, enemy.y, px, py)
		if hasLOS && dist <= enemy.attackRange && enemy.cooldown == 0 {
			damage := enemyAttackDamage(*enemy, frame.Tick+uint64(i*13))
			s.playerHealth -= damage
			s.damageFlashTicks = maxInt64(s.damageFlashTicks, damageFlashTicksOnHit)
			enemy.cooldown = enemy.attackCooldown
			continue
		}

		if dist <= enemy.attackRange*0.7 && hasLOS {
			continue
		}

		step := enemy.speed
		moveX := enemy.x + (dx/dist)*step
		moveY := enemy.y + (dy/dist)*step
		s.tryMoveEnemy(enemy, moveX, moveY, px, py)
	}
}

func (s *LevelSimulation) tryMoveEnemy(enemy *enemyState, moveX float64, moveY float64, playerX float64, playerY float64) {
	if !s.enemyCollides(enemy, moveX, moveY, playerX, playerY) {
		enemy.x = moveX
		enemy.y = moveY
		return
	}

	if !s.enemyCollides(enemy, moveX, enemy.y, playerX, playerY) {
		enemy.x = moveX
		return
	}

	if !s.enemyCollides(enemy, enemy.x, moveY, playerX, playerY) {
		enemy.y = moveY
	}
}

func (s *LevelSimulation) enemyCollides(enemy *enemyState, x float64, y float64, playerX float64, playerY float64) bool {
	if s.collidesRadius(x, y, enemy.radius) {
		return true
	}

	minPlayerDistance := enemy.radius + playerCollisionRadius + 6
	if math.Hypot(playerX-x, playerY-y) < minPlayerDistance {
		return true
	}

	for i := range s.enemies {
		other := &s.enemies[i]
		if other == enemy || other.health <= 0 {
			continue
		}
		if math.Hypot(other.x-x, other.y-y) < (other.radius+enemy.radius)*0.9 {
			return true
		}
	}

	return false
}

func (s *LevelSimulation) collides(playerX float64, playerY float64) bool {
	return s.collidesRadius(playerX, playerY, playerCollisionRadius)
}

func (s *LevelSimulation) collidesRadius(centerX float64, centerY float64, radius float64) bool {
	for _, segment := range s.segments {
		if pointSegmentDistance(centerX, centerY, segment.ax, segment.ay, segment.bx, segment.by) <= radius {
			return true
		}
	}
	return false
}

func (s *LevelSimulation) segmentBlocked(ax float64, ay float64, bx float64, by float64) bool {
	for _, segment := range s.segments {
		hit, t := segmentIntersection(ax, ay, bx, by, segment.ax, segment.ay, segment.bx, segment.by)
		if hit && t > 0.01 && t < 0.99 {
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
		if aIdx >= len(level.Vertexes) || bIdx >= len(level.Vertexes) {
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

func collectEnemyStates(level domain.Level) []enemyState {
	enemies := make([]enemyState, 0, len(level.Things))
	for _, thing := range level.Things {
		profile, ok := enemyProfileByType(thing.Type)
		if !ok {
			continue
		}

		enemies = append(enemies, enemyState{
			x:              float64(thing.X),
			y:              float64(thing.Y),
			typeID:         thing.Type,
			kind:           profile.kind,
			health:         profile.maxHealth,
			radius:         profile.radius,
			speed:          profile.speed,
			attackRange:    profile.attackRange,
			attackMinDmg:   profile.attackMinDmg,
			attackMaxDmg:   profile.attackMaxDmg,
			attackCooldown: profile.attackCooldown,
		})
	}

	return enemies
}

func enemyProfileByType(typeID uint16) (enemyProfile, bool) {
	switch typeID {
	case 3004, 9, 65:
		return enemyProfile{
			kind:           "ZOMBIE",
			maxHealth:      20,
			radius:         20,
			speed:          4.5,
			attackRange:    380,
			attackMinDmg:   3,
			attackMaxDmg:   12,
			attackCooldown: 24,
		}, true
	case 3001:
		return enemyProfile{
			kind:           "IMP",
			maxHealth:      60,
			radius:         22,
			speed:          4.0,
			attackRange:    340,
			attackMinDmg:   4,
			attackMaxDmg:   16,
			attackCooldown: 30,
		}, true
	case 3002, 58:
		return enemyProfile{
			kind:           "DEMON",
			maxHealth:      150,
			radius:         28,
			speed:          5.5,
			attackRange:    58,
			attackMinDmg:   8,
			attackMaxDmg:   24,
			attackCooldown: 18,
		}, true
	case 3005, 69, 3003, 66, 67, 68, 71:
		return enemyProfile{
			kind:           "HEAVY",
			maxHealth:      240,
			radius:         30,
			speed:          3.6,
			attackRange:    420,
			attackMinDmg:   8,
			attackMaxDmg:   22,
			attackCooldown: 32,
		}, true
	case 3006, 64:
		return enemyProfile{
			kind:           "LOSTSOUL",
			maxHealth:      100,
			radius:         20,
			speed:          6.0,
			attackRange:    78,
			attackMinDmg:   6,
			attackMaxDmg:   20,
			attackCooldown: 16,
		}, true
	default:
		return enemyProfile{}, false
	}
}

func hasCommand(commands []domain.Command, expected domain.Command) bool {
	for _, command := range commands {
		if command == expected {
			return true
		}
	}
	return false
}

func isBlockingLine(level domain.Level, line domain.Linedef) bool {
	if (line.Flags & linedefFlagBlocking) != 0 {
		return true
	}

	const noSide = math.MaxUint16

	rightIdx := int(line.RightSide)
	leftIdx := int(line.LeftSide)
	if line.RightSide == noSide || rightIdx >= len(level.Sidedefs) {
		return true
	}
	if line.LeftSide == noSide || leftIdx >= len(level.Sidedefs) {
		return true
	}

	rightSectorIdx := int(level.Sidedefs[rightIdx].Sector)
	leftSectorIdx := int(level.Sidedefs[leftIdx].Sector)
	if rightSectorIdx >= len(level.Sectors) || leftSectorIdx >= len(level.Sectors) {
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

func segmentIntersection(ax float64, ay float64, bx float64, by float64, cx float64, cy float64, dx float64, dy float64) (hit bool, t float64) {
	abx := bx - ax
	aby := by - ay
	cdx := dx - cx
	cdy := dy - cy

	denominator := abx*cdy - aby*cdx
	if math.Abs(denominator) < 0.00001 {
		return false, 0
	}

	acx := cx - ax
	acy := cy - ay

	t = (acx*cdy - acy*cdx) / denominator
	u := (acx*aby - acy*abx) / denominator
	return t >= 0 && t <= 1 && u >= 0 && u <= 1, t
}

func enemyAttackDamage(enemy enemyState, tick uint64) int64 {
	if enemy.attackMaxDmg <= enemy.attackMinDmg {
		return enemy.attackMinDmg
	}
	rangeSize := enemy.attackMaxDmg - enemy.attackMinDmg + 1
	return enemy.attackMinDmg + int64((tick+uint64(enemy.typeID))%uint64(rangeSize))
}

func shotgunLikeDamage(tick uint64, distance float64) int64 {
	base := int64(24 + int((tick*7)%26))
	if distance < 200 {
		base += 22
	} else if distance < 420 {
		base += 10
	} else if distance > 900 {
		base -= 10
	}
	return maxInt64(base, 8)
}

func normalizeAngle(angle float64) float64 {
	const twoPi = 2 * math.Pi
	for angle > math.Pi {
		angle -= twoPi
	}
	for angle < -math.Pi {
		angle += twoPi
	}
	return angle
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
