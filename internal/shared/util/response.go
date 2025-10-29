package util

import (
	"encoding/json"
	"net/http"

	"ride-hail/internal/shared/apperrors"
)

func ResponseInJson(w http.ResponseWriter, statusCode int, object interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(object)
}

func ErrResponseInJson(w http.ResponseWriter, err error) {
	statusCode := apperrors.CheckError(err)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
