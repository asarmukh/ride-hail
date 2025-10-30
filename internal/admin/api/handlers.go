package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"ride-hail/internal/admin/app"
)

type Handler struct {
	service *app.AdminService
}

func NewHandler(service *app.AdminService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetSystemOverview(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	overview, err := h.service.GetSystemOverview(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch system overview", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func (h *Handler) GetActiveRides(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse pagination parameters
	pageStr := r.URL.Query().Get("page")
	pageSizeStr := r.URL.Query().Get("page_size")

	page := 1
	pageSize := 20

	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	response, err := h.service.GetActiveRides(ctx, page, pageSize)
	if err != nil {
		http.Error(w, "Failed to fetch active rides", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
