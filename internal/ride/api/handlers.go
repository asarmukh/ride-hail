package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"ride-hail/internal/ride/domain"
	"strings"
	"time"
)

func (h *Handler) CreateRideHandler(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) CancelRideHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 3 || pathParts[0] != "rides" || pathParts[2] != "cancel" {
		http.Error(w, "invalid URL path", http.StatusBadRequest)
		return
	}
	rideID := pathParts[1]

	var body struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if body.Reason == "" {
		body.Reason = "Cancelled by passenger"
	}

	passengerID := r.Header.Get("X-Passenger-ID")
	if passengerID == "" {
		http.Error(w, "missing passenger_id (use X-Passenger-ID header)", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	refundPercent, err := h.service.CancelRide(ctx, rideID, passengerID, body.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			http.Error(w, "ride not found", http.StatusNotFound)
			return
		case errors.Is(err, domain.ErrForbidden):
			http.Error(w, "you cannot cancel this ride", http.StatusForbidden)
			return
		case errors.Is(err, domain.ErrInvalidStatus):
			http.Error(w, "ride cannot be cancelled at this stage", http.StatusConflict)
			return
		default:
			http.Error(w, "failed to cancel ride", http.StatusInternalServerError)
			return
		}
	}

	resp := map[string]interface{}{
		"ride_id":        rideID,
		"status":         "CANCELLED",
		"refund_percent": refundPercent,
		"message":        "Ride cancelled successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
