package apperrors

import (
	"errors"
)

var ErrDriverOnline = errors.New("Driver is alredy online")

func CheckError(err error) int {
	if errors.Is(err, ErrDriverOnline) {
		return 401
	}

	return 500
}
