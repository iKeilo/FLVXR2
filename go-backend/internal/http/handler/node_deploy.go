package handler

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go-backend/internal/http/response"
	"go-backend/internal/store/model"
)

type nodeTLSRequest struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	ServerJSON string `json:"serverJson"`
	ClientJSON string `json:"clientJson"`
	Remark     string `json:"remark"`
}

type nodeInboundDeployRequest struct {
	ID                 int64  `json:"id"`
	NodeID             int64  `json:"nodeId"`
	Name               string `json:"name"`
	Protocol           string `json:"protocol"`
	ListenAddr         string `json:"listenAddr"`
	ListenPort         int    `json:"listenPort"`
	PublishAddr        string `json:"publishAddr"`
	PublishPort        int    `json:"publishPort"`
	TLSTemplateID      int64  `json:"tlsTemplateId"`
	InboundOptionsJSON string `json:"inboundOptionsJson"`
	Enabled            int    `json:"enabled"`
	Apply              bool   `json:"apply"`
}

type nodeIDRequest struct {
	NodeID int64 `json:"nodeId"`
}

type nodeDeployIDRequest struct {
	ID int64 `json:"id"`
}

type nodeDeployRollbackRequest struct {
	NodeID     int64 `json:"nodeId"`
	RevisionID int64 `json:"revisionId"`
}

func (h *Handler) nodeTLSList(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListNodeTLSTemplates()
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(items))
}

func (h *Handler) nodeTLSSave(w http.ResponseWriter, r *http.Request) {
	var req nodeTLSRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	item := &model.NodeTLSTemplate{
		ID:         req.ID,
		Name:       strings.TrimSpace(req.Name),
		Type:       strings.ToLower(strings.TrimSpace(req.Type)),
		ServerJSON: normalizeJSONText(req.ServerJSON),
		ClientJSON: normalizeJSONText(req.ClientJSON),
		Remark:     nullableString(req.Remark),
	}
	if err := h.repo.SaveNodeTLSTemplate(item); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(item))
}

func (h *Handler) nodeTLSDelete(w http.ResponseWriter, r *http.Request) {
	var req nodeDeployIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	if err := h.repo.DeleteNodeTLSTemplate(req.ID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) nodeDeployDetail(w http.ResponseWriter, r *http.Request) {
	var req nodeIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	node, err := h.repo.GetNodeByID(req.NodeID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault("节点不存在"))
		return
	}
	identity, err := h.ensureNodeIdentity(req.NodeID, false)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	inbounds, _ := h.repo.ListNodeDeployedInbounds(req.NodeID)
	revisions, _ := h.repo.ListNodeConfigRevisions(req.NodeID, 20)
	logs, _ := h.repo.ListNodeDeployLogs(req.NodeID, 50)
	response.WriteJSON(w, response.OK(map[string]interface{}{
		"node":      node,
		"identity":  identity,
		"inbounds":  inbounds,
		"revisions": revisions,
		"logs":      logs,
	}))
}

func (h *Handler) nodeIdentityRegenerate(w http.ResponseWriter, r *http.Request) {
	var req nodeIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	identity, err := h.ensureNodeIdentity(req.NodeID, true)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(identity))
}

func (h *Handler) nodeDeploySaveInbound(w http.ResponseWriter, r *http.Request) {
	var req nodeInboundDeployRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	result, err := h.saveNodeInboundDeployment(req)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	response.WriteJSON(w, response.OK(result))
}

func (h *Handler) nodeDeployDeleteInbound(w http.ResponseWriter, r *http.Request) {
	var req nodeDeployIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.ID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	item, err := h.repo.GetNodeDeployedInbound(req.ID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault("部署入站不存在"))
		return
	}
	if err := h.repo.DeleteNodeDeployedInbound(req.ID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	revision, cfg, err := h.renderAndStoreNodeConfig(item.NodeID, "generated")
	if err == nil {
		_, err = h.sendNodeCommandWithTimeout(item.NodeID, "ApplyCoreConfig", map[string]interface{}{
			"coreType":   revision.CoreType,
			"configJson": cfg,
			"checksum":   revision.Checksum,
		}, 20*time.Second, false, false)
	}
	if err != nil {
		_ = h.repo.CreateNodeDeployLog(item.NodeID, 0, "delete-inbound", "failed", err.Error())
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, "deployed", "")
	_ = h.repo.CreateNodeDeployLog(item.NodeID, revision.ID, "delete-inbound", "success", "OK")
	response.WriteJSON(w, response.OKEmpty())
}

