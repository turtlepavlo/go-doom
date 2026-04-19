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
