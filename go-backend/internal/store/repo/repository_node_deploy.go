package repo

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-backend/internal/store/model"
)

func (r *Repository) ListNodeTLSTemplates() ([]model.NodeTLSTemplate, error) {
	var items []model.NodeTLSTemplate
	err := r.db.Order("id desc").Find(&items).Error
	if err != nil {
		return items, err
	}
	for idx := range items {
		items[idx].UsageCount, _ = r.CountNodeTLSUsage(items[idx].ID)
	}
	return items, err
}

func (r *Repository) SaveNodeTLSTemplate(item *model.NodeTLSTemplate) error {
	now := time.Now().UnixMilli()
	item.Name = strings.TrimSpace(item.Name)
	item.Type = strings.ToLower(strings.TrimSpace(item.Type))
	if item.Name == "" {
		return errors.New("TLS模板名称不能为空")
	}
	if item.Type == "" {
		item.Type = "tls"
	}
	if item.ServerJSON == "" {
		item.ServerJSON = "{}"
	}
	if item.ClientJSON == "" {
		item.ClientJSON = "{}"
	}
	if item.ID <= 0 {
		item.CreatedTime = now
	}
	item.UpdatedTime = now
	return r.db.Save(item).Error
}

func (r *Repository) DeleteNodeTLSTemplate(id int64) error {
	if id <= 0 {
		return errors.New("TLS模板ID无效")
	}
	count, err := r.CountNodeTLSUsage(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("TLS template is used by deployed inbounds")
	}
	return r.db.Delete(&model.NodeTLSTemplate{}, id).Error
}

func (r *Repository) CountNodeTLSUsage(id int64) (int64, error) {
	if id <= 0 {
		return 0, nil
	}
	var count int64
	err := r.db.Model(&model.NodeDeployedInbound{}).
		Where("tls_template_id = ?", id).
		Count(&count).Error
	return count, err
}

func (r *Repository) GetNodeTLSTemplate(id int64) (*model.NodeTLSTemplate, error) {
	if id <= 0 {
		return nil, nil
	}
	var item model.NodeTLSTemplate
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) GetNodeIdentity(nodeID int64) (*model.NodeIdentity, error) {
	var item model.NodeIdentity
	if err := r.db.Where("node_id = ?", nodeID).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) SaveNodeIdentity(item *model.NodeIdentity) error {
	now := time.Now().UnixMilli()
	if item.NodeID <= 0 {
		return errors.New("节点ID无效")
	}
	if item.CreatedTime <= 0 {
		item.CreatedTime = now
	}
	item.UpdatedTime = now
	return r.db.Save(item).Error
}

func (r *Repository) ListNodeDeployedInbounds(nodeID int64) ([]model.NodeDeployedInbound, error) {
	var items []model.NodeDeployedInbound
	err := r.db.Where("node_id = ?", nodeID).Order("id desc").Find(&items).Error
	return items, err
}

func (r *Repository) GetNodeDeployedInbound(id int64) (*model.NodeDeployedInbound, error) {
	var item model.NodeDeployedInbound
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) SaveNodeDeployedInbound(item *model.NodeDeployedInbound) error {
	now := time.Now().UnixMilli()
	if item.NodeID <= 0 {
		return errors.New("节点ID无效")
	}
	if strings.TrimSpace(item.DisplayName) == "" {
		return errors.New("部署名称不能为空")
	}
	if item.CreatedTime <= 0 {
		item.CreatedTime = now
	}
	item.UpdatedTime = now
	return r.db.Save(item).Error
}

func (r *Repository) DeleteNodeDeployedInbound(id int64) error {
	return r.db.Delete(&model.NodeDeployedInbound{}, id).Error
}

func (r *Repository) NextNodeInboundDisplayName(nodeID int64, baseName string, excludeID int64) (string, error) {
	baseName = strings.TrimSpace(baseName)
	if baseName == "" {
		return "", errors.New("部署名称不能为空")
	}
	candidate := baseName
	for idx := 1; idx < 1000; idx++ {
		var count int64
		query := r.db.Model(&model.NodeDeployedInbound{}).Where("node_id = ? AND display_name = ?", nodeID, candidate)
		if excludeID > 0 {
			query = query.Where("id <> ?", excludeID)
		}
		if err := query.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", baseName, idx+1)
	}
	return "", errors.New("无法生成不重复的部署名称")
}

func (r *Repository) CreateNodeConfigRevision(item *model.NodeConfigRevision) error {
	item.CreatedTime = time.Now().UnixMilli()
	return r.db.Create(item).Error
}

func (r *Repository) ListNodeConfigRevisions(nodeID int64, limit int) ([]model.NodeConfigRevision, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var items []model.NodeConfigRevision
	err := r.db.Where("node_id = ?", nodeID).Order("id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repository) GetNodeConfigRevision(id int64) (*model.NodeConfigRevision, error) {
	var item model.NodeConfigRevision
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *Repository) UpdateNodeConfigRevisionStatus(id int64, status, errMsg string) error {
	return r.db.Model(&model.NodeConfigRevision{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":        status,
		"error_message": errMsg,
	}).Error
}

func (r *Repository) ListNodeDeployLogs(nodeID int64, limit int) ([]model.NodeDeployLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var items []model.NodeDeployLog
	err := r.db.Where("node_id = ?", nodeID).Order("id desc").Limit(limit).Find(&items).Error
	return items, err
}

func (r *Repository) CreateNodeDeployLog(nodeID, revisionID int64, action, status, message string) error {
	return r.db.Create(&model.NodeDeployLog{
		NodeID:      nodeID,
		RevisionID:  revisionID,
		Action:      action,
		Status:      status,
		Message:     message,
		CreatedTime: time.Now().UnixMilli(),
	}).Error
}

func nullableTemplateID(id int64) sql.NullInt64 {
	if id <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: id, Valid: true}
}
