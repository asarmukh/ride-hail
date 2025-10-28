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
	logger := util.New()
	start := time.Now()

	passengerID, ok := r.Context().Value("passenger_id").(string)
	if !ok || passengerID == "" {
		logger.Warn("CreateRideHandler", "unauthorized request: missing passenger_id")
		util.WriteJSONError(w, "unauthorized: missing passenger_id", http.StatusUnauthorized)
		return
	}

	role, _ := r.Context().Value("role").(string)
	if role != "PASSENGER" {
		logger.Warn("CreateRideHandler", "forbidden: role is not PASSENGER")
		util.WriteJSONError(w, "forbidden: only passengers can create rides", http.StatusForbidden)
		return
	}

	var input domain.CreateRideRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		logger.Error("CreateRideHandler", err)
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if input.PickupAddress == "" || input.DropoffAddress == "" || input.RideType == "" {
		logger.Warn("CreateRideHandler", "missing required fields in request")
		util.WriteJSONError(w, "missing required fields", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	logger.Info("CreateRideHandler", "creating new ride...")
	ride, err := h.service.CreateRide(ctx, passengerID, input)
	if err != nil {
		logger.Error("CreateRideHandler", err)
		util.WriteJSONError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	resp := domain.CreateRideResponse{
		RideID:                ride.ID,
		RideNumber:            ride.Number,
		Status:                ride.Status,
		EstimatedFare:         ride.EstimatedFare,
		EstimatedDurationMins: ride.EstimatedDuration,
		EstimatedDistanceKm:   ride.EstimatedDistance,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)

	logger.OK("CreateRideHandler", "ride created successfully: "+ride.ID)
	logger.HTTP(http.StatusCreated, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
}

func (h *Handler) CancelRideHandler(w http.ResponseWriter, r *http.Request) {
	logger := util.New()
	start := time.Now()

	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 3 || pathParts[0] != "rides" || pathParts[2] != "cancel" {
		logger.Warn("CancelRideHandler", "invalid URL path: "+r.URL.Path)
		util.WriteJSONError(w, "invalid URL path", http.StatusBadRequest)
		return
	}
	rideID := pathParts[1]
	logger.Info("CancelRideHandler", "request to cancel ride: "+rideID)

	passengerID, ok := r.Context().Value("passenger_id").(string)
	if !ok || passengerID == "" {
		logger.Warn("CancelRideHandler", "unauthorized request: missing passenger_id")
		util.WriteJSONError(w, "unauthorized: missing passenger_id", http.StatusUnauthorized)
		return
	}

	role, _ := r.Context().Value("role").(string)
	if role != "PASSENGER" {
		logger.Warn("CancelRideHandler", "forbidden: non-passenger tried to cancel ride")
		util.WriteJSONError(w, "forbidden: only passengers can cancel rides", http.StatusForbidden)
	}

	var body struct {
		Reason string `json:"reason"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		logger.Error("CancelRideHandler", err)
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(body.Reason) == "" {
		body.Reason = "Cancelled by passenger"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	logger.Info("CancelRideHandler", "processing cancellation for ride "+rideID)
	refundPercent, err := h.service.CancelRide(ctx, rideID, passengerID, body.Reason)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			logger.Warn("CancelRideHandler", "ride not found: "+rideID)
			util.WriteJSONError(w, "ride not found", http.StatusNotFound)
			return
		case errors.Is(err, domain.ErrForbidden):
			logger.Warn("CancelRideHandler", "forbidden cancellation attempt for ride: "+rideID)
			util.WriteJSONError(w, "you cannot cancel this ride", http.StatusForbidden)
			return
		case errors.Is(err, domain.ErrInvalidStatus):
			logger.Warn("CancelRideHandler", "ride cannot be cancelled at current stage: "+rideID)
			util.WriteJSONError(w, "ride cannot be cancelled at this stage", http.StatusConflict)
			return
		default:
			logger.Error("CancelRideHandler", err)
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

	logger.OK("CancelRideHandler", "ride cancelled successfully: "+rideID)
	logger.HTTP(http.StatusOK, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
}
