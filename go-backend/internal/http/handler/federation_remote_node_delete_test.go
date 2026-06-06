package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/store/model"
	"go-backend/internal/store/repo"
)

func insertFederationDeleteTestNode(t *testing.T, r *repo.Repository, name string, isRemote int) int64 {
	t.Helper()
	now := time.Now().UnixMilli()
	if err := r.DB().Exec(`
		INSERT INTO node(name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx, is_remote, remote_url, remote_token, remote_config)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, name, name+"-secret", "10.70.80.90", "10.70.80.90", "", "34000-34010", "", "v1", 1, 1, 1, now, now, 1, "[::]", "[::]", 0, isRemote, "http://peer.example", "peer-token", `{"shareId":123}`).Error; err != nil {
		t.Fatalf("insert node: %v", err)
	}
	return mustLastInsertID(t, r, name)
}

func deleteRemoteNodeForTest(t *testing.T, h *Handler, nodeID int64) response.R {
	t.Helper()
	reqBody, _ := json.Marshal(map[string]interface{}{"nodeId": nodeID})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/federation/node/delete", bytes.NewReader(reqBody))
	setFederationShareAdminAuth(t, req)
	res := httptest.NewRecorder()
	h.federationRemoteNodeDelete(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, res.Code)
	}
	var payload response.R
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

func TestFederationRemoteNodeDelete(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel-delete-remote-node.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	nodeID := insertFederationDeleteTestNode(t, r, "delete-remote-node", 1)

	payload := deleteRemoteNodeForTest(t, h, nodeID)
	if payload.Code != 0 {
		t.Fatalf("expected response code 0, got %d (%s)", payload.Code, payload.Msg)
	}

	var cnt int64
	if err := r.DB().Model(&model.Node{}).Where("id = ?", nodeID).Count(&cnt).Error; err != nil {
		t.Fatalf("count node: %v", err)
	}
	if cnt != 0 {
		t.Fatalf("expected remote node deleted, count=%d", cnt)
	}
}

func TestFederationRemoteNodeDeleteRejectsLocalNode(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel-delete-local-node.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	nodeID := insertFederationDeleteTestNode(t, r, "delete-local-node", 0)

	payload := deleteRemoteNodeForTest(t, h, nodeID)
	if payload.Code == 0 {
		t.Fatalf("expected local node deletion to be rejected")
	}
}

func TestFederationRemoteNodeDeleteRejectsInUseNode(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel-delete-used-remote-node.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	now := time.Now().UnixMilli()
	nodeID := insertFederationDeleteTestNode(t, r, "delete-used-remote-node", 1)

	if err := r.DB().Exec(`INSERT INTO tunnel(name, type, protocol, flow, created_time, updated_time, status, in_ip, inx) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, "delete-used-tunnel", 2, "tls", 0, now, now, 1, "", 0).Error; err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID := mustLastInsertID(t, r, "delete-used-tunnel")
	if err := r.DB().Exec(`INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol) VALUES(?, ?, ?, ?, ?, ?, ?)`, tunnelID, "1", nodeID, 34001, "fifo", 0, "tls").Error; err != nil {
		t.Fatalf("insert chain_tunnel: %v", err)
	}

	payload := deleteRemoteNodeForTest(t, h, nodeID)
	if payload.Code == 0 {
		t.Fatalf("expected in-use remote node deletion to be rejected")
	}
}
