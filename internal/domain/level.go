package domain

import "math"

type Vertex struct {
	X int16
	Y int16
}

type Thing struct {
	X     int16
	Y     int16
	Angle uint16
	Type  uint16
	Flags uint16
}

type Linedef struct {
	StartVertex uint16
	EndVertex   uint16
	Flags       uint16
	SpecialType uint16
	SectorTag   uint16
	RightSide   uint16
	LeftSide    uint16
}

type Sidedef struct {
	XOffset       int16
	YOffset       int16
	UpperTexture  string
	LowerTexture  string
	MiddleTexture string
	Sector        uint16
}

type Sector struct {
	FloorHeight    int16
	CeilingHeight  int16
	FloorTexture   string
	CeilingTexture string
	LightLevel     uint16
	SpecialType    uint16
	Tag            uint16
}

type Level struct {
	Name     string
	Things   []Thing
	Linedefs []Linedef
	Sidedefs []Sidedef
	Vertexes []Vertex
	Sectors  []Sector
}

func NewLevel(
	name string,
	things []Thing,
	linedefs []Linedef,
	sidedefs []Sidedef,
	vertexes []Vertex,
	sectors []Sector,
) (Level, error) {
	if name == "" {
		return Level{}, ErrEmptyMapName
	}

	return Level{
		Name:     name,
		Things:   append([]Thing(nil), things...),
		Linedefs: append([]Linedef(nil), linedefs...),
		Sidedefs: append([]Sidedef(nil), sidedefs...),
		Vertexes: append([]Vertex(nil), vertexes...),
		Sectors:  append([]Sector(nil), sectors...),
	}, nil
}

func (l Level) PlayerStart() (x int, y int, ok bool) {
	spawn, found := l.PlayerSpawn()
	if !found {
		return 0, 0, false
	}
	return spawn.X, spawn.Y, true
}

type PlayerSpawn struct {
	X     int
	Y     int
	Angle float64
}

func (l Level) PlayerSpawn() (spawn PlayerSpawn, ok bool) {
	for _, thing := range l.Things {
		if thing.Type >= 1 && thing.Type <= 4 {
			return PlayerSpawn{
				X:     int(thing.X),
				Y:     int(thing.Y),
				Angle: normalizeThingAngle(thing.Angle),
			}, true
		}
	}

	if len(l.Things) == 0 {
		return PlayerSpawn{}, false
	}

	first := l.Things[0]
	return PlayerSpawn{
		X:     int(first.X),
		Y:     int(first.Y),
		Angle: normalizeThingAngle(first.Angle),
	}, true
}

func normalizeThingAngle(degrees uint16) float64 {
	angle := float64(degrees) * math.Pi / 180
	for angle > math.Pi {
		angle -= 2 * math.Pi
	}
	for angle < -math.Pi {
		angle += 2 * math.Pi
	}
	return angle
}
