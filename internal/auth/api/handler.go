package api

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	user, err := h.service.Register(ctx, req.Email, req.Password, req.Role, req.Name, req.Phone)
	if err != nil {
		util.WriteJSONError(w, err.Error(), http.StatusConflict)
		return
	}

	resp := map[string]interface{}{
		"user_id": user.ID,
		"email":   user.Email,
		"role":    user.Role,
		"status":  user.Status,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	token, user, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		util.WriteJSONError(w, err.Error(), http.StatusUnauthorized)
		return
	}

	resp := map[string]interface{}{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
		"user": map[string]interface{}{
			"user_id": user.ID,
			"email":   user.Email,
			"role":    user.Role,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
