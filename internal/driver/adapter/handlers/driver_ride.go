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

	location := models.LocalHistory{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	location.DriverID = r.PathValue("driver_id")

	result, err := h.service.UpdateLocation(ctx, &location)
	if err != nil {
		fmt.Printf("error: %s ", err.Error())
		util.ErrResponseInJson(w, err, http.StatusBadGateway)
		return
	}

	util.ResponseInJson(w, 200, result)
}

func (h *Handler) StartRide(w http.ResponseWriter, r *http.Request) {
	// ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	// defer cancel()

	// location := models.LocalHistory{}

	// err := json.NewDecoder(r.Body).Decode(&location)
	// if err != nil {
	// 	fmt.Printf("error: %s", err.Error())
	// 	util.ErrResponseInJson(w, err, http.StatusBadGateway)
	// 	return
	// }

	// location.DriverID = r.PathValue("driver_id")

	// result, err := h.service.UpdateLocation(ctx, &location)
	// if err != nil {
	// 	fmt.Printf("error: %s", err.Error())
	// 	util.ErrResponseInJson(w, err, http.StatusBadGateway)
	// 	return
	// }

	// util.ResponseInJson(w, 200, result)
}
