package api

import (
	"context"
	"encoding/json"
	"net/http"
	"ride-hail/internal/ride/domain"
	"ride-hail/internal/shared/util"
	"time"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	logger := util.New()
	start := time.Now()

	logger.Info("RegisterHandler", "incoming register request")

	var req domain.RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		logger.Error("RegisterHandler", err)
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		logger.HTTP(http.StatusBadRequest, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
		return
	}

	ctx := context.Background()
	user, err := h.service.Register(ctx, req.Email, req.Password, req.Role, req.Name, req.Phone)
	if err != nil {
		logger.Error("RegisterHandler", err)
		util.WriteJSONError(w, err.Error(), http.StatusConflict)
		logger.HTTP(http.StatusConflict, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
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

	logger.OK("RegisterHandler", "user registered successfully: "+user.ID)
	logger.HTTP(http.StatusCreated, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	logger := util.New()
	start := time.Now()

	logger.Info("LoginHandler", "incoming login request")

	var req domain.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		logger.Error("LoginHandler", err)
		util.WriteJSONError(w, "invalid JSON body", http.StatusBadRequest)
		logger.HTTP(http.StatusBadRequest, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
		return
	}

	ctx := context.Background()
	token, user, err := h.service.Login(ctx, req.Email, req.Password)
	if err != nil {
		logger.Error("LoginHandler", err)
		util.WriteJSONError(w, err.Error(), http.StatusUnauthorized)
		logger.HTTP(http.StatusUnauthorized, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
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

	logger.OK("LoginHandler", "user logged in successfully: "+user.ID)
	logger.HTTP(http.StatusOK, time.Since(start), r.RemoteAddr, r.Method, r.URL.Path)
}
