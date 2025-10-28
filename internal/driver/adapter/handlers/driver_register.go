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

func (h *Handler) RegisterDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 3)) // I dont know whether we need context here?
	defer cancel()

	driverData := models.Driver{}
	driverData.ID = r.PathValue("user_id")

	err := json.NewDecoder(r.Body).Decode(&driverData)
	if err != nil {
		fmt.Printf("cannot decode json body: %v", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	statusCode, err := h.service.RegisterDriver(ctx, &driverData)
	if err != nil {
		fmt.Printf("cannot register driver: %v", err.Error())
		util.ErrResponseInJson(w, err, statusCode)
		return
	}

	util.ResponseInJson(w, http.StatusOK, map[string]interface{}{
		"driver_id": driverData.ID,
		"message":   "You have successfully registered as driver",
	})
}
