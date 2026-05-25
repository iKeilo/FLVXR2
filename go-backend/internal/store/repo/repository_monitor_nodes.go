package repo

import (
	"errors"

	"go-backend/internal/store/model"
)

func (r *Repository) ListMonitorNodes() ([]model.Node, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var nodes []model.Node
	err := r.db.Select("id", "inx", "name", "status", "version", "updated_time").
		Where("is_remote = ?", 0).
		Order("inx ASC, id ASC").
		Find(&nodes).Error
	return nodes, err
}

func (r *Repository) ListMonitorNodesByIDs(nodeIDs []int64) ([]model.Node, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	if len(nodeIDs) == 0 {
		return nil, nil
	}
	var nodes []model.Node
	err := r.db.Select("id", "inx", "name", "status", "version", "updated_time").
		Where("is_remote = ? AND id IN ?", 0, nodeIDs).
		Order("inx ASC, id ASC").
		Find(&nodes).Error
	return nodes, err
}
