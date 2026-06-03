package handler

import (
	"crypto/ecdh"
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

const coreDeployTimeout = 180 * time.Second

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

func (h *Handler) nodeTLSRealityKeypair(w http.ResponseWriter, r *http.Request) {
	privateKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	publicKey := privateKey.PublicKey()
	response.WriteJSON(w, response.OK(map[string]string{
		"privateKey": base64.RawURLEncoding.EncodeToString(privateKey.Bytes()),
		"publicKey":  base64.RawURLEncoding.EncodeToString(publicKey.Bytes()),
	}))
}

func (h *Handler) nodeTLSRealityShortIDs(w http.ResponseWriter, r *http.Request) {
	response.WriteJSON(w, response.OK(map[string][]string{
		"shortIds": randomRealityShortIDs(),
	}))
}

func (h *Handler) ensureNodeDeployAccess(r *http.Request, nodeID int64) (*model.Node, error) {
	actorUserID, _, err := userRoleFromRequest(r)
	if err != nil {
		return nil, errors.New("无效的token或token已过期")
	}
	node, err := h.repo.GetNodeByID(nodeID)
	if err != nil || node == nil {
		return nil, errors.New("节点不存在")
	}
	if node.OwnerUserID <= 0 {
		if actorUserID == 1 {
			return node, nil
		}
		return nil, errors.New("无权部署该节点")
	}
	if node.OwnerUserID != actorUserID {
		return nil, errors.New("无权部署该节点，仅节点所属账号可以使用部署功能")
	}
	return node, nil
}

func (h *Handler) nodeDeployDetail(w http.ResponseWriter, r *http.Request) {
	var req nodeIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	node, err := h.ensureNodeDeployAccess(r, req.NodeID)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	identity, err := h.ensureNodeIdentity(req.NodeID, false)
	if err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
		return
	}
	inbounds, _ := h.repo.ListNodeDeployedInbounds(req.NodeID)
	for idx := range inbounds {
		if refreshed, err := h.refreshNodeInboundCopyFields(inbounds[idx], identity); err == nil {
			inbounds[idx] = refreshed
		}
	}
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

func (h *Handler) refreshNodeInboundCopyFields(item model.NodeDeployedInbound, identity *model.NodeIdentity) (model.NodeDeployedInbound, error) {
	var tlsTemplate *model.NodeTLSTemplate
	if item.TLSTemplateID.Valid {
		tlsTemplate, _ = h.repo.GetNodeTLSTemplate(item.TLSTemplateID.Int64)
	}
	serverConfig, clientConfig, shareURI, err := renderInboundConfig(
		strings.ToLower(strings.TrimSpace(item.Protocol)),
		item.InternalTag,
		item.ListenAddr,
		item.ListenPort,
		item.PublishAddr,
		item.PublishPort,
		[]byte(normalizeJSONText(item.InboundOptionsJSON)),
		identity,
		tlsTemplate,
	)
	if err != nil {
		return item, err
	}
	if item.ServerConfigJSON == serverConfig && item.ClientConfigJSON == clientConfig && item.ShareURI == shareURI {
		return item, nil
	}
	item.ServerConfigJSON = serverConfig
	item.ClientConfigJSON = clientConfig
	item.ShareURI = shareURI
	_ = h.repo.SaveNodeDeployedInbound(&item)
	return item, nil
}

func (h *Handler) nodeIdentityRegenerate(w http.ResponseWriter, r *http.Request) {
	var req nodeIDRequest
	if err := decodeJSON(r.Body, &req); err != nil || req.NodeID <= 0 {
		response.WriteJSON(w, response.ErrDefault("请求参数无效"))
		return
	}
	if _, err := h.ensureNodeDeployAccess(r, req.NodeID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
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
	if req.ID > 0 {
		existing, err := h.repo.GetNodeDeployedInbound(req.ID)
		if err != nil {
			response.WriteJSON(w, response.ErrDefault("部署入站不存在"))
			return
		}
		if existing.NodeID != req.NodeID {
			response.WriteJSON(w, response.ErrDefault("部署入站不属于该节点"))
			return
		}
	}
	if _, err := h.ensureNodeDeployAccess(r, req.NodeID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
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
	if _, err := h.ensureNodeDeployAccess(r, item.NodeID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
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
		}, coreDeployTimeout, false, false)
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
	if _, err := h.ensureNodeDeployAccess(r, req.NodeID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
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
	}, coreDeployTimeout, false, false)
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
	if _, err := h.ensureNodeDeployAccess(r, req.NodeID); err != nil {
		response.WriteJSON(w, response.ErrDefault(err.Error()))
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
	}, coreDeployTimeout, false, false)
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
		}, coreDeployTimeout, false, false)
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
		normalizeRenderedInbound(obj)
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

