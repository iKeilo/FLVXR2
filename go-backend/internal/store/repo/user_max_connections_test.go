package repo

import (
	"path/filepath"
	"testing"
	"time"

	"go-backend/internal/store/model"
)

func TestUserMaxConnectionsCreateUpdateAndList(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer r.Close()

	now := time.Now().UnixMilli()
	userID, err := r.CreateUser("limit-user", "pwd", 1, now+86400000, 100, 1, 10, 77, 1, now, 0, 0, 0, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	user, err := r.GetUserByID(userID)
	if err != nil {
		t.Fatalf("get user: %v", err)
	}
	if user == nil || user.MaxConnections != 77 {
		t.Fatalf("expected max connections 77, got %#v", user)
	}

	users, err := r.ListUsers()
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	found := false
	for _, item := range users {
		if item["id"] == userID {
			found = true
			if got := item["maxConnections"]; got != 77 {
				t.Fatalf("expected list maxConnections 77, got %#v", got)
			}
		}
	}
	if !found {
		t.Fatalf("created user not found in list")
	}

	if err := r.UpdateUserWithoutPassword(userID, "limit-user", "remark", 100, 10, 0, now+86400000, 1, 1, now, 0, 0, 0, nil); err != nil {
		t.Fatalf("update user: %v", err)
	}
	user, err = r.GetUserByID(userID)
	if err != nil {
		t.Fatalf("get updated user: %v", err)
	}
	if user == nil || user.MaxConnections != 0 {
		t.Fatalf("expected max connections reset to 0, got %#v", user)
	}
}

func TestForwardRecordIncludesUserMaxConnections(t *testing.T) {
	r, err := Open(filepath.Join(t.TempDir(), "panel.db"))
	if err != nil {
		t.Fatalf("open repo: %v", err)
	}
	defer r.Close()

	now := time.Now().UnixMilli()
	userID, err := r.CreateUser("forward-limit-user", "pwd", 1, now+86400000, 100, 1, 10, 64, 1, now, 0, 0, 0, nil)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	forward := model.Forward{
		UserID:         userID,
		UserName:       "forward-limit-user",
		Name:           "forward-limit",
		TunnelID:       1,
		RemoteAddr:     "127.0.0.1:80",
		Strategy:       "fifo",
		CreatedTime:    now,
		UpdatedTime:    now,
		Status:         1,
		Inx:            1,
		MaxConnections: 0,
	}
	if err := r.DB().Create(&forward).Error; err != nil {
		t.Fatalf("create forward: %v", err)
	}

	record, err := r.GetForwardRecord(forward.ID)
	if err != nil {
		t.Fatalf("get forward record: %v", err)
	}
	if record == nil {
		t.Fatalf("expected forward record")
	}
	if record.UserMaxConnections != 64 {
		t.Fatalf("expected user max connections 64, got %d", record.UserMaxConnections)
	}
}
