package handler

import (
	"fmt"
	"net"
	"strings"
)

// DisableSafeRemoteAddrCheckForTesting allows integration tests to use local
// httptest servers without weakening production SSRF checks.
var DisableSafeRemoteAddrCheckForTesting = false

// IsSafeRemoteAddr checks if a host or host:port resolves only to public IPs.
// It accepts comma/newline separated values for fields that store multiple
// targets, and rejects loopback/private/link-local/unspecified addresses.
func IsSafeRemoteAddr(addr string) error {
	if DisableSafeRemoteAddrCheckForTesting {
		return nil
	}

	for _, part := range splitRemoteParts(addr) {
		if err := checkSingleRemoteAddr(part); err != nil {
			return err
		}
	}
	return nil
}

func splitRemoteParts(addr string) []string {
	addr = strings.ReplaceAll(addr, "\n", ",")
	addr = strings.ReplaceAll(addr, "\r", ",")
	parts := strings.Split(addr, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func checkSingleRemoteAddr(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port in address") {
			host = addr
		} else {
			return fmt.Errorf("invalid address format: %v", err)
		}
	}
	host = strings.Trim(host, "[]")
	if host == "" {
		return fmt.Errorf("empty address")
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("could not resolve address %q: %v", addr, err)
	}

	for _, ip := range ips {
		if !isPublicRemoteIP(ip) {
			return fmt.Errorf("address %q resolves to restricted IP: %s", addr, ip.String())
		}
	}
	return nil
}

func isPublicRemoteIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	return !ip.IsLoopback() &&
		!ip.IsPrivate() &&
		!ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() &&
		!ip.IsUnspecified() &&
		!ip.IsMulticast()
}

// IsValidNodeAddress ensures the address is strictly a host or host:port.
// It explicitly denies schemes, paths, and query parameters.
func IsValidNodeAddress(addr string) error {
	addr = strings.TrimSpace(addr)
	if strings.Contains(addr, "://") {
		return fmt.Errorf("address must not contain scheme")
	}
	if strings.ContainsAny(addr, "/?") {
		return fmt.Errorf("address must not contain path or query parameters")
	}

	_, _, err := net.SplitHostPort(addr)
	if err != nil && !strings.Contains(err.Error(), "missing port in address") {
		return fmt.Errorf("invalid address format")
	}
	return nil
}
