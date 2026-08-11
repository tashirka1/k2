package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/tashirka1/k2/internal/auth/model"
)

type AuthStorage interface {
	GetUser(ctx context.Context, username string) (model.AuthUser, error)
	GetFirstUser(ctx context.Context) (model.AuthUser, error)
	UserExists(ctx context.Context) (bool, error)
	UpdateAttempts(ctx context.Context, username string, attempts int, lockedUntil *time.Time) error
	ResetAttempts(ctx context.Context, username string) error
	CreateInitialUser(ctx context.Context, username string, password string) error
	UpsertUser(ctx context.Context, username string, password string) error
}

type Auth struct {
	db *sql.DB
}

func NewAuth(db *sql.DB) *Auth {
	return &Auth{db: db}
}

func (r *Auth) UserExists(ctx context.Context) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM auth_user").Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Auth) GetUser(ctx context.Context, username string) (model.AuthUser, error) {
	u := model.AuthUser{}
	err := r.db.QueryRowContext(ctx, "SELECT id, username, password, login_attempts, COALESCE(locked_until, ''), created_at FROM auth_user WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.Password, &u.LoginAttempts, &u.LockedUntil, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AuthUser{}, model.ErrNotFound
	}
	if err != nil {
		return model.AuthUser{}, err
	}
	return u, nil
}

func (r *Auth) GetFirstUser(ctx context.Context) (model.AuthUser, error) {
	u := model.AuthUser{}
	err := r.db.QueryRowContext(ctx, "SELECT id, username, password, login_attempts, COALESCE(locked_until, ''), created_at FROM auth_user LIMIT 1").
		Scan(&u.ID, &u.Username, &u.Password, &u.LoginAttempts, &u.LockedUntil, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.AuthUser{}, model.ErrNotFound
	}
	if err != nil {
		return model.AuthUser{}, err
	}
	return u, nil
}

func (r *Auth) UpdateAttempts(ctx context.Context, username string, attempts int, lockedUntil *time.Time) error {
	var lu *string
	if lockedUntil != nil {
		s := lockedUntil.Format(time.RFC3339)
		lu = &s
	}
	_, err := r.db.ExecContext(ctx, "UPDATE auth_user SET login_attempts = ?, locked_until = ? WHERE username = ?", attempts, lu, username)
	return err
}

func (r *Auth) ResetAttempts(ctx context.Context, username string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE auth_user SET login_attempts = 0, locked_until = NULL WHERE username = ?", username)
	return err
}

func (r *Auth) CreateInitialUser(ctx context.Context, username string, password string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO auth_user(username, password) VALUES (?, ?)", username, password)
	return err
}

func (r *Auth) UpsertUser(ctx context.Context, username string, password string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auth_user(username, password) VALUES (?, ?)
		ON CONFLICT(username) DO UPDATE SET password = excluded.password, login_attempts = 0, locked_until = NULL`, username, password)
	return err
}
