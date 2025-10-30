package usecase

import (
	"context"
	"fmt"
	"net/http"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
)

func (s *service) RegisterDriver(ctx context.Context, driverData *models.Driver) (int, error) {
	err := s.repo.CheckUserExistsAndIsDriver(ctx, driverData.ID)
	if err != nil {
		fmt.Println("driver does not exist with such id: ", driverData.ID)
		return http.StatusBadRequest, err
	}

	isDuplicateLicenseNumber, err := s.repo.CheckLicenseNumberExists(ctx, (*driverData).LicenseNumber)
	if err != nil {
		fmt.Println("invalid driver data: ", err)
		return http.StatusBadRequest, err
	}

	err = util.CheckDriverData(driverData, isDuplicateLicenseNumber)
	if err != nil {
		fmt.Println("invalid driver data: ", err)
		return http.StatusBadRequest, err
	}

	err = s.repo.InsertDriver(ctx, driverData)
	if err != nil {
		fmt.Println("could not insert driver data to db: ", err)
		return http.StatusBadGateway, err
	}

	return 0, nil
}
