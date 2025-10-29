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

	id := r.PathValue("driver_id")
	fmt.Println("StartDriver called", id)

	location := models.Location{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		fmt.Printf("error: %s ", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}
	location.DriverID = id

	id, err = h.service.StartSession(ctx, location, id)
	if err != nil {
		fmt.Printf("error: %s ", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	util.ResponseInJson(w, 200, map[string]interface{}{
		"status":     "AVAILABLE",
		"session_id": id,
		"message":    "You are now online and ready to accept rides",
	})
}

func (h *Handler) FinishDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	id := r.PathValue("driver_id")

	response, err := h.service.FinishSession(ctx, id)
	if err != nil {
		fmt.Printf("error: %s ", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	util.ResponseInJson(w, 200, response)
}
