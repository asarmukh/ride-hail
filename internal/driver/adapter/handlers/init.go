package handlers

import "net/http"

type Handler struct {
}

func (h *Handler) NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) Router() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /drivers/{driver_id}/online", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /drivers/{driver_id}/offline", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /drivers/{driver_id}/location", func(w http.ResponseWriter, r *http.Request) {})

	return mux
}
