package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
)

func (h *Handler) CurrLocationDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	driverID := r.PathValue("driver_id")

	// Check rate limiting
	if !h.service.CanUpdateLocation(driverID) {
		http.Error(w, "Rate limit exceeded. Max 1 update per 3 seconds", http.StatusTooManyRequests)
		return
	}

	location := models.LocalHistory{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		// slog.Error("error:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	location.DriverID = driverID

	result, err := h.service.UpdateLocation(ctx, &location)
	if err != nil {
		// slog.Error("error:", err)

		util.ErrResponseInJson(w, err, http.StatusBadGateway)

		return
	}

	util.ResponseInJson(w, 200, result)
}

func (h *Handler) StartRide(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	driverID := r.PathValue("driver_id")

	var req struct {
		RideID         string `json:"ride_id"`
		DriverLocation struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"driver_location"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// slog.Error("error decoding request:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Start the ride
	if err := h.service.StartRide(ctx, req.RideID, driverID, req.DriverLocation.Latitude, req.DriverLocation.Longitude); err != nil {
		// slog.Error("error starting ride:", err)
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	response := map[string]interface{}{
		"ride_id": req.RideID,
		"status":  "IN_PROGRESS",
		"message": "Ride started successfully",
	}

	util.ResponseInJson(w, 200, response)
}

func (h *Handler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	driverID := r.PathValue("driver_id")

	var req struct {
		RideID        string `json:"ride_id"`
		FinalLocation struct {
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"final_location"`
		ActualDistanceKm  float64 `json:"actual_distance_km"`
		ActualDurationMin int     `json:"actual_duration_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// slog.Error("error decoding request:", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Complete the ride
	finalFare, err := h.service.CompleteRide(ctx, req.RideID, driverID, req.FinalLocation.Latitude, req.FinalLocation.Longitude, req.ActualDistanceKm, req.ActualDurationMin)
	if err != nil {
		// slog.Error("error completing ride:", err)
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	response := map[string]interface{}{
		"ride_id":    req.RideID,
		"status":     "COMPLETED",
		"final_fare": finalFare,
		"message":    "Ride completed successfully",
	}

	util.ResponseInJson(w, 200, response)
}
