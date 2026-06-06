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

func isCommercialConfigKey(key string) bool {
	switch key {
	case "app_name", "app_logo", "app_favicon", "app_bg_image", "payment_enabled", "registration_enabled", "login_monitor_link":
		return true
	default:
		return false
	}
}
