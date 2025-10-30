package util

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
)

func GenerateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}

	// Set version (4) and variant bits to match UUID v4 format
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	uuid := fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4],
		b[4:6],
		b[6:8],
		b[8:10],
		b[10:16])

	return uuid, nil
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

func WriteJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
