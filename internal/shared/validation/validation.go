package validation

import (
	"errors"
	"fmt"
	"regexp"
)

var uuidRegex = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

// ValidateCoordinates validates latitude and longitude values
func ValidateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return errors.New("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return errors.New("longitude must be between -180 and 180")
	}
	return nil
}

// ValidateUUID validates that a string is a valid UUID
func ValidateUUID(id string) error {
	if !uuidRegex.MatchString(id) {
		return errors.New("invalid UUID format")
	}
	return nil
}

// ValidateRideType validates that a ride type is one of the allowed values
func ValidateRideType(rideType string) error {
	validTypes := []string{"ECONOMY", "PREMIUM", "XL"}
	for _, validType := range validTypes {
		if rideType == validType {
			return nil
		}
	}
	return fmt.Errorf("invalid ride type: must be one of %v", validTypes)
}

// ValidateVehicleType validates that a vehicle type is one of the allowed values
func ValidateVehicleType(vehicleType string) error {
	validTypes := []string{"SEDAN", "SUV", "VAN"}
	for _, validType := range validTypes {
		if vehicleType == validType {
			return nil
		}
	}
	return fmt.Errorf("invalid vehicle type: must be one of %v", validTypes)
}

// ValidatePositiveFloat validates that a float is positive
func ValidatePositiveFloat(value float64, fieldName string) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", fieldName)
	}
	return nil
}

// ValidatePositiveInt validates that an int is positive
func ValidatePositiveInt(value int, fieldName string) error {
	if value <= 0 {
		return fmt.Errorf("%s must be positive", fieldName)
	}
	return nil
}

// ValidateNonNegativeFloat validates that a float is non-negative
func ValidateNonNegativeFloat(value float64, fieldName string) error {
	if value < 0 {
		return fmt.Errorf("%s must be non-negative", fieldName)
	}
	return nil
}

// ValidateStringNotEmpty validates that a string is not empty
func ValidateStringNotEmpty(value, fieldName string) error {
	if value == "" {
		return fmt.Errorf("%s cannot be empty", fieldName)
	}
	return nil
}

// ValidatePaginationParams validates pagination parameters
func ValidatePaginationParams(page, pageSize int) error {
	if page < 1 {
		return errors.New("page must be >= 1")
	}
	if pageSize < 1 {
		return errors.New("page_size must be >= 1")
	}
	if pageSize > 100 {
		return errors.New("page_size must be <= 100")
	}
	return nil
}

// ValidateSpeed validates speed in km/h
func ValidateSpeed(speed float64) error {
	if speed < 0 {
		return errors.New("speed must be non-negative")
	}
	if speed > 300 {
		return errors.New("speed exceeds reasonable limit (300 km/h)")
	}
	return nil
}

// ValidateHeading validates heading in degrees
func ValidateHeading(heading float64) error {
	if heading < 0 || heading > 360 {
		return errors.New("heading must be between 0 and 360 degrees")
	}
	return nil
}

// ValidateAccuracy validates GPS accuracy in meters
func ValidateAccuracy(accuracy float64) error {
	if accuracy < 0 {
		return errors.New("accuracy must be non-negative")
	}
	if accuracy > 10000 {
		return errors.New("accuracy exceeds reasonable limit (10km)")
	}
	return nil
}
