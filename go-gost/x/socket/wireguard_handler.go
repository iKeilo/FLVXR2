//go:build linux

package socket

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type wireGuardPeerPlan struct {
	NodeID              int64    `json:"node_id"`
	PublicKey           string   `json:"public_key"`
	Endpoint            string   `json:"endpoint"`
	AllowedIPs          []string `json:"allowed_ips"`
	PersistentKeepalive int      `json:"persistent_keepalive"`
}

type wireGuardRoutePlan struct {
	Dst   string `json:"dst"`
	Table int    `json:"table"`
}

type wireGuardNFTPlan struct {
	Enabled bool   `json:"enabled"`
	Chain   string `json:"chain"`
	SNAT    bool   `json:"snat"`
}

type wireGuardPathPlan struct {
	PathID       int64                `json:"path_id"`
	Interface    string               `json:"interface"`
	ListenPort   int                  `json:"listen_port"`
	PrivateKey   string               `json:"private_key"`
	Addresses    []string             `json:"addresses"`
	Peers        []wireGuardPeerPlan  `json:"peers"`
	Routes       []wireGuardRoutePlan `json:"routes"`
	Nftables     wireGuardNFTPlan     `json:"nftables"`
	MTU          int                  `json:"mtu"`
	ExpectedHash string               `json:"expected_hash"`
}

type wireGuardForwardRulePlan struct {
	ForwardID  int64   `json:"forward_id"`
	PathID     int64   `json:"path_id"`
	Interface  string  `json:"interface"`
	RuleType   string  `json:"rule_type"`
	Role       string  `json:"role"`
	Protocol   string  `json:"protocol"`
	ListenAddr string  `json:"listen_addr"`
	ListenPort int     `json:"listen_port"`
	RemoteAddr string  `json:"remote_addr"`
	SourceCIDR string  `json:"source_cidr"`
	TargetCIDR string  `json:"target_cidr"`
	SNAT       bool    `json:"snat"`
	NodeOrder  []int64 `json:"node_order"`
	Fwmark     string  `json:"fwmark"`
	Table      int     `json:"table"`
	Comment    string  `json:"comment"`
}

func (w *WebSocketReporter) handleCheckWireGuardSupport(data interface{}) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"ip":        commandExists("ip"),
		"wg":        commandExists("wg"),
		"nft":       commandExists("nft"),
		"supported": commandExists("ip") && commandExists("wg"),
	}
	return result, nil
}

func (w *WebSocketReporter) handleApplyWireGuardPath(data interface{}) (map[string]interface{}, error) {
	var plan wireGuardPathPlan
	if err := decodeWireGuardCommand(data, &plan); err != nil {
		return nil, err
	}
	if plan.PathID <= 0 || strings.TrimSpace(plan.Interface) == "" {
		return nil, fmt.Errorf("invalid wireguard path plan")
	}
	if !commandExists("ip") {
		return nil, fmt.Errorf("ip command not found on this node")
	}
	if !commandExists("wg") {
		return nil, fmt.Errorf("wg command not found on this node; install wireguard-tools before applying WG path")
	}
	if plan.ListenPort <= 0 {
		plan.ListenPort = 51820
	}
	if plan.MTU <= 0 {
		plan.MTU = 1380
	}
	confPath, err := writeWireGuardConfig(plan)
	if err != nil {
		return nil, err
	}
	_ = runCommand("ip", "link", "del", plan.Interface)
	if err := runCommand("ip", "link", "add", "dev", plan.Interface, "type", "wireguard"); err != nil {
		return nil, err
	}
	for _, addr := range plan.Addresses {
		if strings.TrimSpace(addr) == "" {
			continue
		}
		if err := runCommand("ip", "address", "add", addr, "dev", plan.Interface); err != nil {
			_ = runCommand("ip", "link", "del", plan.Interface)
			return nil, err
		}
	}
	if err := runCommand("wg", "setconf", plan.Interface, confPath); err != nil {
		_ = runCommand("ip", "link", "del", plan.Interface)
		return nil, err
	}
	if err := runCommand("ip", "link", "set", "mtu", strconv.Itoa(plan.MTU), "dev", plan.Interface); err != nil {
		_ = runCommand("ip", "link", "del", plan.Interface)
		return nil, err
	}
	if err := runCommand("ip", "link", "set", "up", "dev", plan.Interface); err != nil {
		_ = runCommand("ip", "link", "del", plan.Interface)
		return nil, err
	}
	applyWireGuardRoutes(plan)
	_ = runCommand("sysctl", "-w", "net.ipv4.ip_forward=1")
	applyWireGuardNFTables(plan)
	actualHash := hashWireGuardPlan(plan)
	return map[string]interface{}{
		"path_id":     plan.PathID,
		"interface":   plan.Interface,
		"config_path": confPath,
		"actual_hash": actualHash,
	}, nil
}

