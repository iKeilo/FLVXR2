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

func TestFederationRemoteNodeDeleteCascadesEntryTunnelAndRules(t *testing.T) {
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
	if err := r.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx, max_connections, traffic_limit, speed_limit_enabled, speed_limit, upload_speed, download_speed, mode, wg_path_id, wg_rule_type, source_cidr, target_cidr, snat_enabled)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "admin", "delete-used-forward", tunnelID, "127.0.0.1:80", "fifo", 0, 0, now, now, 1, 0, 0, 0, false, 0, 0, 0, "gost", 0, "port", "", "", true).Error; err != nil {
		t.Fatalf("insert forward: %v", err)
	}
	forwardID := mustLastInsertID(t, r, "delete-used-forward")
	if err := r.DB().Exec(`INSERT INTO forward_port(forward_id, node_id, port) VALUES(?, ?, ?)`, forwardID, nodeID, 34002).Error; err != nil {
		t.Fatalf("insert forward_port: %v", err)
	}

	payload := deleteRemoteNodeForTest(t, h, nodeID)
	if payload.Code != 0 {
		t.Fatalf("expected in-use entry remote node deletion to succeed, got %d (%s)", payload.Code, payload.Msg)
	}

	var nodeCount, tunnelCount, forwardCount, chainCount, portCount int64
	if err := r.DB().Model(&model.Node{}).Where("id = ?", nodeID).Count(&nodeCount).Error; err != nil {
		t.Fatalf("count node: %v", err)
	}
	if err := r.DB().Model(&model.Tunnel{}).Where("id = ?", tunnelID).Count(&tunnelCount).Error; err != nil {
		t.Fatalf("count tunnel: %v", err)
	}
	if err := r.DB().Model(&model.Forward{}).Where("id = ?", forwardID).Count(&forwardCount).Error; err != nil {
		t.Fatalf("count forward: %v", err)
	}
	if err := r.DB().Model(&model.ChainTunnel{}).Where("tunnel_id = ?", tunnelID).Count(&chainCount).Error; err != nil {
		t.Fatalf("count chain: %v", err)
	}
	if err := r.DB().Model(&model.ForwardPort{}).Where("forward_id = ?", forwardID).Count(&portCount).Error; err != nil {
		t.Fatalf("count forward port: %v", err)
	}
	if nodeCount != 0 || tunnelCount != 0 || forwardCount != 0 || chainCount != 0 || portCount != 0 {
		t.Fatalf("expected cascaded delete, node=%d tunnel=%d forward=%d chain=%d port=%d", nodeCount, tunnelCount, forwardCount, chainCount, portCount)
	}
}

func TestFederationRemoteNodeDeleteKeepsMiddleTunnelAndRules(t *testing.T) {
	r, err := repo.Open(filepath.Join(t.TempDir(), "panel-delete-middle-remote-node.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	h := New(r, "test-jwt-secret", "3.0.16")
	now := time.Now().UnixMilli()
	nodeID := insertFederationDeleteTestNode(t, r, "delete-middle-remote-node", 1)
	entryNodeID := insertFederationDeleteTestNode(t, r, "delete-middle-entry-node", 0)
	exitNodeID := insertFederationDeleteTestNode(t, r, "delete-middle-exit-node", 0)

	if err := r.DB().Exec(`INSERT INTO tunnel(name, type, protocol, flow, created_time, updated_time, status, in_ip, inx) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, "keep-middle-tunnel", 2, "tls", 0, now, now, 1, "", 0).Error; err != nil {
		t.Fatalf("insert tunnel: %v", err)
	}
	tunnelID := mustLastInsertID(t, r, "keep-middle-tunnel")
	for _, row := range []struct {
		chainType string
		nodeID    int64
		inx       int
		port      int
	}{
		{"1", entryNodeID, 0, 35001},
		{"2", nodeID, 1, 35002},
		{"3", exitNodeID, 0, 35003},
	} {
		if err := r.DB().Exec(`INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol) VALUES(?, ?, ?, ?, ?, ?, ?)`, tunnelID, row.chainType, row.nodeID, row.port, "fifo", row.inx, "tls").Error; err != nil {
			t.Fatalf("insert chain_tunnel: %v", err)
		}
	}
	if err := r.DB().Exec(`
		INSERT INTO forward(user_id, user_name, name, tunnel_id, remote_addr, strategy, in_flow, out_flow, created_time, updated_time, status, inx, max_connections, traffic_limit, speed_limit_enabled, speed_limit, upload_speed, download_speed, mode, wg_path_id, wg_rule_type, source_cidr, target_cidr, snat_enabled)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, 1, "admin", "keep-middle-forward", tunnelID, "127.0.0.1:80", "fifo", 0, 0, now, now, 1, 0, 0, 0, false, 0, 0, 0, "gost", 0, "port", "", "", true).Error; err != nil {
		t.Fatalf("insert forward: %v", err)
	}
	forwardID := mustLastInsertID(t, r, "keep-middle-forward")
	if err := r.DB().Exec(`INSERT INTO forward_port(forward_id, node_id, port) VALUES(?, ?, ?)`, forwardID, entryNodeID, 35010).Error; err != nil {
		t.Fatalf("insert forward_port: %v", err)
	}

	payload := deleteRemoteNodeForTest(t, h, nodeID)
	if payload.Code != 0 {
		t.Fatalf("expected middle remote node deletion to succeed, got %d (%s)", payload.Code, payload.Msg)
	}

	var nodeCount, tunnelCount, forwardCount, middleChainCount, totalChainCount int64
	if err := r.DB().Model(&model.Node{}).Where("id = ?", nodeID).Count(&nodeCount).Error; err != nil {
		t.Fatalf("count node: %v", err)
	}
	if err := r.DB().Model(&model.Tunnel{}).Where("id = ?", tunnelID).Count(&tunnelCount).Error; err != nil {
		t.Fatalf("count tunnel: %v", err)
	}
	if err := r.DB().Model(&model.Forward{}).Where("id = ?", forwardID).Count(&forwardCount).Error; err != nil {
		t.Fatalf("count forward: %v", err)
	}
	if err := r.DB().Model(&model.ChainTunnel{}).Where("tunnel_id = ? AND node_id = ?", tunnelID, nodeID).Count(&middleChainCount).Error; err != nil {
		t.Fatalf("count middle chain: %v", err)
	}
	if err := r.DB().Model(&model.ChainTunnel{}).Where("tunnel_id = ?", tunnelID).Count(&totalChainCount).Error; err != nil {
		t.Fatalf("count chain: %v", err)
	}
	if nodeCount != 0 || tunnelCount != 1 || forwardCount != 1 || middleChainCount != 0 || totalChainCount != 2 {
		t.Fatalf("expected middle delete to keep tunnel/forward and remove only middle node, node=%d tunnel=%d forward=%d middleChain=%d totalChain=%d", nodeCount, tunnelCount, forwardCount, middleChainCount, totalChainCount)
	}
}
