package repo

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"go-backend/internal/store/model"

	"gorm.io/gorm"
)

// RemoteNodeRow holds the columns fetched for a remote node listing.
type RemoteNodeRow struct {
	ID           int64
	Name         string
	RemoteURL    sql.NullString
	RemoteToken  sql.NullString
	RemoteConfig sql.NullString
}

// NodeBasicInfo holds name, server_ip, and status for a node.
type NodeBasicInfo struct {
	Name     string
	ServerIP string
	Status   int
}

// FederationBindingRow holds the columns for an active federation tunnel binding.
type FederationBindingRow struct {
	ID              int64
	TunnelID        int64
	TunnelName      string
	ChainType       int
	HopInx          int
	AllocatedPort   int
	ResourceKey     string
	RemoteBindingID string
	UpdatedTime     int64
}

type ActiveForwardPortRow struct {
	ForwardID   int64
	ForwardName string
	TunnelID    int64
	TunnelName  string
	Port        int
	UpdatedTime int64
}

type RemoteNodeChainTunnelRow struct {
	TunnelID   int64
	TunnelName string
	ChainType  int
	HopInx     int
}

type RemoteNodeReferenceCounts struct {
	ChainTunnel              int64
	ForwardPort              int64
	ActiveFederationBindings int64
}

// ListRemoteNodes returns all nodes with is_remote=1, ordered by id desc.
func (r *Repository) ListRemoteNodes() ([]RemoteNodeRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var result []RemoteNodeRow
	err := r.db.Model(&model.Node{}).
		Select("id, name, remote_url, remote_token, remote_config").
		Where("is_remote = 1").
		Order("id DESC").
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = make([]RemoteNodeRow, 0)
	}
	return result, nil
}

// UpdateNodeRemoteConfig sets the remote_config JSON for a given node.
func (r *Repository) UpdateNodeRemoteConfig(nodeID int64, configJSON string) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	return r.db.Model(&model.Node{}).Where("id = ?", nodeID).Update("remote_config", configJSON).Error
}

func (r *Repository) CountRemoteNodeReferences(nodeID int64) (RemoteNodeReferenceCounts, error) {
	if r == nil || r.db == nil {
		return RemoteNodeReferenceCounts{}, errors.New("repository not initialized")
	}

	var counts RemoteNodeReferenceCounts
	if err := r.db.Model(&model.ChainTunnel{}).Where("node_id = ?", nodeID).Count(&counts.ChainTunnel).Error; err != nil {
		return counts, err
	}
	if err := r.db.Model(&model.ForwardPort{}).Where("node_id = ?", nodeID).Count(&counts.ForwardPort).Error; err != nil {
		return counts, err
	}
	if err := r.db.Model(&model.FederationTunnelBinding{}).
		Where("node_id = ? AND status = 1", nodeID).
		Count(&counts.ActiveFederationBindings).Error; err != nil {
		return counts, err
	}
	return counts, nil
}

// ListActiveBindingsForNode returns active federation tunnel bindings for a node.
func (r *Repository) ListActiveBindingsForNode(nodeID int64) ([]FederationBindingRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var result []FederationBindingRow
	err := r.db.Model(&model.FederationTunnelBinding{}).
		Select("federation_tunnel_binding.id, federation_tunnel_binding.tunnel_id, COALESCE(tunnel.name, '') AS tunnel_name, federation_tunnel_binding.chain_type, federation_tunnel_binding.hop_inx, federation_tunnel_binding.allocated_port, federation_tunnel_binding.resource_key, federation_tunnel_binding.remote_binding_id, federation_tunnel_binding.updated_time").
		Joins("LEFT JOIN tunnel ON tunnel.id = federation_tunnel_binding.tunnel_id").
		Where("federation_tunnel_binding.node_id = ? AND federation_tunnel_binding.status = 1", nodeID).
		Order("federation_tunnel_binding.allocated_port ASC, federation_tunnel_binding.id ASC").
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = make([]FederationBindingRow, 0)
	}
	return result, nil
}

