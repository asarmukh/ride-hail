package app

import (
	"context"
	"time"

	"ride-hail/internal/admin/repo"
)

type AdminService struct {
	repo *repo.AdminRepo
}

func NewAdminService(repo *repo.AdminRepo) *AdminService {
	return &AdminService{repo: repo}
}

type OverviewResponse struct {
	Timestamp          string                  `json:"timestamp"`
	Metrics            *repo.SystemMetrics     `json:"metrics"`
	DriverDistribution repo.DriverDistribution `json:"driver_distribution"`
	Hotspots           []Hotspot               `json:"hotspots"`
}

type Hotspot struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}

type ActiveRidesResponse struct {
	Rides      []repo.ActiveRide `json:"rides"`
	TotalCount int               `json:"total_count"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

func (s *AdminService) GetSystemOverview(ctx context.Context) (*OverviewResponse, error) {
	metrics, err := s.repo.GetSystemMetrics(ctx)
	if err != nil {
		return nil, err
	}

	distribution, err := s.repo.GetDriverDistribution(ctx)
	if err != nil {
		return nil, err
	}

	// TODO: Implement hotspots calculation based on geospatial data
	// For now, return empty hotspots
	hotspots := []Hotspot{}

	return &OverviewResponse{
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		Metrics:            metrics,
		DriverDistribution: distribution,
		Hotspots:           hotspots,
	}, nil
}

func (s *AdminService) GetActiveRides(ctx context.Context, page, pageSize int) (*ActiveRidesResponse, error) {
	// Set defaults
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	rides, totalCount, err := s.repo.GetActiveRides(ctx, page, pageSize)
	if err != nil {
		return nil, err
	}

	return &ActiveRidesResponse{
		Rides:      rides,
		TotalCount: totalCount,
		Page:       page,
		PageSize:   pageSize,
	}, nil
}
