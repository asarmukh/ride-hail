package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ride-hail/internal/driver/models"
)

func CheckDriverData(d models.Driver) error {
	// License number: required, trim spaces
	if strings.TrimSpace(d.LicenseNumber) == "" {
		return errors.New("license_number is required")
	}

	// Vehicle type: required, should be one of allowed options
	allowedTypes := map[string]bool{
		"ECONOMY": true,
		"PREMIUM": true,
		"XL":      true,
	}
	vehicleType := strings.ToUpper(strings.TrimSpace(d.VehicleType))
	if !allowedTypes[vehicleType] {
		return errors.New("vehicle_type must be one of: ECONOMY, PREMIUM, XL")
	}

	err := checkVehicleAttributes(d.VehicleAttrs)
	if err != nil {
		return err
	}

	return nil
}

func checkVehicleAttributes(v models.VehicleAttributes) error {
	if strings.TrimSpace(v.Color) == "" {
		return errors.New("vehicle_attrs.color is required")
	}
	if strings.TrimSpace(v.Model) == "" {
		return errors.New("vehicle_attrs.model is required")
	}

	currentYear := time.Now().Year()
	if v.Year < 1990 || v.Year > currentYear {
		return errors.New("vehicle_attrs.year must be between 1990 and current year")
	}

	return nil
}

func ValidateLocation(location models.Location) error {
	// Latitude: -90 to 90
	if location.Latitude < -90 || location.Latitude > 90 {
		return fmt.Errorf("latitude must be  between => 180 > x > -180")
	}

	// Longitude: -180 to 180
	if location.Longitude < -180 || location.Longitude > 180 {
		return fmt.Errorf("longitude must be between => 90 > x > -90")
	}

	return nil
}

func ValidateCompleteRideRequest(finalLocation models.Location, distance float64, duration int) error {
	if distance <= 0 {
		return fmt.Errorf("actual_distance_km must be greater than 0")
	}

	if duration <= 0 {
		return fmt.Errorf("actual_duration_minutes must be greater than 0")
	}

	err := ValidateLocation(finalLocation)
	if err != nil {
		return err
	}

	return nil
}

// calculateFinalFare calculates the final fare based on actual distance and duration
func CalculateFinalFare(vehicleType string, distanceKM, durationMinutes float64) float64 {
	var baseFare, ratePerKM, ratePerMin float64

	switch vehicleType {
	case "ECONOMY":
		baseFare = 500.0
		ratePerKM = 100.0
		ratePerMin = 50.0
	case "PREMIUM":
		baseFare = 800.0
		ratePerKM = 120.0
		ratePerMin = 60.0
	case "XL":
		baseFare = 1000.0
		ratePerKM = 150.0
		ratePerMin = 75.0
	default:
		baseFare = 500.0
		ratePerKM = 100.0
		ratePerMin = 50.0
	}

	finalFare := baseFare + (distanceKM * ratePerKM) + (durationMinutes * ratePerMin)

	// Apply minimum fare
	if finalFare < baseFare {
		finalFare = baseFare
	}

	return finalFare
}

// calculateDriverEarnings calculates driver's share of the fare
func CalculateDriverEarnings(finalFare float64, vehicleType string) float64 {
	// Driver gets 80% of the fare across all vehicle types
	driverShare := 0.8
	return finalFare * driverShare
}