func (h *Handler) nodeDeployApply(w http.ResponseWriter, r *http.Request) {
	var req nodeIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	revision, cfg, err := h.renderAndStoreNodeConfig(req.NodeID, "generated")
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	_, err = h.sendNodeCommandWithTimeout(req.NodeID, "ApplyCoreConfig", map[string]interface{}{
		"coreType":   revision.CoreType,
		"configJson": cfg,
		"checksum":   revision.Checksum,
	}, 20*time.Second, false, false)
	if err != nil {
		_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, "failed", err.Error())
		_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "apply", "failed", err.Error())
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, "deployed", "")
	_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "apply", "success", "OK")
	response.WriteJSON(w, response.OK(revision))
}

func (h *Handler) nodeDeployRollback(w http.ResponseWriter, r *http.Request) {
	var req nodeDeployRollbackRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 || req.RevisionID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	revision, err := h.repo.GetNodeConfigRevision(req.RevisionID)
	if err != nil || revision.NodeID != req.NodeID {
		response.WriteJSON(w, response.ErrDefault("配置版本不存在"))
		return
	}
	_, err = h.sendNodeCommandWithTimeout(req.NodeID, "ApplyCoreConfig", map[string]interface{}{
		"coreType":   revision.CoreType,
		"configJson": revision.ConfigJSON,
		"checksum":   revision.Checksum,
	}, 20*time.Second, false, false)
	if err != nil {
		_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "rollback", "failed", err.Error())
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, "deployed", "")
	_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "rollback", "success", "OK")
	response.WriteJSON(w, response.OK(revision))
}

