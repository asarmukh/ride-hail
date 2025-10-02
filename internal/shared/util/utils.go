package util

import (
	"fmt"
	"math"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.New().String()
}

func toRadians(degree float64) float64 {
	return degree * math.Pi / 180
}

func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	// R is the earth radius in km
	const R = 6371

	phi1 := toRadians(lat1)
	phi2 := toRadians(lat2)
	deltaPhi := toRadians(lat2 - lat1)
	deltaLambda := toRadians(lon2 - lon1)

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distance := R * c
	fmt.Println(distance)

	return distance
}
