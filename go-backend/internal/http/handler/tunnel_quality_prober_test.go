package handler

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

func TestTunnelQualityProberUsesConfiguredProbeTarget(t *testing.T) {
	h := setupProbeTargetTunnelHandler(t)
	seedProbeTargetTunnel(t, h, 77, "quality-target", "speed.example.com", 8443)
	if err := h.repo.DB().Exec(`
		INSERT INTO node(id, name, secret, server_ip, server_ip_v4, server_ip_v6, port, interface_name, version, http, tls, socks, created_time, updated_time, status, tcp_listen_addr, udp_listen_addr, inx)
		VALUES(30, 'exit-a', 'exit-secret', '10.0.0.30', '10.0.0.30', '', '30000-30010', '', 'v1', 1, 1, 1, ?, ?, 1, '[::]', '[::]', 0)
	`, time.Now().UnixMilli(), time.Now().UnixMilli()).Error; err != nil {
		t.Fatalf("insert exit node: %v", err)
	}
	if err := h.repo.DB().Exec(`
		INSERT INTO chain_tunnel(tunnel_id, chain_type, node_id, port, strategy, inx, protocol)
		VALUES(77, '3', 30, 30001, 'round', 1, 'tls')
	`).Error; err != nil {
		t.Fatalf("insert exit chain: %v", err)
	}

	p := newTunnelQualityProber(h)
	var calls []string
	p.probeNode = func(nodeID int64, ip string, port int, options diagnosisExecOptions) (float64, float64, error) {
		calls = append(calls, fmt.Sprintf("%d|%s|%d", nodeID, ip, port))
		return 10, 0, nil
	}
	p.probeTunnel(77)

	if !slices.Contains(calls, "10|speed.example.com|8443") {
		t.Fatalf("expected type 1 public probe from entry to configured target, calls=%+v", calls)
	}
	if slices.Contains(calls, "30|speed.example.com|8443") {
		t.Fatalf("did not expect type 1 public probe from exit node, calls=%+v", calls)
	}
	snaps := p.GetAll()
	if len(snaps) != 1 {
		t.Fatalf("expected one quality snapshot, got %+v", snaps)
	}
	if snaps[0].ProbeTargetHost != "speed.example.com" || snaps[0].ProbeTargetPort != 8443 {
		t.Fatalf("unexpected snapshot target metadata: %+v", snaps[0])
	}
}

func TestTunnelQualityProberStoresProbeTargetWhenChainIncomplete(t *testing.T) {
	h := setupProbeTargetTunnelHandler(t)
	seedProbeTargetTunnel(t, h, 78, "quality-target-incomplete", "speed.example.com", 8443)
	if err := h.repo.DB().Exec(`DELETE FROM chain_tunnel WHERE tunnel_id = ?`, 78).Error; err != nil {
		t.Fatalf("delete chain rows: %v", err)
	}

	p := newTunnelQualityProber(h)
	p.probeTunnel(78)

	snaps := p.GetAll()
	if len(snaps) != 1 {
		t.Fatalf("expected one quality snapshot, got %+v", snaps)
	}
	if snaps[0].ErrorMessage == "" {
		t.Fatalf("expected incomplete chain error, got %+v", snaps[0])
	}
	if snaps[0].ProbeTargetHost != "speed.example.com" || snaps[0].ProbeTargetPort != 8443 {
		t.Fatalf("unexpected snapshot target metadata: %+v", snaps[0])
	}
}
