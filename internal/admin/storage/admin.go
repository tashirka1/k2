package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/tashirka1/k2/internal/admin/model"
)

type AdminStorage interface {
	GetUser(ctx context.Context, username string) (model.AdminUser, error)
	GetFirstUser(ctx context.Context) (model.AdminUser, error)
	UserExists(ctx context.Context) (bool, error)
	UpdateAttempts(ctx context.Context, username string, attempts int, lockedUntil *time.Time) error
	ResetAttempts(ctx context.Context, username string) error
	CreateInitialUser(ctx context.Context, username string, password string) error
}

type Admin struct {
	db *sql.DB
}

func NewAdmin(db *sql.DB) *Admin {
	return &Admin{db: db}
}

func (r *Admin) UserExists(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admin_user").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Admin) GetUser(ctx context.Context, username string) (model.AdminUser, error) {
	u := model.AdminUser{}
	err := r.db.QueryRowContext(ctx, "SELECT id, username, password, login_attempts, COALESCE(locked_until, ''), created_at FROM admin_user WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.Password, &u.LoginAttempts, &u.LockedUntil, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AdminUser{}, model.ErrNotFound
	}
	if err != nil {
		return model.AdminUser{}, err
	}
	return u, nil
}

func (r *Admin) GetFirstUser(ctx context.Context) (model.AdminUser, error) {
	u := model.AdminUser{}
	err := r.db.QueryRowContext(ctx, "SELECT id, username, password, login_attempts, COALESCE(locked_until, ''), created_at FROM admin_user LIMIT 1").
		Scan(&u.ID, &u.Username, &u.Password, &u.LoginAttempts, &u.LockedUntil, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AdminUser{}, model.ErrNotFound
	}
	if err != nil {
		return model.AdminUser{}, err
	}
	return u, nil
}

func (r *Admin) UpdateAttempts(ctx context.Context, username string, attempts int, lockedUntil *time.Time) error {
	var lu interface{}
	if lockedUntil != nil {
		lu = lockedUntil.Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, "UPDATE admin_user SET login_attempts = ?, locked_until = ? WHERE username = ?", attempts, lu, username)
	return err
}

func (r *Admin) ResetAttempts(ctx context.Context, username string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE admin_user SET login_attempts = 0, locked_until = NULL WHERE username = ?", username)
	return err
}

func (r *Admin) CreateInitialUser(ctx context.Context, username string, password string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO admin_user(username, password) VALUES (?, ?)", username, password)
	return err
}
