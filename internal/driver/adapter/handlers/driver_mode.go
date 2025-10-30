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

func (h *Handler) StartDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	driverID := r.PathValue("driver_id")
	fmt.Println("StartDriver called", driverID)

	location := models.Location{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		fmt.Printf("failed to decode request body")
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}
	location.DriverID = driverID

	driverID, err = h.service.StartSession(ctx, driverID, location)
	if err != nil {
		fmt.Printf("error: %s ", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	util.ResponseInJson(w, 200, map[string]interface{}{
		"status":     "AVAILABLE",
		"session_id": driverID,
		"message":    "You are now online and ready to accept rides",
	})
}

func (h *Handler) FinishDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	id := r.PathValue("driver_id")

	err := h.service.FinishSession(ctx, id)
	if err != nil {
		// slog.Error("error:", err)

		// util.ErrResponseInJson(w, err)
		return
	}

	util.ResponseInJson(w, 200, map[string]interface{}{
		"status":     models.DriverAvailable,
		"session_id": id,
		"message":    "You are now online and ready to accept rides",
	})
}