func (w *WebSocketReporter) handleRemoveWireGuardPath(data interface{}) (map[string]interface{}, error) {
	var req struct {
		PathID    int64  `json:"path_id"`
		Interface string `json:"interface"`
	}
	if err := decodeWireGuardCommand(data, &req); err != nil {
		return nil, err
	}
	if req.Interface == "" && req.PathID > 0 {
		req.Interface = fmt.Sprintf("wg-flvx-%d", req.PathID)
	}
	if req.Interface == "" {
		return nil, fmt.Errorf("wireguard interface is required")
	}
	_ = runCommand("ip", "link", "del", req.Interface)
	if req.PathID > 0 {
		_ = os.Remove(wireGuardConfigPath(req.PathID))
		if commandExists("nft") {
			_ = runShell("nft delete chain inet flvx " + shellQuote(fmt.Sprintf("flvx_wg_path_%d", req.PathID)))
		}
	}
	return map[string]interface{}{"removed": true, "interface": req.Interface}, nil
}

func (w *WebSocketReporter) handleGetWireGuardPathStatus(data interface{}) (map[string]interface{}, error) {
	var req struct {
		PathID    int64  `json:"path_id"`
		Interface string `json:"interface"`
	}
	if err := decodeWireGuardCommand(data, &req); err != nil {
		return nil, err
	}
	if req.Interface == "" && req.PathID > 0 {
		req.Interface = fmt.Sprintf("wg-flvx-%d", req.PathID)
	}
	if req.Interface == "" {
		return nil, fmt.Errorf("wireguard interface is required")
	}
	linkOut, linkErr := commandOutput("ip", "link", "show", req.Interface)
	wgOut, wgErr := commandOutput("wg", "show", req.Interface, "dump")
	return map[string]interface{}{
		"interface": req.Interface,
		"up":        linkErr == nil,
		"link":      linkOut,
		"wg":        wgOut,
		"wg_error":  errorString(wgErr),
	}, nil
}

func (w *WebSocketReporter) handleProbeWireGuardPath(data interface{}) (map[string]interface{}, error) {
	var req struct {
		PathID    int64  `json:"path_id"`
		Interface string `json:"interface"`
		Target    string `json:"target"`
	}
	if err := decodeWireGuardCommand(data, &req); err != nil {
		return nil, err
	}
	if req.Interface == "" && req.PathID > 0 {
		req.Interface = fmt.Sprintf("wg-flvx-%d", req.PathID)
	}
	if strings.TrimSpace(req.Target) == "" {
		return w.handleGetWireGuardPathStatus(data)
	}
	start := time.Now()
	out, err := commandOutput("ping", "-c", "3", "-I", req.Interface, req.Target)
	return map[string]interface{}{
		"interface": req.Interface,
		"target":    req.Target,
		"success":   err == nil,
		"duration":  time.Since(start).Milliseconds(),
		"output":    out,
		"error":     errorString(err),
	}, nil
}

func (w *WebSocketReporter) handleApplyWGForwardRule(data interface{}) (map[string]interface{}, error) {
	var plan wireGuardForwardRulePlan
	if err := decodeWireGuardCommand(data, &plan); err != nil {
		return nil, err
	}
	if plan.ForwardID <= 0 || plan.PathID <= 0 {
		return nil, fmt.Errorf("invalid WG forward rule plan")
	}
	if strings.TrimSpace(plan.Interface) == "" {
		plan.Interface = fmt.Sprintf("wg-flvx-%d", plan.PathID)
	}
	if !commandExists("ip") {
		return nil, fmt.Errorf("ip command not found on this node")
	}
	if _, err := commandOutput("ip", "link", "show", plan.Interface); err != nil {
		return nil, fmt.Errorf("WG interface %s is not active: %w", plan.Interface, err)
	}
	if err := writeWGForwardRulePlan(plan); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"forward_id": plan.ForwardID,
		"path_id":    plan.PathID,
		"interface":  plan.Interface,
		"role":       plan.Role,
		"active":     true,
	}, nil
}

func (w *WebSocketReporter) handleRemoveWGForwardRule(data interface{}) (map[string]interface{}, error) {
	var req struct {
		ForwardID int64 `json:"forward_id"`
		PathID    int64 `json:"path_id"`
	}
	if err := decodeWireGuardCommand(data, &req); err != nil {
		return nil, err
	}
	if req.ForwardID <= 0 {
		return nil, fmt.Errorf("forward_id is required")
	}
	_ = os.Remove(wgForwardRulePath(req.ForwardID))
	return map[string]interface{}{"forward_id": req.ForwardID, "removed": true}, nil
}

