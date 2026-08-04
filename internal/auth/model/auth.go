package model

import "errors"

var (
	ErrNotFound     = errors.New("auth: not found")
	ErrInvalidLogin = errors.New("auth: invalid username or password")
	ErrLockedOut    = errors.New("auth: account locked out")
)

type AuthUser struct {
	Username      string
	Password      string
	LockedUntil   string
	CreatedAt     string
	ID            int
	LoginAttempts int
}
