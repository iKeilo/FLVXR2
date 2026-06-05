package repo

import (
	"errors"
	"fmt"
	"time"

	"go-backend/internal/store/model"
	"gorm.io/gorm"
)

type PathTunnelDetail struct {
	Path      model.PathTunnel            `json:"path"`
	Segments  []model.PathSegment         `json:"segments"`
	Resources []model.NodeRuntimeResource `json:"resources,omitempty"`
	Runtime   *model.PathRuntimeVersion   `json:"runtime,omitempty"`
}

func (r *Repository) GetWGNodeIdentity(nodeID int64) (*model.WGNodeIdentity, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var item model.WGNodeIdentity
	err := r.db.Where("node_id = ?", nodeID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) SaveWGNodeIdentity(item *model.WGNodeIdentity) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	if item == nil || item.NodeID <= 0 {
		return errors.New("invalid wireguard identity")
	}
	now := time.Now().UnixMilli()
	item.UpdatedTime = now
	if item.CreatedTime == 0 {
		item.CreatedTime = now
	}
	var existing model.WGNodeIdentity
	err := r.db.Where("node_id = ?", item.NodeID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.db.Create(item).Error
	}
	if err != nil {
		return err
	}
	item.ID = existing.ID
	item.CreatedTime = existing.CreatedTime
	return r.db.Save(item).Error
}

func (r *Repository) ListPathTunnels() ([]model.PathTunnel, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var items []model.PathTunnel
	err := r.db.Order("id DESC").Find(&items).Error
	return items, err
}

