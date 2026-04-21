package domain

type GameState struct {
	Tick    uint64
	PlayerX int64
	PlayerY int64
	Angle   float64
	Running bool
}

type Frame struct {
	Tick             uint64
	PlayerX          int64
	PlayerY          int64
	Angle            float64
	Running          bool
	Health           int64
	Ammo             int64
	EnemyCount       int64
	EnemyAlive       int64
	Kills            int64
	ShotsFired       int64
	ShotHits         int64
	WeaponCooldown   int64
	WeaponFlashTicks int64
	DamageFlashTicks int64
	Enemies          []EnemySnapshot
}

type EnemySnapshot struct {
	X         int64
	Y         int64
	TypeID    uint16
	Kind      string
	Health    int64
	HurtTicks int64
	Alive     bool
}
