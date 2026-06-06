package handler

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/store/model"
	"go-backend/internal/store/repo"
)

type wgIdentityRequest struct {
	NodeID int64 `json:"nodeId"`
}

type pathIDRequest struct {
	ID int64 `json:"id"`
}

type pathCreateRequest struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Transport     string  `json:"transport"`
	NodeIDs       []int64 `json:"nodeIds"`
	Remark        string  `json:"remark"`
	ListenStart   int     `json:"listenStart"`
	MTU           int     `json:"mtu"`
	TunnelGroupID int64   `json:"tunnelGroupId"`
	Flow          int     `json:"flow"`
	TrafficRatio  float64 `json:"trafficRatio"`
}

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

func (h *Handler) wgIdentity(w http.ResponseWriter, r *http.Request) {
	var req wgIdentityRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	item, err := h.ensureWGIdentity(req.NodeID, false)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(item))
}

func (h *Handler) wgIdentityRegenerate(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req wgIdentityRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	item, err := h.ensureWGIdentity(req.NodeID, true)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(item))
}

func (h *Handler) pathList(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	items, err := h.repo.ListPathTunnels()
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(items))
}

func (h *Handler) pathDetail(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	item, err := h.repo.GetPathTunnelDetail(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if item == nil {
		response.WriteJSON(w, response.ErrDefault("Path 不存在"))
		return
	}
	response.WriteJSON(w, response.OK(item))
}

func (h *Handler) pathCreate(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathCreateRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Transport = strings.ToLower(strings.TrimSpace(req.Transport))
	if req.Transport == "" {
		req.Transport = "wireguard"
	}
	if req.Transport != "wireguard" {
		response.WriteJSON(w, response.ErrDefault("第一版仅支持 WireGuard Path"))
		return
	}
	if req.Name == "" || len(req.NodeIDs) < 2 || len(req.NodeIDs) > 8 {
		response.WriteJSON(w, response.ErrDefault("请填写名称，并选择 2-3 个节点"))
		return
	}
	if hasDuplicateNodeIDs(req.NodeIDs) {
		response.WriteJSON(w, response.ErrDefault("同一条 Path 内不能重复选择节点"))
		return
	}
	if req.Flow <= 0 {
		req.Flow = 1
	}
	if req.TrafficRatio <= 0 {
		req.TrafficRatio = 1
	}
	actorID, _, _ := userRoleFromRequest(r)
	for _, nodeID := range req.NodeIDs {
		node, err := h.getNodeRecord(nodeID)
		if err != nil {
			response.WriteJSON(w, response.ErrDefault(err.Error()))
			return
		}
		if node.IsRemote == 1 {
			response.WriteJSON(w, response.ErrDefault("WG 隧道不支持远程共享节点"))
			return
		}
		if _, err := h.ensureWGIdentity(nodeID, false); err != nil {
			response.WriteJSON(w, response.ErrDefault(err.Error()))
			return
		}
	}
	path := &model.PathTunnel{
		Name:          req.Name,
		Transport:     "wireguard",
		Status:        "pending",
		OwnerUserID:   actorID,
		CreatedBy:     actorID,
		TunnelGroupID: req.TunnelGroupID,
		Flow:          req.Flow,
		TrafficRatio:  req.TrafficRatio,
		Remark:        strings.TrimSpace(req.Remark),
	}
	segments, resources, err := h.buildInitialWGPathRecords(req)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if err := h.repo.CreatePathTunnel(path, segments, resources); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	detail, _ := h.repo.GetPathTunnelDetail(path.ID)
	response.WriteJSON(w, response.OK(detail))
}

func (h *Handler) pathUpdate(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathCreateRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Transport = strings.ToLower(strings.TrimSpace(req.Transport))
	if req.Transport == "" {
		req.Transport = "wireguard"
	}
	if req.Transport != "wireguard" {
		response.WriteJSON(w, response.ErrDefault("第一版仅支持 WireGuard Path"))
		return
	}
	if req.Name == "" || len(req.NodeIDs) < 2 || len(req.NodeIDs) > 8 {
		response.WriteJSON(w, response.ErrDefault("请填写名称，并选择 2-8 个节点"))
		return
	}
	if hasDuplicateNodeIDs(req.NodeIDs) {
		response.WriteJSON(w, response.ErrDefault("同一条 Path 内不能重复选择节点"))
		return
	}
	if req.Flow <= 0 {
		req.Flow = 1
	}
	if req.TrafficRatio <= 0 {
		req.TrafficRatio = 1
	}
	existing, err := h.repo.GetPathTunnelDetail(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if existing == nil {
		response.WriteJSON(w, response.ErrDefault("Path 不存在"))
		return
	}
	for _, nodeID := range req.NodeIDs {
		node, err := h.getNodeRecord(nodeID)
		if err != nil {
			response.WriteJSON(w, response.ErrDefault(err.Error()))
			return
		}
		if node.IsRemote == 1 {
			response.WriteJSON(w, response.ErrDefault("WG 隧道不支持远程共享节点"))
			return
		}
		if _, err := h.ensureWGIdentity(nodeID, false); err != nil {
			response.WriteJSON(w, response.ErrDefault(err.Error()))
			return
		}
	}
	path := &model.PathTunnel{
		ID:            req.ID,
		Name:          req.Name,
		Transport:     "wireguard",
		Status:        "pending",
		TunnelGroupID: req.TunnelGroupID,
		Flow:          req.Flow,
		TrafficRatio:  req.TrafficRatio,
		Remark:        strings.TrimSpace(req.Remark),
	}
	segments, resources, err := h.buildInitialWGPathRecords(req)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if err := h.repo.UpdatePathTunnel(path, segments, resources); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	detail, _ := h.repo.GetPathTunnelDetail(req.ID)
	response.WriteJSON(w, response.OK(detail))
}

func (h *Handler) pathApply(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	detail, err := h.repo.GetPathTunnelDetail(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if detail == nil {
		response.WriteJSON(w, response.ErrDefault("Path 不存在"))
		return
	}
	plans, expectedHash, err := h.buildWGPathPlans(detail)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	runtime := &model.PathRuntimeVersion{PathID: req.ID, ExpectedHash: expectedHash, Status: "applying"}
	if err := h.repo.CreatePathRuntimeVersion(runtime); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	_ = h.repo.UpdatePathTunnelStatus(req.ID, "applying")
	var errs []string
	var actualHashes []string
	for nodeID, plan := range plans {
		result, err := h.sendNodeCommandWithTimeout(nodeID, "ApplyWireGuardPath", plan, 2*time.Minute, false, false)
		if err != nil {
			errs = append(errs, fmt.Sprintf("node %d: %v", nodeID, err))
			continue
		}
		if v, ok := result.Data["actual_hash"].(string); ok && v != "" {
			actualHashes = append(actualHashes, v)
		}
	}
	if len(errs) > 0 {
		msg := strings.Join(errs, "; ")
		_ = h.repo.UpdatePathRuntimeVersion(runtime.ID, "failed", strings.Join(actualHashes, ","), msg)
		_ = h.repo.UpdatePathTunnelStatus(req.ID, "failed")
		response.WriteJSON(w, response.ErrDefault(msg))
		return
	}
	_ = h.repo.UpdatePathRuntimeVersion(runtime.ID, "active", strings.Join(actualHashes, ","), "OK")
	_ = h.repo.UpdatePathTunnelStatus(req.ID, "active")
	response.WriteJSON(w, response.OK(map[string]interface{}{"pathId": req.ID, "status": "active"}))
}

func (h *Handler) pathRemove(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	detail, err := h.repo.GetPathTunnelDetail(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if detail == nil {
		response.WriteJSON(w, response.ErrDefault("Path 不存在"))
		return
	}
	nodeIDs := uniquePathNodeIDs(detail.Segments)
	var errs []string
	for _, nodeID := range nodeIDs {
		if _, err := h.sendNodeCommandWithTimeout(nodeID, "RemoveWireGuardPath", map[string]interface{}{
			"path_id":   req.ID,
			"interface": wgInterfaceName(req.ID),
		}, time.Minute, true, true); err != nil {
			errs = append(errs, fmt.Sprintf("node %d: %v", nodeID, err))
		}
	}
	if len(errs) > 0 {
		_ = h.repo.UpdatePathTunnelStatus(req.ID, "failed")
		response.WriteJSON(w, response.ErrDefault(strings.Join(errs, "; ")))
		return
	}
	_ = h.repo.UpdatePathTunnelStatus(req.ID, "removed")
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) pathDelete(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	if err := h.repo.DeletePathTunnel(req.ID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) pathStatus(w http.ResponseWriter, r *http.Request) {
	if !h.ensureAdminAccess(w, r) {
		return
	}
	var req pathIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	detail, err := h.repo.GetPathTunnelDetail(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	if detail == nil {
		response.WriteJSON(w, response.ErrDefault("Path 不存在"))
		return
	}
	results := make(map[int64]interface{})
	for _, nodeID := range uniquePathNodeIDs(detail.Segments) {
		res, err := h.sendNodeCommandWithTimeout(nodeID, "GetWireGuardPathStatus", map[string]interface{}{
			"path_id":   req.ID,
			"interface": wgInterfaceName(req.ID),
		}, 30*time.Second, false, false)
		if err != nil {
			results[nodeID] = map[string]interface{}{"success": false, "message": err.Error()}
		} else {
			results[nodeID] = res.Data
		}
	}
	response.WriteJSON(w, response.OK(results))
}

func (h *Handler) pathProbe(w http.ResponseWriter, r *http.Request) {
	h.pathStatus(w, r)
}

func (h *Handler) ensureWGIdentity(nodeID int64, regenerate bool) (*model.WGNodeIdentity, error) {
	if !regenerate {
		if item, err := h.repo.GetWGNodeIdentity(nodeID); err != nil {
			return nil, err
		} else if item != nil {
			return item, nil
		}
	}
	privateKey, publicKey, err := generateWGKeypair()
	if err != nil {
		return nil, err
	}
	item := &model.WGNodeIdentity{
		NodeID:              nodeID,
		PrivateKeyEncrypted: privateKey,
		PublicKey:           publicKey,
		DefaultListenPort:   51820 + int(nodeID%10000),
		Enabled:             1,
	}
	if err := h.repo.SaveWGNodeIdentity(item); err != nil {
		return nil, err
	}
	return item, nil
}

func generateWGKeypair() (string, string, error) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(privateKey.Bytes()), base64.StdEncoding.EncodeToString(privateKey.PublicKey().Bytes()), nil
}

func hasDuplicateNodeIDs(ids []int64) bool {
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 || seen[id] {
			return true
		}
		seen[id] = true
	}
	return false
}

func (h *Handler) buildInitialWGPathRecords(req pathCreateRequest) ([]model.PathSegment, []model.NodeRuntimeResource, error) {
	listenStart := req.ListenStart
	if listenStart <= 0 {
		listenStart = 51820
	}
	baseIP := net.IPv4(10, 88, byte(time.Now().UnixNano()%200+1), 0)
	var segments []model.PathSegment
	for i := 0; i < len(req.NodeIDs)-1; i++ {
		fromIP := make(net.IP, len(baseIP))
		toIP := make(net.IP, len(baseIP))
		copy(fromIP, baseIP)
		copy(toIP, baseIP)
		fromIP[3] = byte(i*2 + 1)
		toIP[3] = byte(i*2 + 2)
		segments = append(segments, model.PathSegment{
			Sequence:     i + 1,
			FromNodeID:   req.NodeIDs[i],
			ToNodeID:     req.NodeIDs[i+1],
			Transport:    "wireguard",
			Status:       "pending",
			ListenPort:   listenStart + i,
			TunnelIPFrom: fromIP.String(),
			TunnelIPTo:   toIP.String(),
		})
	}
	nodeIDs := append([]int64{}, req.NodeIDs...)
	sort.Slice(nodeIDs, func(i, j int) bool { return nodeIDs[i] < nodeIDs[j] })
	nodeIDs = uniqueWGPathInt64s(nodeIDs)
	var resources []model.NodeRuntimeResource
	for _, nodeID := range nodeIDs {
		resources = append(resources,
			model.NodeRuntimeResource{NodeID: nodeID, ResourceType: "wireguard_interface", ResourceKey: "__path_interface__"},
			model.NodeRuntimeResource{NodeID: nodeID, ResourceType: "route_table", ResourceKey: "__path_table__"},
			model.NodeRuntimeResource{NodeID: nodeID, ResourceType: "fwmark", ResourceKey: "__path_fwmark__"},
			model.NodeRuntimeResource{NodeID: nodeID, ResourceType: "nft_chain", ResourceKey: "__path_chain__"},
		)
	}
	for i, nodeID := range nodeIDs {
		resources = append(resources, model.NodeRuntimeResource{
			NodeID:       nodeID,
			ResourceType: "port",
			Protocol:     "udp",
			Port:         listenStart + i,
			ResourceKey:  fmt.Sprintf("udp:%d", listenStart+i),
		})
	}
	return segments, resources, nil
}

func uniqueWGPathInt64s(ids []int64) []int64 {
	out := ids[:0]
	var last int64
	for i, id := range ids {
		if i == 0 || id != last {
			out = append(out, id)
			last = id
		}
	}
	return out
}

func uniquePathNodeIDs(segments []model.PathSegment) []int64 {
	m := make(map[int64]bool)
	for _, seg := range segments {
		m[seg.FromNodeID] = true
		m[seg.ToNodeID] = true
	}
	out := make([]int64, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (h *Handler) buildWGPathPlans(detail *repo.PathTunnelDetail) (map[int64]wireGuardPathPlan, string, error) {
	if detail == nil || detail.Path.ID <= 0 || len(detail.Segments) == 0 {
		return nil, "", errors.New("invalid path detail")
	}
	type nodePlanState struct {
		nodeID    int64
		private   string
		public    string
		addresses []string
		peers     []wireGuardPeerPlan
	}
	states := make(map[int64]*nodePlanState)
	addressesByNode := make(map[int64][]string)
	getState := func(nodeID int64) (*nodePlanState, error) {
		if st := states[nodeID]; st != nil {
			return st, nil
		}
		ident, err := h.repo.GetWGNodeIdentity(nodeID)
		if err != nil {
			return nil, err
		}
		if ident == nil {
			return nil, fmt.Errorf("node %d WireGuard identity not found", nodeID)
		}
		st := &nodePlanState{nodeID: nodeID, private: ident.PrivateKeyEncrypted, public: ident.PublicKey}
		states[nodeID] = st
		return st, nil
	}
	nodeOrder := orderedWGPathNodeIDs(detail.Segments)
	nodeIndex := make(map[int64]int, len(nodeOrder))
	for i, nodeID := range nodeOrder {
		nodeIndex[nodeID] = i
	}
	for _, seg := range detail.Segments {
		from, err := getState(seg.FromNodeID)
		if err != nil {
			return nil, "", err
		}
		to, err := getState(seg.ToNodeID)
		if err != nil {
			return nil, "", err
		}
		from.addresses = appendUniqueString(from.addresses, seg.TunnelIPFrom+"/32")
		to.addresses = appendUniqueString(to.addresses, seg.TunnelIPTo+"/32")
		addressesByNode[seg.FromNodeID] = appendUniqueString(addressesByNode[seg.FromNodeID], seg.TunnelIPFrom+"/32")
		addressesByNode[seg.ToNodeID] = appendUniqueString(addressesByNode[seg.ToNodeID], seg.TunnelIPTo+"/32")
	}
	for _, seg := range detail.Segments {
		from, err := getState(seg.FromNodeID)
		if err != nil {
			return nil, "", err
		}
		to, err := getState(seg.ToNodeID)
		if err != nil {
			return nil, "", err
		}
		toNode, err := h.getNodeRecord(seg.ToNodeID)
		if err != nil {
			return nil, "", err
		}
		fromNode, err := h.getNodeRecord(seg.FromNodeID)
		if err != nil {
			return nil, "", err
		}
		toPort := findPathResourcePort(detail.Resources, seg.ToNodeID, "udp")
		if toPort <= 0 {
			toPort = seg.ListenPort
		}
		fromPort := findPathResourcePort(detail.Resources, seg.FromNodeID, "udp")
		if fromPort <= 0 {
			fromPort = seg.ListenPort
		}
		from.peers = append(from.peers, wireGuardPeerPlan{
			NodeID:              seg.ToNodeID,
			PublicKey:           to.public,
			Endpoint:            net.JoinHostPort(resolveNodeEndpoint(toNode), strconv.Itoa(toPort)),
			AllowedIPs:          allowedWGPathIPs(nodeOrder, nodeIndex, addressesByNode, seg.ToNodeID, 1),
			PersistentKeepalive: 25,
		})
		to.peers = append(to.peers, wireGuardPeerPlan{
			NodeID:              seg.FromNodeID,
			PublicKey:           from.public,
			Endpoint:            net.JoinHostPort(resolveNodeEndpoint(fromNode), strconv.Itoa(fromPort)),
			AllowedIPs:          allowedWGPathIPs(nodeOrder, nodeIndex, addressesByNode, seg.FromNodeID, -1),
			PersistentKeepalive: 25,
		})
	}
	plans := make(map[int64]wireGuardPathPlan, len(states))
	table := 100000 + int(detail.Path.ID)
	mtu := 1380
	for nodeID, st := range states {
		sort.Strings(st.addresses)
		listenPort := findPathResourcePort(detail.Resources, nodeID, "udp")
		if listenPort <= 0 {
			listenPort = 51820 + int(detail.Path.ID%10000)
		}
		plan := wireGuardPathPlan{
			PathID:     detail.Path.ID,
			Interface:  wgInterfaceName(detail.Path.ID),
			ListenPort: listenPort,
			PrivateKey: st.private,
			Addresses:  st.addresses,
			Peers:      st.peers,
			Routes:     []wireGuardRoutePlan{{Dst: wgPathCIDR(detail.Segments), Table: table}},
			Nftables:   wireGuardNFTPlan{Enabled: true, Chain: fmt.Sprintf("flvx_wg_path_%d", detail.Path.ID), SNAT: true},
			MTU:        mtu,
		}
		plans[nodeID] = plan
	}
	raw, _ := json.Marshal(plans)
	sum := sha256.Sum256(raw)
	expectedHash := hex.EncodeToString(sum[:])
	for nodeID, plan := range plans {
		plan.ExpectedHash = expectedHash
		plans[nodeID] = plan
	}
	return plans, expectedHash, nil
}

func orderedWGPathNodeIDs(segments []model.PathSegment) []int64 {
	if len(segments) == 0 {
		return nil
	}
	sorted := append([]model.PathSegment(nil), segments...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Sequence < sorted[j].Sequence })
	out := []int64{sorted[0].FromNodeID}
	for _, seg := range sorted {
		out = appendUniqueInt64(out, seg.ToNodeID)
	}
	return out
}

func allowedWGPathIPs(order []int64, index map[int64]int, addresses map[int64][]string, peerNodeID int64, direction int) []string {
	start, ok := index[peerNodeID]
	if !ok {
		return append([]string(nil), addresses[peerNodeID]...)
	}
	var out []string
	if direction >= 0 {
		for i := start; i < len(order); i++ {
			out = append(out, addresses[order[i]]...)
		}
	} else {
		for i := start; i >= 0; i-- {
			out = append(out, addresses[order[i]]...)
		}
	}
	sort.Strings(out)
	return out
}

func findPathResourcePort(resources []model.NodeRuntimeResource, nodeID int64, protocol string) int {
	protocol = strings.ToLower(strings.TrimSpace(protocol))
	for _, item := range resources {
		if item.NodeID == nodeID && item.ResourceType == "port" && strings.ToLower(item.Protocol) == protocol && item.Port > 0 {
			return item.Port
		}
	}
	return 0
}

func appendUniqueString(items []string, item string) []string {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func appendUniqueInt64(items []int64, item int64) []int64 {
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}

func resolveNodeEndpoint(node *nodeRecord) string {
	for _, value := range []string{node.ServerIPv4, node.ServerIP, node.ServerIPv6} {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return node.Name
}

func wgInterfaceName(pathID int64) string {
	return fmt.Sprintf("wg-flvx-%d", pathID)
}

func wgPathCIDR(segments []model.PathSegment) string {
	if len(segments) == 0 {
		return "10.88.0.0/24"
	}
	ip := net.ParseIP(segments[0].TunnelIPFrom).To4()
	if ip == nil {
		return "10.88.0.0/24"
	}
	ip[3] = 0
	return ip.String() + "/24"
}