func (w *WebSocketReporter) handleGetWGForwardRuleStatus(data interface{}) (map[string]interface{}, error) {
	var req struct {
		ForwardID int64  `json:"forward_id"`
		PathID    int64  `json:"path_id"`
		Interface string `json:"interface"`
	}
	if err := decodeWireGuardCommand(data, &req); err != nil {
		return nil, err
	}
	if req.ForwardID <= 0 {
		return nil, fmt.Errorf("forward_id is required")
	}
	_, statErr := os.Stat(wgForwardRulePath(req.ForwardID))
	if req.Interface == "" && req.PathID > 0 {
		req.Interface = fmt.Sprintf("wg-flvx-%d", req.PathID)
	}
	linkUp := false
	if req.Interface != "" {
		_, linkErr := commandOutput("ip", "link", "show", req.Interface)
		linkUp = linkErr == nil
	}
	return map[string]interface{}{
		"forward_id": req.ForwardID,
		"path_id":    req.PathID,
		"interface":  req.Interface,
		"configured": statErr == nil,
		"link_up":    linkUp,
		"success":    statErr == nil && linkUp,
	}, nil
}

func decodeWireGuardCommand(data interface{}, target interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return err
	}
	return nil
}

func writeWireGuardConfig(plan wireGuardPathPlan) (string, error) {
	if err := os.MkdirAll(filepath.Dir(wireGuardConfigPath(plan.PathID)), 0700); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + strings.TrimSpace(plan.PrivateKey) + "\n")
	b.WriteString("ListenPort = " + strconv.Itoa(plan.ListenPort) + "\n")
	for _, peer := range plan.Peers {
		b.WriteString("\n[Peer]\n")
		b.WriteString("PublicKey = " + strings.TrimSpace(peer.PublicKey) + "\n")
		if strings.TrimSpace(peer.Endpoint) != "" {
			b.WriteString("Endpoint = " + strings.TrimSpace(peer.Endpoint) + "\n")
		}
		if len(peer.AllowedIPs) > 0 {
			b.WriteString("AllowedIPs = " + strings.Join(peer.AllowedIPs, ", ") + "\n")
		}
		if peer.PersistentKeepalive > 0 {
			b.WriteString("PersistentKeepalive = " + strconv.Itoa(peer.PersistentKeepalive) + "\n")
		}
	}
	confPath := wireGuardConfigPath(plan.PathID)
	if err := os.WriteFile(confPath, []byte(b.String()), 0600); err != nil {
		return "", err
	}
	return confPath, nil
}

func wgForwardRulePath(forwardID int64) string {
	return filepath.Join("/etc/flvx_agent/wg", fmt.Sprintf("forward-%d.json", forwardID))
}

func writeWGForwardRulePlan(plan wireGuardForwardRulePlan) error {
	path := wgForwardRulePath(plan.ForwardID)
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0600)
}

func applyWireGuardNFTables(plan wireGuardPathPlan) {
	if !plan.Nftables.Enabled || !commandExists("nft") {
		return
	}
	chain := strings.TrimSpace(plan.Nftables.Chain)
	if chain == "" {
		chain = fmt.Sprintf("flvx_wg_path_%d", plan.PathID)
	}
	_ = runShell("nft add table inet flvx 2>/dev/null || true")
	_ = runShell("nft add chain inet flvx " + shellQuote(chain) + " '{ type nat hook postrouting priority srcnat; policy accept; }' 2>/dev/null || true")
	if plan.Nftables.SNAT {
		_ = runShell("nft add rule inet flvx " + shellQuote(chain) + " oifname != " + shellQuote(plan.Interface) + " masquerade 2>/dev/null || true")
	}
}

func applyWireGuardRoutes(plan wireGuardPathPlan) {
	seen := make(map[string]bool)
	addRoute := func(dst string) {
		dst = strings.TrimSpace(dst)
		if dst == "" || seen[dst] {
			return
		}
		seen[dst] = true
		_ = runCommand("ip", "route", "replace", dst, "dev", plan.Interface)
	}
	for _, route := range plan.Routes {
		addRoute(route.Dst)
	}
	for _, peer := range plan.Peers {
		for _, allowedIP := range peer.AllowedIPs {
			addRoute(allowedIP)
		}
	}
}

func wireGuardConfigPath(pathID int64) string {
	return filepath.Join("/etc/flvx_agent/wireguard", fmt.Sprintf("path-%d.conf", pathID))
}

func hashWireGuardPlan(plan wireGuardPathPlan) string {
	raw, _ := json.Marshal(plan)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func runCommand(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %s: %w", name, strings.Join(args, " "), strings.TrimSpace(string(out)), err)
	}
	return nil
}

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("%s %s failed: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runShell(script string) error {
	out, err := exec.Command("sh", "-lc", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("shell command failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
