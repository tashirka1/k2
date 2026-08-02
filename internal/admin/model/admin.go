package model

import "errors"

var (
	ErrNotFound     = errors.New("admin: not found")
	ErrInvalidLogin = errors.New("admin: invalid username or password")
	ErrLockedOut    = errors.New("admin: account locked out")
)

type AdminUser struct {
	Username      string
	Password      string
	LockedUntil   string
	CreatedAt     string
	ID            int
	LoginAttempts int
}
