package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"go-backend/internal/auth"
	"go-backend/internal/http/middleware"
	"go-backend/internal/http/response"
	"go-backend/internal/store/repo"
)

func TestParseRemoteProviderConfigCapabilities(t *testing.T) {
	info := parseRemoteProviderConfig(`{
		"providerType": "flvxt2",
		"protocolVersion": "flvxt2-v1",
		"features": ["runtime_command", "wg_path"],
		"runtimeModes": ["gost", "wireguard"],
		"supportsNftables": "1",
		"supportsWGPath": true,
		"supportsTLSInbound": "true"
	}`)

	if info.ProviderType != "flvxt2" {
		t.Fatalf("provider type = %q", info.ProviderType)
	}
	if info.ProtocolVersion != "flvxt2-v1" {
		t.Fatalf("protocol version = %q", info.ProtocolVersion)
	}
	if len(info.Features) != 2 || info.Features[1] != "wg_path" {
		t.Fatalf("features = %#v", info.Features)
	}
	if len(info.RuntimeModes) != 2 || info.RuntimeModes[1] != "wireguard" {
		t.Fatalf("runtime modes = %#v", info.RuntimeModes)
	}
	if !info.SupportsNftables || !info.SupportsWGPath || !info.SupportsTLSInbound {
		t.Fatalf("support flags were not parsed: %#v", info)
	}
}

func TestParseRemoteProviderConfigLegacyEmpty(t *testing.T) {
	info := parseRemoteProviderConfig(`{"shareId":1,"portRangeStart":30000}`)

	if info.ProviderType != "" || info.ProtocolVersion != "" {
		t.Fatalf("legacy config should not invent provider info: %#v", info)
	}
	if len(info.Features) != 0 || len(info.RuntimeModes) != 0 {
		t.Fatalf("legacy config should not invent capabilities: %#v", info)
	}
	if info.SupportsNftables || info.SupportsWGPath || info.SupportsTLSInbound {
		t.Fatalf("legacy config should not imply extended capabilities: %#v", info)
	}
}

func TestNodeImportRequiresAdmin(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	body, _ := json.Marshal(nodeImportRequest{
		RemoteURL: "https://example.com",
		Token:     "share-token",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/node/import", bytes.NewReader(body))
	req = req.WithContext(context.WithValue(req.Context(), middleware.ClaimsContextKey, auth.Claims{
		Sub:    "2",
		User:   "normal_user",
		Name:   "normal_user",
		RoleID: 1,
	}))
	res := httptest.NewRecorder()

	h.nodeImport(res, req)

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != 403 {
		t.Fatalf("expected 403 for non-admin import, got code=%d msg=%q", payload.Code, payload.Msg)
	}
}

func TestNodeImportRejectsRestrictedRemoteURL(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	body, _ := json.Marshal(nodeImportRequest{
		RemoteURL: "http://127.0.0.1:6365",
		Token:     "share-token",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/node/import", bytes.NewReader(body))
	setFederationShareAdminAuth(t, req)
	res := httptest.NewRecorder()

	h.nodeImport(res, req)

	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != -1 || !strings.Contains(payload.Msg, "Remote URL is not allowed") {
		t.Fatalf("expected restricted URL rejection, got code=%d msg=%q", payload.Code, payload.Msg)
	}
}