func (r *Repository) ListActiveForwardPortsForNode(nodeID int64) ([]ActiveForwardPortRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var result []ActiveForwardPortRow
	err := r.db.Model(&model.ForwardPort{}).
		Select("forward_port.forward_id, COALESCE(forward.name, '') AS forward_name, forward.tunnel_id, COALESCE(tunnel.name, '') AS tunnel_name, forward_port.port, forward.updated_time").
		Joins("JOIN forward ON forward.id = forward_port.forward_id").
		Joins("LEFT JOIN tunnel ON tunnel.id = forward.tunnel_id").
		Where("forward_port.node_id = ? AND forward_port.port > 0", nodeID).
		Order("forward_port.port ASC, forward_port.id ASC").
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = make([]ActiveForwardPortRow, 0)
	}
	return result, nil
}

func (r *Repository) ListRemoteNodeChainTunnels(nodeID int64) ([]RemoteNodeChainTunnelRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	type row struct {
		TunnelID   int64
		TunnelName string
		ChainType  string
		HopInx     sql.NullInt64
	}
	var rows []row
	err := r.db.Model(&model.ChainTunnel{}).
		Select("chain_tunnel.tunnel_id, COALESCE(tunnel.name, '') AS tunnel_name, chain_tunnel.chain_type, chain_tunnel.inx AS hop_inx").
		Joins("LEFT JOIN tunnel ON tunnel.id = chain_tunnel.tunnel_id").
		Where("chain_tunnel.node_id = ?", nodeID).
		Order("chain_tunnel.tunnel_id ASC, chain_tunnel.chain_type ASC, chain_tunnel.inx ASC, chain_tunnel.id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]RemoteNodeChainTunnelRow, 0, len(rows))
	for _, item := range rows {
		chainType := 0
		if strings.TrimSpace(item.ChainType) != "" {
			if parsed, parseErr := strconv.Atoi(strings.TrimSpace(item.ChainType)); parseErr == nil {
				chainType = parsed
			}
		}
		hopInx := 0
		if item.HopInx.Valid {
			hopInx = int(item.HopInx.Int64)
		}
		result = append(result, RemoteNodeChainTunnelRow{
			TunnelID:   item.TunnelID,
			TunnelName: item.TunnelName,
			ChainType:  chainType,
			HopInx:     hopInx,
		})
	}
	return result, nil
}

func (r *Repository) ListForwardPreviewRowsByTunnel(tunnelID int64) ([]ActiveForwardPortRow, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var result []ActiveForwardPortRow
	err := r.db.Model(&model.Forward{}).
		Select("forward.id AS forward_id, COALESCE(forward.name, '') AS forward_name, forward.tunnel_id, COALESCE(tunnel.name, '') AS tunnel_name, COALESCE(MIN(forward_port.port), 0) AS port, forward.updated_time").
		Joins("LEFT JOIN tunnel ON tunnel.id = forward.tunnel_id").
		Joins("LEFT JOIN forward_port ON forward_port.forward_id = forward.id").
		Where("forward.tunnel_id = ?", tunnelID).
		Group("forward.id, forward.name, forward.tunnel_id, tunnel.name, forward.updated_time").
		Order("forward.id ASC").
		Find(&result).Error
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = make([]ActiveForwardPortRow, 0)
	}
	return result, nil
}

func (r *Repository) DeleteRemoteNodeMiddleReferences(nodeID int64, keepTunnelIDs []int64) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		q := tx.Where("node_id = ? AND chain_type = ?", nodeID, "2")
		if len(keepTunnelIDs) > 0 {
			q = q.Where("tunnel_id IN ?", keepTunnelIDs)
		}
		if err := q.Delete(&model.ChainTunnel{}).Error; err != nil {
			return err
		}
		qb := tx.Where("node_id = ? AND chain_type = ? AND status = 1", nodeID, 2)
		if len(keepTunnelIDs) > 0 {
			qb = qb.Where("tunnel_id IN ?", keepTunnelIDs)
		}
		return qb.Model(&model.FederationTunnelBinding{}).Updates(map[string]interface{}{
			"status":       0,
			"updated_time": time.Now().UnixMilli(),
		}).Error
	})
}

// GetNodeBasicInfo returns the name, server_ip, and status for a given node.
func (r *Repository) GetNodeBasicInfo(nodeID int64) (*NodeBasicInfo, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var n model.Node
	err := r.db.Select("name", "server_ip", "status").Where("id = ?", nodeID).First(&n).Error
	if err != nil {
		return nil, err
	}
	return &NodeBasicInfo{Name: n.Name, ServerIP: n.ServerIP, Status: n.Status}, nil
}

