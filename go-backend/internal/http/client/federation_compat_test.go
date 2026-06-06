package client

import "testing"

func TestNormalizeRemoteNodeInfoLegacyProvider(t *testing.T) {
	info := &RemoteNodeInfo{}

	normalizeRemoteNodeInfo(info)

	if info.ProviderType != "sagit-chu-compatible" {
		t.Fatalf("provider type = %q, want sagit-chu-compatible", info.ProviderType)
	}
	if info.ProtocolVersion != "baseline" {
		t.Fatalf("protocol version = %q, want baseline", info.ProtocolVersion)
	}
	if len(info.Features) != len(baselineFederationFeatures) {
		t.Fatalf("features len = %d, want %d", len(info.Features), len(baselineFederationFeatures))
	}
	if len(info.RuntimeModes) != 1 || info.RuntimeModes[0] != "gost" {
		t.Fatalf("runtime modes = %#v, want [gost]", info.RuntimeModes)
	}
	if info.SupportsWGPath || info.SupportsTLSInbound || info.SupportsNftables {
		t.Fatalf("legacy provider must not imply extended capabilities: %#v", info)
	}
}

func TestNormalizeRemoteNodeInfoKeepsAdvertisedCapabilities(t *testing.T) {
	info := &RemoteNodeInfo{
		ProviderType:       "flvxt2",
		ProtocolVersion:    "flvxt2-v1",
		Features:           []string{"runtime_command"},
		RuntimeModes:       []string{"gost", "wireguard"},
		SupportsWGPath:     true,
		SupportsTLSInbound: true,
	}

	normalizeRemoteNodeInfo(info)

	if info.ProviderType != "flvxt2" || info.ProtocolVersion != "flvxt2-v1" {
		t.Fatalf("advertised identity was overwritten: %#v", info)
	}
	if len(info.Features) != 1 || info.Features[0] != "runtime_command" {
		t.Fatalf("features were overwritten: %#v", info.Features)
	}
	if len(info.RuntimeModes) != 2 || info.RuntimeModes[1] != "wireguard" {
		t.Fatalf("runtime modes were overwritten: %#v", info.RuntimeModes)
	}
	if !info.SupportsWGPath || !info.SupportsTLSInbound {
		t.Fatalf("advertised extended capabilities were lost: %#v", info)
	}
}
