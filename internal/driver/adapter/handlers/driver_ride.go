package handlers

import (
	"context"
	"encoding/json"
	"fmt"
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

	var requestBody struct {
		RideID         string          `json:"ride_id"`
		DriverLocation models.Location `json:"driver_location"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	statusCode, err := h.service.StartRide(ctx, requestBody.RideID, driverID, requestBody.DriverLocation)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		util.ErrResponseInJson(w, err, statusCode)
		return
	}

	var responseBody struct {
		RideID    string `json:"ride_id"`
		Status    string `json:"status"`
		StartedAt string `json:"started_at"`
		Message   string `json:"message"`
	}

	responseBody.RideID = requestBody.RideID
	responseBody.Status = "BUSY"
	responseBody.StartedAt = time.Now().Format(time.RFC3339)
	responseBody.Message = "Ride started successfully"
	util.ResponseInJson(w, http.StatusOK, responseBody)
}


//----------------------------------------------------------------------------------------------------
func (h *Handler) CompleteRide(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	driverID := r.PathValue("driver_id")

	var requestBody struct {
		ID                    string          `json:"ride_id"`
		FinalLocation         models.Location `json:"final_location"`
		ActualDistanceKM      float64         `json:"actual_distance_km"`
		ActualDurationMinutes int             `json:"actual_duration_minutes"`
	}

	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	driverEarnings, statusCode, err := h.service.CompleteRide(
		ctx,
		driverID,
		requestBody.ID,
		requestBody.FinalLocation,
		requestBody.ActualDistanceKM,
		requestBody.ActualDurationMinutes,
	)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		util.ErrResponseInJson(w, err, statusCode)
		return
	}

	var responseBody struct {
		RideID         string  `json:"ride_id"`
		Status         string  `json:"status"`
		CompletedAt    string  `json:"completed_at"`
		DriverEarnings float64 `json:"driver_earnings"`
		Message        string  `json:"message"`
	}

	responseBody.RideID = requestBody.ID
	responseBody.Status = "BUSY"
	responseBody.CompletedAt = time.Now().Format(time.RFC3339)
	responseBody.DriverEarnings = driverEarnings
	responseBody.Message = "Ride started successfully"
	util.ResponseInJson(w, http.StatusOK, responseBody)
}
