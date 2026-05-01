package repo

import (
	"path/filepath"
	"testing"
	"time"
)

func TestBackupRoundTripsTunnelProbeTarget(t *testing.T) {
	source, err := Open(filepath.Join(t.TempDir(), "source.db"))
	if err != nil {
		t.Fatalf("open source repo: %v", err)
	}
	defer source.Close()

	now := time.Now().UnixMilli()
	if err := source.DB().Exec(`
		INSERT INTO tunnel(id, name, traffic_ratio, type, protocol, flow, created_time, updated_time, status, in_ip, inx, probe_target_host, probe_target_port)
		VALUES(20, 'backup-target', 1, 2, 'tls', 1, ?, ?, 1, '', 1, 'speed.example.com', 8443)
	`, now, now).Error; err != nil {
		t.Fatalf("insert source tunnel: %v", err)
	}

	backup, err := source.ExportAll()
	if err != nil {
		t.Fatalf("export backup: %v", err)
	}
	if len(backup.Tunnels) != 1 {
		t.Fatalf("expected one exported tunnel, got %d", len(backup.Tunnels))
	}
	if backup.Tunnels[0].ProbeTargetHost != "speed.example.com" || backup.Tunnels[0].ProbeTargetPort != 8443 {
		t.Fatalf("unexpected exported probe target: %+v", backup.Tunnels[0])
	}

	dest, err := Open(filepath.Join(t.TempDir(), "dest.db"))
	if err != nil {
		t.Fatalf("open dest repo: %v", err)
	}
	defer dest.Close()

	result, err := dest.Import(backup, []string{"tunnels"})
	if err != nil {
		t.Fatalf("import backup: %v", err)
	}
	if result.TunnelsImported != 1 {
		t.Fatalf("expected one imported tunnel, got %d", result.TunnelsImported)
	}

	items, err := dest.ListTunnels()
	if err != nil {
		t.Fatalf("list imported tunnels: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one imported tunnel item, got %d", len(items))
	}
	if items[0]["probeTargetHost"] != "speed.example.com" || items[0]["probeTargetPort"] != 8443 {
		t.Fatalf("unexpected imported probe target: %+v", items[0])
	}
}
