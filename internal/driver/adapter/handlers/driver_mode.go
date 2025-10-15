package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
	"time"
)

func (h *Handler) StartDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	id := r.PathValue("driver_id")
	fmt.Println("StartDriver called", id)

	location := models.Location{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		slog.Error("error:", err)
		return
	}
	location.DriverID = id

	id, err = h.service.StartSession(ctx, location)
	if err != nil {
		slog.Error("error:", err)

		util.ErrResponseInJson(w, err)
		return
	}

	util.ResponseInJson(w, 200, map[string]interface{}{
		"status":     models.DriverAvailable,
		"session_id": id,
		"message":    "You are now online and ready to accept rides",
	})
}

func (h *Handler) FinishDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	id := r.PathValue("driver_id")

	err := h.service.FinishSession(ctx, id)
	if err != nil {
		slog.Error("error:", err)

		util.ErrResponseInJson(w, err)
		return
	}

	util.ResponseInJson(w, 200, map[string]interface{}{
		"status":     models.DriverAvailable,
		"session_id": id,
		"message":    "You are now online and ready to accept rides",
	})
}
