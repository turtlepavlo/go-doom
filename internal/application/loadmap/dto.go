package loadmap

// DTO contracts for the loadmap application service.

type RawMap struct {
	Name     string
	Things   []byte
	Linedefs []byte
	Sidedefs []byte
	Vertexes []byte
	Sectors  []byte
}
