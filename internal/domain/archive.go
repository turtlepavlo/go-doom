package domain

type Header struct {
	Kind            string
	LumpCount       int64
	DirectoryOffset int64
}

type Lump struct {
	Name   string
	Offset int64
	Size   int64
}

type Map struct {
	Name  string
	Lumps []Lump
}

type Archive struct {
	Header Header
	Lumps  []Lump
	Maps   []Map
}