func (h *Handler) saveNodeInboundDeployment(req nodeInboundDeployRequest) (map[string]interface{}, error) {
	node, err := h.repo.GetNodeByID(req.NodeID)
	if err != nil {
		return nil, errors.New("节点不存在")
	}
	identity, err := h.ensureNodeIdentity(req.NodeID, false)
	if err != nil {
		return nil, err
	}
	protocol := strings.ToLower(strings.TrimSpace(req.Protocol))
	if protocol == "" {
		return nil, errors.New("协议不能为空")
	}
	listenAddr := strings.TrimSpace(req.ListenAddr)
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	if req.ListenPort <= 0 || req.ListenPort > 65535 {
		req.ListenPort = randomPort()
	}
	publishAddr := strings.TrimSpace(req.PublishAddr)
	if publishAddr == "" {
		publishAddr = firstNonEmpty(node.ServerIP, nullableToString(node.ServerIPV4), nullableToString(node.ServerIPV6))
	}
	if req.PublishPort <= 0 || req.PublishPort > 65535 {
		req.PublishPort = req.ListenPort
	}
	displayName := strings.TrimSpace(req.Name)
	if displayName == "" {
		displayName = fmt.Sprintf("%s-%s", node.Name, strings.ToUpper(protocol))
	}
	displayName, err = h.repo.NextNodeInboundDisplayName(req.NodeID, displayName, req.ID)
	if err != nil {
		return nil, err
	}
	options := []byte(normalizeJSONText(req.InboundOptionsJSON))
	tlsTemplate, _ := h.repo.GetNodeTLSTemplate(req.TLSTemplateID)
	internalTag := fmt.Sprintf("node-%d-%s-%s-%s", req.NodeID, protocol, identity.ServiceSuffix, randomHex(3))
	if req.ID > 0 {
		internalTag = fmt.Sprintf("node-%d-%s-%d", req.NodeID, protocol, req.ID)
	}
	serverConfig, clientConfig, shareURI, err := renderInboundConfig(protocol, internalTag, listenAddr, req.ListenPort, publishAddr, req.PublishPort, options, identity, tlsTemplate)
	if err != nil {
		return nil, err
	}
	enabled := req.Enabled
	if enabled == 0 {
		enabled = 1
	}
	item := &model.NodeDeployedInbound{
		ID:                 req.ID,
		NodeID:             req.NodeID,
		DisplayName:        displayName,
		InternalTag:        internalTag,
		Protocol:           protocol,
		ListenAddr:         listenAddr,
		ListenPort:         req.ListenPort,
		PublishAddr:        publishAddr,
		PublishPort:        req.PublishPort,
		TLSTemplateID:      nullableInt64FromValue(req.TLSTemplateID),
		InboundOptionsJSON: string(options),
		ClientConfigJSON:   clientConfig,
		ServerConfigJSON:   serverConfig,
		ShareURI:           shareURI,
		Enabled:            enabled,
	}
	if req.ID > 0 {
		old, err := h.repo.GetNodeDeployedInbound(req.ID)
		if err == nil {
			item.CreatedTime = old.CreatedTime
		}
	}
	if err := h.repo.SaveNodeDeployedInbound(item); err != nil {
		return nil, err
	}
	revision, cfg, err := h.renderAndStoreNodeConfig(req.NodeID, "generated")
	if err != nil {
		return nil, err
	}
	deployStatus := "generated"
	if req.Apply {
		_, err = h.sendNodeCommandWithTimeout(req.NodeID, "ApplyCoreConfig", map[string]interface{}{
			"coreType":   revision.CoreType,
			"configJson": cfg,
			"checksum":   revision.Checksum,
		}, 20*time.Second, false, false)
		if err != nil {
			_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, "failed", err.Error())
			_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "apply-inbound", "failed", err.Error())
			return nil, err
		}
		deployStatus = "deployed"
		_ = h.repo.UpdateNodeConfigRevisionStatus(revision.ID, deployStatus, "")
		_ = h.repo.CreateNodeDeployLog(req.NodeID, revision.ID, "apply-inbound", "success", "OK")
	}
	return map[string]interface{}{
		"inbound":      item,
		"revision":     revision,
		"configJson":   cfg,
		"deployStatus": deployStatus,
	}, nil
}

func (h *Handler) renderAndStoreNodeConfig(nodeID int64, status string) (*model.NodeConfigRevision, string, error) {
	inbounds, err := h.repo.ListNodeDeployedInbounds(nodeID)
	if err != nil {
		return nil, "", err
	}
	rendered := make([]map[string]interface{}, 0, len(inbounds))
	for _, inbound := range inbounds {
		if inbound.Enabled != 1 {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(inbound.ServerConfigJSON), &obj); err != nil {
			return nil, "", err
		}
		rendered = append(rendered, obj)
	}
	cfgObj := map[string]interface{}{
		"log":      map[string]interface{}{"level": "warning"},
		"inbounds": rendered,
		"outbounds": []map[string]interface{}{
			{"type": "direct", "tag": "direct"},
		},
	}
	raw, err := json.MarshalIndent(cfgObj, "", "  ")
	if err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(raw)
	revision := &model.NodeConfigRevision{
		NodeID:     nodeID,
		CoreType:   "sing-box",
		ConfigJSON: string(raw),
		Status:     status,
		Checksum:   hex.EncodeToString(sum[:]),
	}
	if err := h.repo.CreateNodeConfigRevision(revision); err != nil {
		return nil, "", err
	}
	return revision, string(raw), nil
}

