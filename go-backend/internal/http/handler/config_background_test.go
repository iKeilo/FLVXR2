package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go-backend/internal/auth"
	"go-backend/internal/http/middleware"
	"go-backend/internal/http/response"
	"go-backend/internal/store/repo"
)

func TestConfigBackgroundReadUsesSessionCookieWhenAuthMiddlewareIsSkipped(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer r.Close()

	const secret = "test-jwt-secret"
	h := New(r, secret, "3.0.16")
	now := time.Now().UnixMilli()
	if err := r.UpsertConfig(configKeyGlobalAppBackground, "global-bg", now); err != nil {
		t.Fatalf("upsert global background: %v", err)
	}
	if err := r.UpsertUserSetting(7, configKeyAppBackground, "personal-bg", now); err != nil {
		t.Fatalf("upsert user background: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/get", bytes.NewBufferString(`{"name":"app_bg_image"}`))
	token, err := auth.GenerateToken(7, "admin", 0, secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	req.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	res := httptest.NewRecorder()
	h.getConfigByName(res, req)

	payload := decodeConfigBackgroundResponse(t, res)
	data, ok := payload.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected data type: %T", payload.Data)
	}
	if got := data["value"]; got != "personal-bg" {
		t.Fatalf("expected personal background, got %#v", got)
	}

	listReq := httptest.NewRequest(http.MethodPost, "/api/v1/config/list", nil)
	listReq.AddCookie(&http.Cookie{Name: middleware.SessionCookieName, Value: token})
	listRes := httptest.NewRecorder()
	h.getConfigs(listRes, listReq)
	listPayload := decodeConfigBackgroundResponse(t, listRes)
	cfgMap, ok := listPayload.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected list data type: %T", listPayload.Data)
	}
	if got := cfgMap[configKeyAppBackground]; got != "personal-bg" {
		t.Fatalf("expected personal background in list, got %#v", got)
	}
}

func TestConfigBackgroundReadFallsBackToGlobalForGuest(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer r.Close()

	h := New(r, "test-jwt-secret", "3.0.16")
	if err := r.UpsertConfig(configKeyGlobalAppBackground, "global-bg", time.Now().UnixMilli()); err != nil {
		t.Fatalf("upsert global background: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/config/get", bytes.NewBufferString(`{"name":"app_bg_image"}`))
	res := httptest.NewRecorder()
	h.getConfigByName(res, req)

	payload := decodeConfigBackgroundResponse(t, res)
	data, ok := payload.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected data type: %T", payload.Data)
	}
	if got := data["value"]; got != "global-bg" {
		t.Fatalf("expected global background, got %#v", got)
	}
}

func decodeConfigBackgroundResponse(t *testing.T, res *httptest.ResponseRecorder) response.R {
	t.Helper()
	if res.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d body=%s", res.Code, res.Body.String())
	}
	var payload response.R
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v body=%s", err, res.Body.String())
	}
	if payload.Code != 0 {
		t.Fatalf("unexpected response code: %d msg=%s", payload.Code, payload.Msg)
	}
	return payload
}
