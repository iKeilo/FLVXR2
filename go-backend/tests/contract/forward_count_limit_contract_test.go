package contract_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-backend/internal/auth"
	"go-backend/internal/http/response"
	"go-backend/internal/store/repo"
)

func TestForwardCreateAllowedWhenUserForwardCountWouldHaveExceededLegacyLimit(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	server := httptest.NewServer(router)
	defer server.Close()

	now := time.Now().UnixMilli()
	userID := int64(100)
	tunnelID, nodeSecret := seedForwardCountLimitTunnelFixture(t, repo, now, "legacy-user-limit")
	stopNode := startMockNodeSession(t, server.URL, nodeSecret)
	defer stopNode()

	seedForwardCountLimitUserFixture(t, repo, userID, tunnelID, now, 2, 99999)
	seedForwardRecord(t, repo, 1, userID, tunnelID, "legacy-user-forward-1", "8.8.8.8:53", now, 1)
	seedForwardRecord(t, repo, 2, userID, tunnelID, "legacy-user-forward-2", "8.8.4.4:53", now, 1)

	token, err := auth.GenerateToken(userID, userFixtureName(userID), 1, secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	res := performForwardCreate(t, router, token, tunnelID, "legacy-user-forward-3", "1.1.1.1:53")

	assertResponseCode(t, res, 0)
}

func TestForwardResumeAllowedWhenUserForwardCountWouldHaveExceededLegacyLimit(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	server := httptest.NewServer(router)
	defer server.Close()

	now := time.Now().UnixMilli()
	userID := int64(101)
	tunnelID, nodeSecret := seedForwardCountLimitTunnelFixture(t, repo, now, "legacy-resume-limit")
	stopNode := startMockNodeSession(t, server.URL, nodeSecret)
	defer stopNode()

	seedForwardCountLimitUserFixture(t, repo, userID, tunnelID, now, 2, 99999)

	token, err := auth.GenerateToken(userID, userFixtureName(userID), 1, secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	createRes := performForwardCreate(t, router, token, tunnelID, "resume-target", "1.1.1.1:53")
	assertResponseCode(t, createRes, 0)
	pausedForwardID := mustLastInsertID(t, repo, "resume-target")

	pauseReq := httptest.NewRequest(http.MethodPost, "/api/v1/forward/pause", bytes.NewBufferString(fmt.Sprintf(`{"id":%d}`, pausedForwardID)))
	pauseReq.Header.Set("Authorization", token)
	pauseReq.Header.Set("Content-Type", "application/json")
	pauseRes := httptest.NewRecorder()
	router.ServeHTTP(pauseRes, pauseReq)
	assertResponseCode(t, pauseRes, 0)

	seedForwardRecord(t, repo, 2001, userID, tunnelID, "legacy-resume-forward-1", "8.8.8.8:53", now, 1)
	seedForwardRecord(t, repo, 2002, userID, tunnelID, "legacy-resume-forward-2", "8.8.4.4:53", now, 1)

	resumeReq := httptest.NewRequest(http.MethodPost, "/api/v1/forward/resume", bytes.NewBufferString(fmt.Sprintf(`{"id":%d}`, pausedForwardID)))
	resumeReq.Header.Set("Authorization", token)
	resumeReq.Header.Set("Content-Type", "application/json")
	resumeRes := httptest.NewRecorder()
	router.ServeHTTP(resumeRes, resumeReq)

	assertResponseCode(t, resumeRes, 0)

	status := mustQueryInt(t, repo, `SELECT status FROM forward WHERE id = ?`, pausedForwardID)
	if status != 1 {
		t.Fatalf("expected forward status to become 1 after resume, got %d", status)
	}
}

func TestForwardCreateAllowedWhenTunnelForwardCountWouldHaveExceededLegacyLimit(t *testing.T) {
	secret := "contract-jwt-secret"
	router, repo := setupContractRouter(t, secret)
	server := httptest.NewServer(router)
	defer server.Close()

	now := time.Now().UnixMilli()
	userID := int64(102)
	tunnelID, nodeSecret := seedForwardCountLimitTunnelFixture(t, repo, now, "legacy-tunnel-limit")
	stopNode := startMockNodeSession(t, server.URL, nodeSecret)
	defer stopNode()

	seedForwardCountLimitUserFixture(t, repo, userID, tunnelID, now, 99999, 1)
	seedForwardRecord(t, repo, 3001, userID, tunnelID, "legacy-tunnel-forward-1", "8.8.8.8:53", now, 1)

	token, err := auth.GenerateToken(userID, userFixtureName(userID), 1, secret)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	res := performForwardCreate(t, router, token, tunnelID, "legacy-tunnel-forward-2", "1.1.1.1:53")

	assertResponseCode(t, res, 0)
}

func performForwardCreate(t *testing.T, router http.Handler, token string, tunnelID int64, name string, remoteAddr string) *httptest.ResponseRecorder {
	t.Helper()
	payload := fmt.Sprintf(`{"tunnelId":%d,"name":"%s","remoteAddr":"%s","strategy":"fifo"}`, tunnelID, name, remoteAddr)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/forward/create", bytes.NewBufferString(payload))
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func seedForwardCountLimitTunnelFixture(t *testing.T, repo *repo.Repository, now int64, prefix string) (int64, string) {
	t.Helper()
	secret := prefix + "-secret"
	nodeName := prefix + "-node"
	tunnelName := prefix + "-tunnel"

	if err := repo.DB().Exec(`
		INSERT INTO tunnel(id, name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx)
		VALUES(1, ?, 1.0, 1, 'tls', 99999, ?, ?, 1, NULL, 0)
	`, tunnelName, now, now).Error; err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES(?, ?, '10.50.0.1', '10.50.0.1', '', '10000-10010', '', 'v1', 1, 1, 1, ?, ?, 1, '[::]', '[::]', 0)
	`, nodeName, secret, now, now).Error; err != nil {
		t.Fatalf("insert node: %v", err)
	}
	nodeID := mustLastInsertID(t, repo, nodeName)

	if err := repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(1, 1, ?, 10001, 'round', 1, 'tls')
	`, nodeID).Error; err != nil {
		t.Fatalf("insert chain_tunnel: %v", err)
	}

	return 1, secret
}

func seedForwardCountLimitUserFixture(t *testing.T, repo *repo.Repository, userID int64, tunnelID int64, now int64, userNum int, tunnelNum int) {
	t.Helper()
	if err := repo.DB().Exec(`
		INSERT INTO user(id, user, pwd, role_id, exp_time, flow, in_flow, out_flow, flow_reset_time, num, created_time, updated_time, status)
		VALUES(?, ?, 'pwd', 1, 2727251700000, 99999, 0, 0, 1, ?, ?, ?, 1)
	`, userID, userFixtureName(userID), userNum, now, now).Error; err != nil {
		t.Fatalf("insert user: %v", err)
	}

	if err := repo.DB().Exec(`
		INSERT INTO user_tunnel(id, user_id, tunnel_id, speed_id, num, flow, in_flow, out_flow, flow_reset_time, exp_time, status)
		VALUES(?, ?, ?, NULL, ?, 99999, 0, 0, 1, 2727251700000, 1)
	`, userID+1000, userID, tunnelID, tunnelNum).Error; err != nil {
		t.Fatalf("insert user_tunnel: %v", err)
	}
}

func seedForwardRecord(t *testing.T, repo *repo.Repository, forwardID int64, userID int64, tunnelID int64, name string, remoteAddr string, now int64, status int) {
	t.Helper()
	if err := repo.DB().Exec(`
		INSERT INTO forward(id, user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx)
		VALUES(?, ?, ?, ?, ?, ?, 'fifo', 0, 0, ?, ?, ?, 0)
	`, forwardID, userID, userFixtureName(userID), name, tunnelID, remoteAddr, now, now, status).Error; err != nil {
		t.Fatalf("insert forward %s: %v", name, err)
	}
}

func userFixtureName(userID int64) string {
	return fmt.Sprintf("user_%d", userID)
}

func assertResponseCode(t *testing.T, res *httptest.ResponseRecorder, expected int) {
	t.Helper()
	var out response.R
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Code != expected {
		t.Fatalf("expected code=%d, got code=%d msg=%q", expected, out.Code, out.Msg)
	}
}
