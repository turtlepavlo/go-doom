package domain

import "regexp"

var (
	episodeMapPattern = regexp.MustCompile(`^E[1-9]M[1-9]$`)
	numericMapPattern = regexp.MustCompile(`^MAP[0-9]{2}$`)
)

func BuildMapIndex(lumps []Lump) []Map {
	maps := make([]Map, 0)
	currentMapIndex := -1

	for _, lump := range lumps {
		if IsMapMarker(lump.Name) {
			maps = append(maps, Map{
				Name:  lump.Name,
				Lumps: make([]Lump, 0, 10),
			})
			currentMapIndex = len(maps) - 1
			continue
		}

		if currentMapIndex >= 0 {
			maps[currentMapIndex].Lumps = append(maps[currentMapIndex].Lumps, lump)
		}
	}

	return maps
}

func IsMapMarker(name string) bool {
	return episodeMapPattern.MatchString(name) || numericMapPattern.MatchString(name)
}
