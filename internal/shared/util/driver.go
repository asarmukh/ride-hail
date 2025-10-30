package util

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ride-hail/internal/driver/models"
)

func CheckDriverData(driverData *models.Driver, isDuplicateLicenseNumber bool) error {
	// License number: required, trim spaces
	if strings.TrimSpace(driverData.LicenseNumber) == "" {
		return errors.New("license_number is required")
	}

	if isDuplicateLicenseNumber {
		return errors.New("license_number already exists")
	}

	// Vehicle type: required, should be one of allowed options
	allowedTypes := map[string]bool{
		"ECONOMY": true,
		"PREMIUM": true,
		"XL":      true,
	}
	(*driverData).VehicleType = strings.ToUpper(strings.TrimSpace((*driverData).VehicleType))
	if !allowedTypes[driverData.VehicleType] {
		return errors.New("vehicle_type must be one of: ECONOMY, PREMIUM, XL")
	}

	err := checkVehicleAttributes((*driverData).VehicleAttrs)
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
