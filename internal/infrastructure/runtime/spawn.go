package runtime

import (
	"math"

	"github.com/turtlepavlo/go-doom/internal/domain"
)

type PlayerSpawn struct {
	X     int64
	Y     int64
	Angle float64
}

func FindPlayerSpawn(level domain.Level) (spawn PlayerSpawn, ok bool) {
	for _, thing := range level.Things {
		if thing.Type >= 1 && thing.Type <= 4 {
			return PlayerSpawn{
				X:     int64(thing.X),
				Y:     int64(thing.Y),
				Angle: normalizeThingAngle(thing.Angle),
			}, true
		}
	}

	if len(level.Things) == 0 {
		return PlayerSpawn{}, false
	}

	first := level.Things[0]
	return PlayerSpawn{
		X:     int64(first.X),
		Y:     int64(first.Y),
		Angle: normalizeThingAngle(first.Angle),
	}, true
}

func normalizeThingAngle(degrees uint16) float64 {
	angle := float64(degrees) * math.Pi / 180
	return math.Mod(angle+math.Pi, 2*math.Pi) - math.Pi
}
