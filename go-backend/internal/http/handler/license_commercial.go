package handler

import (
	"net/http"

	"go-backend/internal/http/response"
	"go-backend/internal/middleware"
)

func (h *Handler) ensureCommercialFeature(w http.ResponseWriter, feature string) bool {
	if err := middleware.CheckCommercialFeature(feature); err != nil {
		response.WriteJSON(w, response.Err(403, err.Error()))
		return false
	}
	return true
}
