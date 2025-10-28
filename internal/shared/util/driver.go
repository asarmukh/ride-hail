package util

import (
	"errors"
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

func LocationIsValid(loc models.Location) bool {
	// Latitude: -90 to 90
	if loc.Latitude < -90 || loc.Latitude > 90 {
		return false
	}

	// Longitude: -180 to 180
	if loc.Longitude < -180 || loc.Longitude > 180 {
		return false
	}

	return true
}
