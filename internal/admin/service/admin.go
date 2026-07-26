package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/tashirka1/k2/internal/admin/model"
	"github.com/tashirka1/k2/internal/admin/storage"
)

type AdminService interface {
	EnsureCredentials(ctx context.Context) (username string, password string, err error)
	CheckLogin(ctx context.Context, username string, password string, now time.Time) (bool, error)
	GetCredentials(ctx context.Context) (username string, password string, err error)
}

type Admin struct {
	r storage.AdminStorage
}

func NewAdmin(r storage.AdminStorage) *Admin {
	return &Admin{r: r}
}

func (s *Admin) EnsureCredentials(ctx context.Context) (string, string, error) {
	exists, err := s.r.UserExists(ctx)
	if err != nil {
		return "", "", err
	}
	if exists {
		return s.GetCredentials(ctx)
	}
	return s.createCredentials(ctx)
}

func (s *Admin) createCredentials(ctx context.Context) (string, string, error) {
	username, err := randomWords(2)
	if err != nil {
		return "", "", fmt.Errorf("generate username: %w", err)
	}
	password, err := randomPassword(16)
	if err != nil {
		return "", "", fmt.Errorf("generate password: %w", err)
	}
	if err := s.r.CreateInitialUser(ctx, username, password); err != nil {
		return "", "", fmt.Errorf("create initial user: %w", err)
	}
	return username, password, nil
}

func (s *Admin) CheckLogin(ctx context.Context, username string, password string, now time.Time) (bool, error) {
	user, err := s.r.GetUser(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return false, model.ErrInvalidLogin
		}
		return false, err
	}

	if user.LockedUntil != "" {
		lockedUntil, err := time.Parse(time.RFC3339, user.LockedUntil)
		if err == nil && now.Before(lockedUntil) {
			return false, model.ErrLockedOut
		}
	}

	if user.Password != password {
		attempts := user.LoginAttempts + 1
		var lockedUntil *time.Time
		if attempts >= 3 {
			t := now.Add(1 * time.Minute)
			lockedUntil = &t
		}
		if err := s.r.UpdateAttempts(ctx, username, attempts, lockedUntil); err != nil {
			return false, err
		}
		return false, model.ErrInvalidLogin
	}

	if err := s.r.ResetAttempts(ctx, username); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Admin) GetCredentials(ctx context.Context) (string, string, error) {
	user, err := s.r.GetFirstUser(ctx)
	if err != nil {
		return "", "", err
	}
	return user.Username, user.Password, nil
}

var wordList = []string{
	"apple", "banana", "cherry", "dragon", "eagle",
	"forest", "garden", "hollow", "island", "jade",
	"koala", "lemon", "mango", "night", "ocean",
	"panda", "quartz", "raven", "stone", "tiger",
	"umbra", "violet", "whale", "xenon", "yacht", "zebra",
}

func randomWords(count int) (string, error) {
	words := make([]string, count)
	for i := range words {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(wordList))))
		if err != nil {
			return "", err
		}
		words[i] = wordList[n.Int64()]
	}
	result := words[0]
	for i := 1; i < count; i++ {
		result += "-" + words[i]
	}
	return result, nil
}

func randomPassword(length int) (string, error) {
	const upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const lower = "abcdefghijklmnopqrstuvwxyz"
	const digits = "0123456789"
	const special = "!@#$%^&*"
	all := upper + lower + digits + special

	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(all))))
		if err != nil {
			return "", err
		}
		b[i] = all[n.Int64()]
	}
	return string(b), nil
}
