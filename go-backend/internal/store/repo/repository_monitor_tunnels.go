package repo

import (
	"errors"

	"go-backend/internal/store/model"
)

func (r *Repository) ListMonitorTunnels() ([]model.Tunnel, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	var tunnels []model.Tunnel
	err := r.db.Select("id", "inx", "name", "status", "updated_time").
		Order("inx ASC, id ASC").
		Find(&tunnels).Error
	return tunnels, err
}

func (r *Repository) ListMonitorTunnelsByIDs(tunnelIDs []int64) ([]model.Tunnel, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	if len(tunnelIDs) == 0 {
		return nil, nil
	}
	var tunnels []model.Tunnel
	err := r.db.Select("id", "inx", "name", "status", "updated_time").
		Where("id IN ?", tunnelIDs).
		Order("inx ASC, id ASC").
		Find(&tunnels).Error
	return tunnels, err
}

func (r *Repository) ListUserTunnelIDSet(userID int64) ([]int64, error) {
	if r == nil || r.db == nil {
		return nil, errors.New("repository not initialized")
	}
	if userID <= 0 {
		return nil, nil
	}
	var ids []int64
	err := r.db.Model(&model.UserTunnel{}).
		Where("user_id = ? AND status = 1", userID).
		Pluck("tunnel_id", &ids).Error
	return ids, err
}
