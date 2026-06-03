package middleware

import (
	"net/http"

	"go-backend/internal/http/response"
	"go-backend/internal/middleware"
	"go-backend/internal/store/repo"
)

// TrialGuard keeps legacy resource routes compatible with the commercial
// license model. Basic deployments are not capped by node/tunnel/user count;
// only a blocked license state prevents resource mutation.
func TrialGuard(next http.Handler, r *repo.Repository) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if r == nil {
			next.ServeHTTP(w, req)
			return
		}

		tier, reason := middleware.GetLicenseTier()
		if tier == middleware.TierBlocked {
			response.WriteJSON(w, response.Err(403, "授权无效，无法操作："+reason))
			return
		}

		next.ServeHTTP(w, req)
	})
}
