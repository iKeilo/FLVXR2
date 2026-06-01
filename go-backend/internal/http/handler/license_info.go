package handler

import (
	"net/http"
	"os"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/license"
	"go-backend/internal/middleware"
)

// licenseInfo returns the current license state
// This endpoint is called on page load/refresh
// It always triggers a background check to ensure status is up-to-date
func (h *Handler) licenseInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.WriteJSON(w, response.ErrDefault("请求失败"))
		return
	}

	// Always trigger background check on page refresh to get latest status
	// We use synchronous check here to ensure the state is updated *before* the handler returns.
	// This prevents a race condition where the user sees "Valid" but the LicenseGuard still thinks it's "Invalid"
	// immediately after a refresh.
	middleware.ForceSyncCheck()

	// Get refreshed license state
	valid, expireTime, reason, isTrial := middleware.GetLicenseState()

	// Check if license is configured
	// 1. Prioritize Database (Real-time Config)
	cfg1, _ := h.repo.GetConfigByName("license_server_url")
	cfg2, _ := h.repo.GetConfigByName("license_key")
	cfg3, _ := h.repo.GetConfigByName("server_domain")

	// Get values from DB
	var serverUrl, licenseKey, domain string
	if cfg1 != nil {
		serverUrl = cfg1.Value
	}
	if cfg2 != nil {
		licenseKey = cfg2.Value
	}
	if cfg3 != nil {
		domain = cfg3.Value
	}

	// 2. Fallback to Environment Variables (Startup Config / .env)
	if serverUrl == "" {
		serverUrl = os.Getenv("LICENSE_SERVER_URL")
	}
	if serverUrl == "" && licenseKey != "" {
		serverUrl = license.DefaultServerURL
	}
	if licenseKey == "" {
		licenseKey = os.Getenv("LICENSE_KEY")
	}
	if domain == "" {
		domain = os.Getenv("SERVER_DOMAIN")
	}

	configured := serverUrl != "" || licenseKey != ""

	hasLicenseKey := licenseKey != ""
	actualLicenseKey := licenseKey

	tier, _ := middleware.GetLicenseTier()

	hmacKey := os.Getenv("HMAC_SECRET_KEY")
	if hmacKey == "" {
		cfg, _ := h.repo.GetConfigByName("hmac_key")
		if cfg != nil {
			hmacKey = cfg.Value
		}
	}

	// Calculate trial remaining days
	trialRemainingDays := 0
	if isTrial && valid && expireTime > 0 {
		remaining := expireTime - time.Now().UnixMilli()
		if remaining > 0 {
			trialRemainingDays = int(remaining / 86400000)
		}
	}

	entitlements := middleware.GetCommercialEntitlements()
	commercialProfile := middleware.GetCommercialProfile()

	response.WriteJSON(w, response.OK(map[string]interface{}{
		"valid":                valid,
		"expire_time":          expireTime,
		"reason":               reason,
		"configured":           configured,
		"has_license_key":      hasLicenseKey,
		"license_key":          actualLicenseKey,
		"domain":               domain,
		"tier":                 string(tier),
		"hmac_key":             hmacKey,
		"is_trial":             isTrial,
		"trial_remaining_days": trialRemainingDays,
		"commercial_profile":   commercialProfile,
		"entitlements":         entitlements,
		"billing_allowed":      entitlements.BillingAllowed,
		"commercial_allowed":   entitlements.CommercialAllowed,
		"multi_tenant_allowed": entitlements.MultiTenantAllowed,
		"white_label_allowed":  entitlements.WhiteLabelAllowed,
		"resale_allowed":       entitlements.ResaleAllowed,
	}))
}
