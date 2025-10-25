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

	if d.Rating != 0 {
		return errors.New("cannot change rating")
	}

	if d.TotalRides != 0 {
		return errors.New("cannot change total_rides")
	}

	if d.TotalEarnings != 0 {
		return errors.New("cannot change total_earnings")
	}

	if d.Status != "" {
		return errors.New("cannot change status")
	}

	if d.IsVerified != false {
		return errors.New("cannot change is_verified")
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
