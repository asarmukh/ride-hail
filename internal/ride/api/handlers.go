package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"
	"strings"
	"time"
)

func (h *Handler) CreateRideHandler(w http.ResponseWriter, r *http.Request) {
	passengerID, ok := r.Context().Value("passenger_id").(string)
	if !ok || passengerID == "" {
		util.WriteJSONError(w, "unauthorized: missing passenger_id", http.StatusUnauthorized)
		return
	}

	role, _ := r.Context().Value("role").(string)
	if role != "PASSENGER" {
		util.WriteJSONError(w, "forbidden: only passengers can create rides", http.StatusForbidden)
		return
	}

	var input domain.CreateRideRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if input.PickupAddress == "" || input.DropoffAddress == "" || input.RideType == "" {
		util.WriteJSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	ride, err := h.service.CreateRide(ctx, passengerID, input)
	if err != nil {
		util.WriteJSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := domain.CreateRideResponse{
		RideID:                ride.ID,
		RideNumber:            ride.Number,
		Status:                ride.Status,
		EstimatedFare:         ride.EstimatedFare,
		EstimatedDurationMins: ride.EstimatedDuration,
		EstimatedDistanceKm:   ride.EstimatedDistance,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) CancelRideHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "rides" || pathParts[2] != "cancel" {
		util.WriteJSONError(w, "invalid URL path", http.StatusBadRequest)
		return
	}
	rideID := pathParts[1]

	passengerID, ok := r.Context().Value("passenger_id").(string)
	if !ok || passengerID == "" {
		util.WriteJSONError(w, "unauthorized: missing passenger_id", http.StatusUnauthorized)
		return
	}

	role, _ := r.Context().Value("role").(string)
	if role != "PASSENGER" {
		util.WriteJSONError(w, "forbidden: only passengers can cancel rides", http.StatusForbidden)
	}

	var body struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "Cancelled by passenger"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	refundPercent, err := h.service.CancelRide(ctx, rideID, passengerID, body.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			util.WriteJSONError(w, "ride not found", http.StatusNotFound)
			return
		case errors.Is(err, domain.ErrForbidden):
			util.WriteJSONError(w, "you cannot cancel this ride", http.StatusForbidden)
			return
		case errors.Is(err, domain.ErrInvalidStatus):
			util.WriteJSONError(w, "ride cannot be cancelled at this stage", http.StatusConflict)
			return
		default:
			util.WriteJSONError(w, "failed to cancel ride", http.StatusInternalServerError)
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
