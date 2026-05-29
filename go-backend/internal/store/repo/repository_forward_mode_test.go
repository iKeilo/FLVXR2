package repo

import (
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateForwardModePersistsNftables(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "forward-mode.db"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	now := time.Now().UnixMilli()
	if err := r.DB().Exec(`
		INSERT INTO forward(
			id, user_id, user_name, name, tunnel_id, remote_addr, strategy,
			in_flow, out_flow, created_time, updated_time, status, inx,
			speed_id, max_conn, ip_max_conn, ip_speed_id, proxy_protocol, mode
		)
		VALUES(1, 1, 'user', 'forward-1', 1, '127.0.0.1:9000', 'fifo',
			0, 0, ?, ?, 1, 0,
			NULL, 0, 0, NULL, 0, 'gost')
	`, now, now).Error; err != nil {
		t.Fatalf("insert forward: %v", err)
	}

	if err := r.UpdateForwardMode(1, "nftables", now+1); err != nil {
		t.Fatalf("update forward mode: %v", err)
	}

	record, err := r.GetForwardRecord(1)
	if err != nil {
		t.Fatalf("get forward: %v", err)
	}
	if record == nil {
		t.Fatalf("expected forward record")
	}
	if record.Mode != "nftables" {
		t.Fatalf("expected nftables mode, got %q", record.Mode)
	}
}