func normalizeRenderedInbound(inbound map[string]interface{}) {
	delete(inbound, "flow")
	if tls := asMap(inbound["tls"]); len(tls) > 0 {
		if normalizedTLS := normalizeServerTLS(tls); normalizedTLS != nil {
			inbound["tls"] = normalizedTLS
		} else {
			delete(inbound, "tls")
		}
	}
}

func normalizeServerTLS(tls map[string]interface{}) map[string]interface{} {
	if len(tls) == 0 {
		return nil
	}
	reality := asMap(tls["reality"])
	if enabled, ok := reality["enabled"].(bool); ok && enabled {
		tls["enabled"] = true
		return tls
	}
	acme := asMap(tls["acme"])
	if len(acme) > 0 {
		tls["enabled"] = true
		return tls
	}
	if hasTLSCertificate(tls) {
		tls["enabled"] = true
		return tls
	}
	return nil
}

func hasTLSCertificate(tls map[string]interface{}) bool {
	certPath := optString(tls, "certificate_path", "")
	keyPath := optString(tls, "key_path", "")
	if certPath != "" && keyPath != "" {
		return true
	}
	return hasNonEmptyValue(tls["certificate"]) && hasNonEmptyValue(tls["key"])
}

func hasNonEmptyValue(v interface{}) bool {
	switch raw := v.(type) {
	case string:
		return strings.TrimSpace(raw) != ""
	case []interface{}:
		return len(raw) > 0
	case []string:
		return len(raw) > 0
	default:
		return v != nil
	}
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
		if strings.EqualFold(k, "flow") ||
			strings.EqualFold(k, "network") ||
			strings.EqualFold(k, "packet_encoding") {
			continue
		}
		if strings.EqualFold(k, "multiplex") {
			if multiplex := normalizeInboundMultiplex(v); multiplex != nil {
				inbound["multiplex"] = multiplex
			}
			continue
		}
		inbound[k] = v
	}
	var tlsClient map[string]interface{}
	tlsActive := false
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
			if strings.TrimSpace(optString(clientReality, "short_id", "")) == "" {
				clientReality["short_id"] = pickRealityShortID(reality["short_id"], identity.RealityShortID)
			}
			tlsClient["reality"] = clientReality
		}
		if normalizedTLS := normalizeServerTLS(tlsServer); normalizedTLS != nil {
			inbound["tls"] = normalizedTLS
			tlsActive = true
		}
	}
	switch protocol {
	case "vless":
		user := map[string]interface{}{"uuid": identity.UUID}
		if flow := optString(opt, "flow", ""); tlsActive && flow != "" {
			user["flow"] = flow
		}
		inbound["users"] = []map[string]interface{}{user}
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
	client := buildClientOutbound(protocol, tag, publishAddr, publishPort, opt, identity, tlsClient, tlsActive)
	serverRaw, _ := json.MarshalIndent(inbound, "", "  ")
	clientRaw, _ := json.MarshalIndent(client, "", "  ")
	return string(serverRaw), string(clientRaw), shareURI(protocol, publishAddr, publishPort, identity, tag, tlsTemplate, tlsClient, opt, tlsActive), nil
}

func buildClientOutbound(protocol, tag, publishAddr string, publishPort int, opt map[string]interface{}, identity *model.NodeIdentity, tlsClient map[string]interface{}, tlsActive bool) map[string]interface{} {
	client := map[string]interface{}{
		"type":        protocol,
		"tag":         tag,
		"server":      publishAddr,
		"server_port": publishPort,
	}
	switch protocol {
	case "vless":
		client["uuid"] = identity.UUID
		if flow := optString(opt, "flow", ""); tlsActive && flow != "" {
			client["flow"] = flow
		}
		if network := optString(opt, "network", ""); network != "" {
			client["network"] = network
		}
		if packetEncoding := optString(opt, "packet_encoding", ""); packetEncoding != "" && packetEncoding != "none" {
			client["packet_encoding"] = packetEncoding
		}
	case "trojan":
		client["password"] = identity.TrojanPassword
	case "hysteria2":
		client["password"] = identity.Hysteria2Password
	case "tuic":
		client["uuid"] = identity.TUICUUID
		client["password"] = identity.TUICPassword
	case "shadowsocks":
		client["method"] = optString(opt, "method", "2022-blake3-aes-128-gcm")
		client["password"] = identity.MixedPassword
	case "mixed", "socks", "http":
		client["username"] = identity.UUID
		client["password"] = identity.MixedPassword
	}
	if len(tlsClient) > 0 {
		client["tls"] = tlsClient
	}
	if multiplex := normalizeClientMultiplex(opt["multiplex"]); multiplex != nil {
		client["multiplex"] = multiplex
	}
	if transport := asMap(opt["transport"]); len(transport) > 0 {
		client["transport"] = transport
	}
	return client
}

func normalizeInboundMultiplex(v interface{}) map[string]interface{} {
	multiplex := asMap(v)
	if len(multiplex) == 0 || !optBool(multiplex, "enabled", false) {
		return nil
	}
	normalized := map[string]interface{}{"enabled": true}
	if padding, ok := multiplex["padding"].(bool); ok {
		normalized["padding"] = padding
	}
	if brutal := asMap(multiplex["brutal"]); len(brutal) > 0 {
		normalized["brutal"] = brutal
	}
	return normalized
}

