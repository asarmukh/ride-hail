package apperrors

import (
	"errors"
)

var (
	ErrDriverOnline   = errors.New("Driver is alredy online")
	ErrDriverNotFound = errors.New("Driver is not found")
)

func CheckError(err error) int {
	if errors.Is(err, ErrDriverOnline) {
		return 401
	}

	if errors.Is(err, ErrDriverNotFound) {
		return 404
	}

	return 500
}
