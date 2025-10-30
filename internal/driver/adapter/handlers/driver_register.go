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

	err := json.NewDecoder(r.Body).Decode(&driverData)
	if err != nil {
		fmt.Println("cannot decode json body:", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	driverData.ID = r.PathValue("driver_id")

	statusCode, err := h.service.RegisterDriver(ctx, &driverData)
	if err != nil {
		fmt.Println("cannot register driver: ", err)
		http.Error(w, err.Error(), statusCode)
		return
	}

	util.ResponseInJson(w, 201, map[string]interface{}{
		"driver_id": driverData.ID,
		"message":   "You have successfully registered as driver",
	})
}
