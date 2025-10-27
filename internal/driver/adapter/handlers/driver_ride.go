package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"ride-hail/internal/driver/models"
	"ride-hail/internal/shared/util"
	"time"
)

func (h *Handler) CurrLocationDriver(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), (time.Second * 30))
	defer cancel()

	location := models.LocalHistory{}

	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		slog.Error("error:", err)
		return
	}

	location.DriverID = r.PathValue("driver_id")

	result, err := h.service.UpdateLocation(ctx, &location)
	if err != nil {
		slog.Error("error:", err)

		util.ErrResponseInJson(w, err)

		return
	}

	util.ResponseInJson(w, 200, result)
}
