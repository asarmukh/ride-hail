package api

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/ride/app"
	"ride-hail/internal/ride/domain"
	"time"
)

type RideHandler struct {
	service *app.RideService
}

func NewRideHandler(service *app.RideService) *RideHandler {
	return &RideHandler{service: service}
}

func (h *RideHandler) CreateRideHandler(w http.ResponseWriter, r *http.Request) {
	var input domain.CreateRideInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if input.PassengerID == "" ||
		input.PickupAddress == "" ||
		input.DropoffAddress == "" ||
		input.RideType == "" {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ride, err := h.service.CreateRide(ctx, input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := map[string]interface{}{
		"ride_id":                    ride.ID,
		"ride_number":                ride.Number,
		"status":                     ride.Status,
		"estimated_fare":             ride.EstimatedFare,
		"estimated_duration_minutes": ride.EstimatedDuration,
		"estimated_distance_km":      ride.EstimatedDistance,
	}
	_ = json.NewEncoder(w).Encode(resp)
}
