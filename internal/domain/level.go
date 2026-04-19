package domain

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
	for _, thing := range l.Things {
		if thing.Type >= 1 && thing.Type <= 4 {
			return int(thing.X), int(thing.Y), true
		}
	}

	if len(l.Things) == 0 {
		return 0, 0, false
	}

	first := l.Things[0]
	return int(first.X), int(first.Y), true
}
