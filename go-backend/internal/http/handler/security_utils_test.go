package handler

import "testing"

func TestIsSafeRemoteAddrRejectsRestrictedAddresses(t *testing.T) {
	cases := []string{
		"127.0.0.1:8080",
		"localhost:8080",
		"10.0.0.1:6365",
		"192.168.1.2",
		"[::1]:6365",
	}

	for _, tc := range cases {
		if err := IsSafeRemoteAddr(tc); err == nil {
			t.Fatalf("IsSafeRemoteAddr(%q) succeeded, want rejection", tc)
		}
	}
}

func TestIsSafeRemoteAddrAcceptsPublicAddress(t *testing.T) {
	if err := IsSafeRemoteAddr("1.1.1.1:443"); err != nil {
		t.Fatalf("IsSafeRemoteAddr(public) error = %v", err)
	}
}

func TestIsSafeRemoteAddrTestingBypass(t *testing.T) {
	old := DisableSafeRemoteAddrCheckForTesting
	DisableSafeRemoteAddrCheckForTesting = true
	defer func() { DisableSafeRemoteAddrCheckForTesting = old }()

	if err := IsSafeRemoteAddr("127.0.0.1:8080"); err != nil {
		t.Fatalf("testing bypass error = %v", err)
	}
}

func TestIsValidNodeAddressRejectsURLShapes(t *testing.T) {
	cases := []string{
		"https://example.com",
		"example.com/path",
		"example.com?token=1",
	}

	for _, tc := range cases {
		if err := IsValidNodeAddress(tc); err == nil {
			t.Fatalf("IsValidNodeAddress(%q) succeeded, want rejection", tc)
		}
	}
}