// CreateFederationTunnel creates a tunnel and chain_tunnel entry in a transaction,
// returning the new tunnel ID.
func (r *Repository) CreateFederationTunnel(name string, tunnelType int, protocol string, now int64, nodeID int64, remotePort int) (int64, error) {
	if r == nil || r.db == nil {
		return 0, errors.New("repository not initialized")
	}
	tunnel := model.Tunnel{
		Name:        name,
		Type:        tunnelType,
		Protocol:    protocol,
		Flow:        0,
		CreatedTime: now,
		UpdatedTime: now,
		Status:      1,
		InIP:        sql.NullString{String: "", Valid: false},
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&tunnel).Error; err != nil {
			return err
		}
		ct := model.ChainTunnel{
			TunnelID:  tunnel.ID,
			ChainType: "1",
			NodeID:    nodeID,
			Port:      sql.NullInt64{Int64: int64(remotePort), Valid: true},
			Strategy:  sql.NullString{String: "fifo", Valid: true},
			Inx:       sql.NullInt64{Int64: 0, Valid: true},
			Protocol:  sql.NullString{String: protocol, Valid: true},
		}
		if err := tx.Create(&ct).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return tunnel.ID, nil
}

// ListUsedPortsOnNode returns all ports in use on a given node from chain_tunnel and forward_port tables.
func (r *Repository) ListUsedPortsOnNode(nodeID int64) ([]int, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	used := make(map[int]struct{})

	var chainPorts []int
	err := r.db.Model(&model.ChainTunnel{}).
		Where("node_id = ? AND port > 0", nodeID).
		Pluck("port", &chainPorts).Error
	if err != nil {
		return nil, err
	}
	for _, p := range chainPorts {
		if p > 0 {
			used[p] = struct{}{}
		}
	}

	var forwardPorts []int
	err = r.db.Model(&model.ForwardPort{}).
		Where("node_id = ? AND port > 0", nodeID).
		Pluck("port", &forwardPorts).Error
	if err != nil {
		return nil, err
	}
	for _, p := range forwardPorts {
		if p > 0 {
			used[p] = struct{}{}
		}
	}

	result := make([]int, 0, len(used))
	for p := range used {
		result = append(result, p)
	}
	return result, nil
}

// ListTunnelIDsByNamePrefix returns all tunnel IDs whose name starts with the given prefix.
func (r *Repository) ListTunnelIDsByNamePrefix(prefix string) ([]int64, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var ids []int64
	err := r.db.Model(&model.Tunnel{}).
		Where("name LIKE ?", prefix+"%").
		Order("id ASC").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	if ids == nil {
		ids = make([]int64, 0)
	}
	return ids, nil
}

func (r *Repository) NextIndex(table string) int {
	if r == nil || r.db == nil {
		return 0
	}
	var modelRef interface{}
	switch table {
	case "node":
		modelRef = &model.Node{}
	case "tunnel":
		modelRef = &model.Tunnel{}
	case "forward":
		modelRef = &model.Forward{}
	default:
		return 0
	}

	type inxRow struct {
		Inx int
	}
	var row inxRow
	err := r.db.Model(modelRef).
		Select("inx").
		Order("inx ASC, id ASC").
		Limit(1).
		Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0
	}
	if err != nil {
		return 0
	}
	return row.Inx - 1
}

// CreateRemoteNode inserts a new remote node.
func (r *Repository) CreateRemoteNode(name, secret, serverIP, portRange string, now int64, status int, inx int, remoteURL, remoteToken, remoteConfigJSON string) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	node := model.Node{
		Name:          name,
		Secret:        secret,
		ServerIP:      serverIP,
		ServerIPV4:    sql.NullString{},
		ServerIPV6:    sql.NullString{},
		Port:          portRange,
		InterfaceName: sql.NullString{},
		Version:       sql.NullString{},
		HTTP:          0,
		TLS:           0,
		Socks:         0,
		CreatedTime:   now,
		UpdatedTime:   sql.NullInt64{Int64: now, Valid: true},
		Status:        status,
		TCPListenAddr: "[::]",
		UDPListenAddr: "[::]",
		Inx:           inx,
		IsRemote:      1,
		RemoteURL:     sql.NullString{String: remoteURL, Valid: remoteURL != ""},
		RemoteToken:   sql.NullString{String: remoteToken, Valid: remoteToken != ""},
		RemoteConfig:  sql.NullString{String: remoteConfigJSON, Valid: remoteConfigJSON != ""},
	}
	return r.db.Create(&node).Error
}