func (h *Handler) ensureNodeIdentity(nodeID int64, force bool) (*model.NodeIdentity, error) {
	if !force {
		if item, err := h.repo.GetNodeIdentity(nodeID); err == nil && item != nil {
			return item, nil
		}
	}
	uuid := randomUUID()
	now := time.Now().UnixMilli()
	item := &model.NodeIdentity{
		NodeID:                  nodeID,
		UUID:                    uuid,
		Seed:                    randomHex(16),
		MixedPassword:           deployRandomToken(12),
		TrojanPassword:          deployRandomToken(12),
		Hysteria2Password:       deployRandomToken(12),
		TUICUUID:                randomUUID(),
		TUICPassword:            deployRandomToken(12),
		RealityShortID:          randomHex(8),
		PathSuffix:              deployRandomToken(8),
		ServiceSuffix:           strings.ToLower(deployRandomToken(8)),
		ProtocolCredentialsJSON: "{}",
		CreatedTime:             now,
		UpdatedTime:             now,
	}
	if err := h.repo.SaveNodeIdentity(item); err != nil {
		return nil, err
	}
	return item, nil
}

func renderInboundConfig(protocol, tag, listenAddr string, listenPort int, publishAddr string, publishPort int, options []byte, identity *model.NodeIdentity, tlsTemplate *model.NodeTLSTemplate) (string, string, string, error) {
	var opt map[string]interface{}
	if len(options) == 0 {
		opt = map[string]interface{}{}
	} else if err := json.Unmarshal(options, &opt); err != nil {
		return "", "", "", err
	}
	inbound := map[string]interface{}{
		"type":        protocol,
		"tag":         tag,
		"listen":      listenAddr,
		"listen_port": listenPort,
	}
	for k, v := range opt {
		inbound[k] = v
	}
	var tlsClient map[string]interface{}
	if tlsTemplate != nil {
		var tlsServer map[string]interface{}
		_ = json.Unmarshal([]byte(tlsTemplate.ServerJSON), &tlsServer)
		if tlsServer == nil {
			tlsServer = map[string]interface{}{}
		}
		_ = json.Unmarshal([]byte(tlsTemplate.ClientJSON), &tlsClient)
		if tlsClient == nil {
			tlsClient = map[string]interface{}{}
		}
		if strings.EqualFold(tlsTemplate.Type, "reality") {
			tlsServer["enabled"] = true
			reality := asMap(tlsServer["reality"])
			reality["enabled"] = true
			if reality["short_id"] == nil {
				reality["short_id"] = []string{identity.RealityShortID}
			}
			tlsServer["reality"] = reality
			clientReality := asMap(tlsClient["reality"])
			clientReality["enabled"] = true
			if clientReality["short_id"] == nil {
				clientReality["short_id"] = identity.RealityShortID
			}
			tlsClient["reality"] = clientReality
		}
		inbound["tls"] = tlsServer
	}
	switch protocol {
	case "vless":
		inbound["users"] = []map[string]interface{}{{"uuid": identity.UUID, "flow": optString(opt, "flow", "")}}
	case "trojan":
		inbound["users"] = []map[string]interface{}{{"password": identity.TrojanPassword}}
	case "hysteria2":
		inbound["users"] = []map[string]interface{}{{"password": identity.Hysteria2Password}}
	case "tuic":
		inbound["users"] = []map[string]interface{}{{"uuid": identity.TUICUUID, "password": identity.TUICPassword}}
	case "shadowsocks":
		inbound["method"] = optString(opt, "method", "2022-blake3-aes-128-gcm")
		inbound["password"] = identity.MixedPassword
	case "mixed", "socks", "http":
		inbound["users"] = []map[string]interface{}{{"username": identity.UUID, "password": identity.MixedPassword}}
	}
	client := map[string]interface{}{
		"type":        protocol,
		"tag":         tag,
		"server":      publishAddr,
		"server_port": publishPort,
		"uuid":        identity.UUID,
		"password":    protocolPassword(protocol, identity),
		"tlsTemplate": nil,
	}
	if tlsTemplate != nil {
		client["tlsTemplate"] = tlsClient
	}
	serverRaw, _ := json.MarshalIndent(inbound, "", "  ")
	clientRaw, _ := json.MarshalIndent(client, "", "  ")
	return string(serverRaw), string(clientRaw), shareURI(protocol, publishAddr, publishPort, identity, tag, tlsTemplate, tlsClient), nil
}

