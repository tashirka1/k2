package model

import "errors"

var (
	ErrNotFound     = errors.New("admin: not found")
	ErrInvalidLogin = errors.New("admin: invalid username or password")
	ErrLockedOut    = errors.New("admin: account locked out")
)

type AdminUser struct {
	ID            int
	Username      string
	Password      string
	LoginAttempts int
	LockedUntil   string
	CreatedAt     string
}