func (r *Repository) GetPathTunnelDetail(pathID int64) (*PathTunnelDetail, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var path model.PathTunnel
	err := r.db.Where("id = ?", pathID).First(&path).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var segments []model.PathSegment
	if err := r.db.Where("path_id = ?", pathID).Order("sequence ASC").Find(&segments).Error; err != nil {
		return nil, err
	}
	var resources []model.NodeRuntimeResource
	if err := r.db.Where("owner_type = ? AND owner_id = ?", "path", pathID).Order("node_id ASC, resource_type ASC").Find(&resources).Error; err != nil {
		return nil, err
	}
	var runtime model.PathRuntimeVersion
	runtimePtr := (*model.PathRuntimeVersion)(nil)
	if err := r.db.Where("path_id = ?", pathID).Order("version DESC").First(&runtime).Error; err == nil {
		runtimePtr = &runtime
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &PathTunnelDetail{Path: path, Segments: segments, Resources: resources, Runtime: runtimePtr}, nil
}

func (r *Repository) CreatePathTunnel(path *model.PathTunnel, segments []model.PathSegment, resources []model.NodeRuntimeResource) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	if path == nil || len(segments) == 0 {
		return errors.New("invalid path tunnel")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		path.CreatedTime = now
		path.UpdatedTime = now
		if path.Status == "" {
			path.Status = "pending"
		}
		if path.Transport == "" {
			path.Transport = "wireguard"
		}
		if err := tx.Create(path).Error; err != nil {
			return err
		}
		for i := range segments {
			segments[i].PathID = path.ID
			segments[i].CreatedTime = now
			segments[i].UpdatedTime = now
			if segments[i].Status == "" {
				segments[i].Status = "pending"
			}
			if err := tx.Create(&segments[i]).Error; err != nil {
				return err
			}
		}
		for i := range resources {
			normalizePathRuntimeResource(path.ID, &resources[i])
			resources[i].OwnerType = "path"
			resources[i].OwnerID = path.ID
			resources[i].CreatedTime = now
			resources[i].UpdatedTime = now
			if resources[i].Status == "" {
				resources[i].Status = "active"
			}
			if conflict, err := findRuntimeResourceConflict(tx, resources[i]); err != nil {
				return err
			} else if conflict != nil {
				return fmt.Errorf("node %d resource conflict: %s %s", resources[i].NodeID, resources[i].ResourceType, resources[i].ResourceKey)
			}
			if err := tx.Create(&resources[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) UpdatePathTunnel(path *model.PathTunnel, segments []model.PathSegment, resources []model.NodeRuntimeResource) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	if path == nil || path.ID <= 0 || len(segments) == 0 {
		return errors.New("invalid path tunnel")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		updates := map[string]interface{}{
			"name":            path.Name,
			"transport":       path.Transport,
			"tunnel_group_id": path.TunnelGroupID,
			"flow":            path.Flow,
			"traffic_ratio":   path.TrafficRatio,
			"remark":          path.Remark,
			"updated_time":    now,
		}
		if path.Status != "" {
			updates["status"] = path.Status
		}
		if err := tx.Model(&model.PathTunnel{}).Where("id = ?", path.ID).Updates(updates).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.PathSegment{}, "path_id = ?", path.ID).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.NodeRuntimeResource{}).
			Where("owner_type = ? AND owner_id = ?", "path", path.ID).
			Updates(map[string]interface{}{"status": "released", "updated_time": now}).Error; err != nil {
			return err
		}
		for i := range segments {
			segments[i].PathID = path.ID
			segments[i].CreatedTime = now
			segments[i].UpdatedTime = now
			if segments[i].Status == "" {
				segments[i].Status = "pending"
			}
			if err := tx.Create(&segments[i]).Error; err != nil {
				return err
			}
		}
		for i := range resources {
			normalizePathRuntimeResource(path.ID, &resources[i])
			resources[i].OwnerType = "path"
			resources[i].OwnerID = path.ID
			resources[i].CreatedTime = now
			resources[i].UpdatedTime = now
			if resources[i].Status == "" {
				resources[i].Status = "active"
			}
			if conflict, err := findRuntimeResourceConflict(tx, resources[i]); err != nil {
				return err
			} else if conflict != nil {
				return fmt.Errorf("node %d resource conflict: %s %s", resources[i].NodeID, resources[i].ResourceType, resources[i].ResourceKey)
			}
			if err := tx.Create(&resources[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func normalizePathRuntimeResource(pathID int64, item *model.NodeRuntimeResource) {
	if item == nil || pathID <= 0 {
		return
	}
	switch item.ResourceType {
	case "wireguard_interface":
		if item.ResourceKey == "" || item.ResourceKey == "__path_interface__" {
			item.ResourceKey = fmt.Sprintf("wg-flvx-%d", pathID)
		}
	case "route_table":
		if item.ResourceKey == "" || item.ResourceKey == "__path_table__" {
			item.ResourceKey = fmt.Sprintf("%d", 100000+pathID)
		}
	case "fwmark":
		if item.ResourceKey == "" || item.ResourceKey == "__path_fwmark__" {
			item.ResourceKey = fmt.Sprintf("0x%x", 0x10000000+pathID)
		}
	case "nft_chain":
		if item.ResourceKey == "" || item.ResourceKey == "__path_chain__" {
			item.ResourceKey = fmt.Sprintf("flvx_wg_path_%d", pathID)
		}
	}
}

func findRuntimeResourceConflict(tx *gorm.DB, item model.NodeRuntimeResource) (*model.NodeRuntimeResource, error) {
	var existing model.NodeRuntimeResource
	q := tx.Where("node_id = ? AND resource_type = ? AND status != ?", item.NodeID, item.ResourceType, "released")
	if item.ResourceType == "port" {
		q = q.Where("protocol = ? AND port = ?", item.Protocol, item.Port)
	} else {
		q = q.Where("resource_key = ?", item.ResourceKey)
	}
	err := q.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if existing.OwnerType == item.OwnerType && existing.OwnerID == item.OwnerID {
		return nil, nil
	}
	return &existing, nil
}

func (r *Repository) UpdatePathTunnelStatus(pathID int64, status string) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	return r.db.Model(&model.PathTunnel{}).Where("id = ?", pathID).Updates(map[string]interface{}{
		"status":       status,
		"updated_time": time.Now().UnixMilli(),
	}).Error
}

func (r *Repository) DeletePathTunnel(pathID int64) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UnixMilli()
		if err := tx.Model(&model.NodeRuntimeResource{}).
			Where("owner_type = ? AND owner_id = ?", "path", pathID).
			Updates(map[string]interface{}{"status": "released", "updated_time": now}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.PathSegment{}, "path_id = ?", pathID).Error; err != nil {
			return err
		}
		if err := tx.Delete(&model.PathRuntimeVersion{}, "path_id = ?", pathID).Error; err != nil {
			return err
		}
		return tx.Delete(&model.PathTunnel{}, "id = ?", pathID).Error
	})
}

func (r *Repository) CreatePathRuntimeVersion(item *model.PathRuntimeVersion) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	if item == nil || item.PathID <= 0 {
		return errors.New("invalid path runtime version")
	}
	var maxVersion int
	_ = r.db.Model(&model.PathRuntimeVersion{}).Where("path_id = ?", item.PathID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error
	item.Version = maxVersion + 1
	item.CreatedTime = time.Now().UnixMilli()
	if item.Status == "" {
		item.Status = "pending"
	}
	return r.db.Create(item).Error
}

func (r *Repository) UpdatePathRuntimeVersion(id int64, status, actualHash, message string) error {
	if r == nil || r.db == nil {
		return errors.New("repository not initialized")
	}
	return r.db.Model(&model.PathRuntimeVersion{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      status,
		"actual_hash": actualHash,
		"message":     message,
	}).Error
}