func shareURI(protocol, host string, port int, identity *model.NodeIdentity, tag string, tlsTemplate *model.NodeTLSTemplate, tlsClient map[string]interface{}) string {
	escapedTag := url.QueryEscape(tag)
	query := ""
	if tlsTemplate != nil {
		params := url.Values{}
		if strings.EqualFold(tlsTemplate.Type, "reality") {
			params.Set("security", "reality")
		} else {
			params.Set("security", "tls")
		}
		if sni := optString(tlsClient, "server_name", ""); sni != "" {
			params.Set("sni", sni)
		}
		if fp := optString(tlsClient, "utls_fingerprint", ""); fp != "" {
			params.Set("fp", fp)
		}
		if reality := asMap(tlsClient["reality"]); len(reality) > 0 {
			if sid, ok := reality["short_id"].(string); ok && sid != "" {
				params.Set("sid", sid)
			}
			if pk, ok := reality["public_key"].(string); ok && pk != "" {
				params.Set("pbk", pk)
			}
		}
		query = params.Encode()
	}
	switch protocol {
	case "vless":
		if query != "" {
			query = "encryption=none&" + query
		} else {
			query = "encryption=none"
		}
		return fmt.Sprintf("vless://%s@%s:%d?%s#%s", identity.UUID, host, port, query, escapedTag)
	case "trojan":
		if query != "" {
			return fmt.Sprintf("trojan://%s@%s:%d?%s#%s", url.QueryEscape(identity.TrojanPassword), host, port, query, escapedTag)
		}
		return fmt.Sprintf("trojan://%s@%s:%d#%s", url.QueryEscape(identity.TrojanPassword), host, port, escapedTag)
	case "hysteria2":
		if query != "" {
			return fmt.Sprintf("hysteria2://%s@%s:%d?%s#%s", url.QueryEscape(identity.Hysteria2Password), host, port, query, escapedTag)
		}
		return fmt.Sprintf("hysteria2://%s@%s:%d#%s", url.QueryEscape(identity.Hysteria2Password), host, port, escapedTag)
	case "tuic":
		return fmt.Sprintf("tuic://%s:%s@%s:%d#%s", identity.TUICUUID, url.QueryEscape(identity.TUICPassword), host, port, escapedTag)
	case "shadowsocks":
		auth := base64.RawURLEncoding.EncodeToString([]byte("2022-blake3-aes-128-gcm:" + identity.MixedPassword))
		return fmt.Sprintf("ss://%s@%s:%d#%s", auth, host, port, escapedTag)
	default:
		return ""
	}
}

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func protocolPassword(protocol string, identity *model.NodeIdentity) string {
	switch protocol {
	case "trojan":
		return identity.TrojanPassword
	case "hysteria2":
		return identity.Hysteria2Password
	case "tuic":
		return identity.TUICPassword
	default:
		return identity.MixedPassword
	}
}

func normalizeJSONText(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "{}"
	}
	return string(normalizeJSONBytes([]byte(v)))
}

func normalizeJSONBytes(v []byte) []byte {
	v = []byte(strings.TrimSpace(string(v)))
	if len(v) == 0 || string(v) == "null" {
		return []byte("{}")
	}
	var any interface{}
	if err := json.Unmarshal(v, &any); err != nil {
		return []byte("{}")
	}
	raw, _ := json.Marshal(any)
	return raw
}

func nullableString(v string) sql.NullString {
	v = strings.TrimSpace(v)
	return sql.NullString{String: v, Valid: v != ""}
}

func nullableInt64FromValue(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v > 0}
}

func nullableToString(v sql.NullString) string {
	if v.Valid {
		return v.String
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return "127.0.0.1"
}

func optString(m map[string]interface{}, key, fallback string) string {
	if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func randomPort() int {
	n, _ := rand.Int(rand.Reader, big.NewInt(50000))
	return int(n.Int64()) + 10000
}

func randomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func deployRandomToken(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, n)
	for i := range out {
		v, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		out[i] = alphabet[v.Int64()]
	}
	return string(out)
}
