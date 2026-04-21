package loadiwad

// DTO contracts for the loadiwad application service.

type RawArchive struct {
	Type            string
	LumpCount       uint32
	DirectoryOffset uint32
	Lumps           []RawLump
}

type RawLump struct {
	Name   string
	Offset uint32
	Size   uint32
}