func normalizeClientMultiplex(v interface{}) map[string]interface{} {
	multiplex := asMap(v)
	if len(multiplex) == 0 || !optBool(multiplex, "enabled", false) {
		return nil
	}
	normalized := map[string]interface{}{"enabled": true}
	if protocol := optString(multiplex, "protocol", "smux"); protocol != "" {
		normalized["protocol"] = protocol
	}
	copyOptionalNumber(normalized, multiplex, "max_connections")
	copyOptionalNumber(normalized, multiplex, "min_streams")
	copyOptionalNumber(normalized, multiplex, "max_streams")
	if padding, ok := multiplex["padding"].(bool); ok {
		normalized["padding"] = padding
	}
	if brutal := asMap(multiplex["brutal"]); len(brutal) > 0 {
		normalized["brutal"] = brutal
	}
	return normalized
}

func copyOptionalNumber(dst, src map[string]interface{}, key string) {
	switch value := src[key].(type) {
	case float64:
		if value > 0 {
			dst[key] = int(value)
		}
	case int:
		if value > 0 {
			dst[key] = value
		}
	case int64:
		if value > 0 {
			dst[key] = value
		}
	}
}

func shareURI(protocol, host string, port int, identity *model.NodeIdentity, tag string, tlsTemplate *model.NodeTLSTemplate, tlsClient map[string]interface{}, opt map[string]interface{}, tlsActive bool) string {
	escapedTag := url.QueryEscape(tag)
	params := url.Values{}
	addTransportLinkParams(params, asMap(opt["transport"]))
	if tlsTemplate != nil && tlsActive {
		if strings.EqualFold(tlsTemplate.Type, "reality") {
			params.Set("security", "reality")
		} else {
			params.Set("security", "tls")
		}
		if sni := optString(tlsClient, "server_name", ""); sni != "" {
			params.Set("sni", sni)
		}
		fp := optString(tlsClient, "utls_fingerprint", "")
		if fp == "" {
			fp = optString(asMap(tlsClient["utls"]), "fingerprint", "")
		}
		if fp != "" {
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
	}
	if packetEncoding := optString(opt, "packet_encoding", ""); packetEncoding != "" && packetEncoding != "none" {
		params.Set("packetEncoding", packetEncoding)
	}
	if flow := optString(opt, "flow", ""); tlsActive && flow != "" {
		params.Set("flow", flow)
	}
	query := params.Encode()
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

func addTransportLinkParams(params url.Values, transport map[string]interface{}) {
	transportType := optString(transport, "type", "tcp")
	params.Set("type", transportType)
	switch transportType {
	case "http":
		if host, ok := transport["host"].([]interface{}); ok && len(host) > 0 {
			hosts := make([]string, 0, len(host))
			for _, item := range host {
				if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
					hosts = append(hosts, s)
				}
			}
			if len(hosts) > 0 {
				params.Set("host", strings.Join(hosts, ","))
			}
		}
		if path := optString(transport, "path", ""); path != "" {
			params.Set("path", path)
		}
	case "ws":
		if path := optString(transport, "path", ""); path != "" {
			params.Set("path", path)
		}
		if host := optString(asMap(transport["headers"]), "Host", ""); host != "" {
			params.Set("host", host)
		}
	case "grpc":
		if serviceName := optString(transport, "service_name", ""); serviceName != "" {
			params.Set("serviceName", serviceName)
		}
	case "httpupgrade":
		if host := optString(transport, "host", ""); host != "" {
			params.Set("host", host)
		}
		if path := optString(transport, "path", ""); path != "" {
			params.Set("path", path)
		}
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

func optBool(m map[string]interface{}, key string, fallback bool) bool {
	if v, ok := m[key].(bool); ok {
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

func randomRealityShortIDs() []string {
	shortIDs := make([]string, 24)
	for i := 1; i < len(shortIDs); i++ {
		byteCount, _ := rand.Int(rand.Reader, big.NewInt(8))
		shortIDs[i] = randomHex(int(byteCount.Int64()) + 1)
	}
	return shortIDs
}

func pickRealityShortID(v interface{}, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	switch raw := v.(type) {
	case []interface{}:
		values := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				values = append(values, s)
			}
		}
		if len(values) > 0 {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(values))))
			return values[int(n.Int64())]
		}
	case []string:
		values := make([]string, 0, len(raw))
		for _, item := range raw {
			if strings.TrimSpace(item) != "" {
				values = append(values, item)
			}
		}
		if len(values) > 0 {
			n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(values))))
			return values[int(n.Int64())]
		}
	case string:
		if strings.TrimSpace(raw) != "" {
			return raw
		}
	}
	if fallback == "" {
		return randomHex(8)
	}
	return fallback
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
