package usecase

import (
	"context"

	"ride-hail/internal/driver/models"
)

func (s *service) UpdateLocation(ctx context.Context, data *models.LocalHistory) (*models.Coordinate, error) {
	update := true

	// if err := s.repo.CheckLocationExists(ctx, data.DriverID); errors.Is(err, pgx.ErrNoRows) {
	// 	update = false
	// } else if err != nil {
	// 	return nil, err
	// }

	return s.repo.UpdateCurrLocation(ctx, data, update)
}
