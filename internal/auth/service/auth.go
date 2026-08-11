package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/tashirka1/k2/internal/auth/model"
	"github.com/tashirka1/k2/internal/auth/storage"
)

type AuthService interface {
	EnsureCredentials(ctx context.Context, cfgUsername string, cfgPassword string) (model.Credentials, error)
	CheckLogin(ctx context.Context, username string, password string, now time.Time) (int, error)
	GetCredentials(ctx context.Context) (model.Credentials, error)
}

type Auth struct {
	r storage.AuthStorage
}

func NewAuth(r storage.AuthStorage) *Auth {
	return &Auth{r: r}
}

func (s *Auth) EnsureCredentials(ctx context.Context, cfgUsername string, cfgPassword string) (model.Credentials, error) {
	if cfgUsername != "" && cfgPassword != "" {
		if err := s.r.UpsertUser(ctx, cfgUsername, cfgPassword); err != nil {
			return model.Credentials{}, fmt.Errorf("create initial user: %w", err)
		}
		return model.Credentials{Username: cfgUsername, Password: cfgPassword}, nil
	}
	exists, err := s.r.UserExists(ctx)
	if err != nil {
		return model.Credentials{}, err
	}
	if exists {
		return s.GetCredentials(ctx)
	}
	return s.createCredentials(ctx)
}

func (s *Auth) createCredentials(ctx context.Context) (model.Credentials, error) {
	username, err := randomWords(2)
	if err != nil {
		return model.Credentials{}, fmt.Errorf("generate username: %w", err)
	}
	password, err := randomPassword(16)
	if err != nil {
		return model.Credentials{}, fmt.Errorf("generate password: %w", err)
	}
	if err := s.r.CreateInitialUser(ctx, username, password); err != nil {
		return model.Credentials{}, fmt.Errorf("create initial user: %w", err)
	}
	return model.Credentials{Username: username, Password: password}, nil
}

func (s *Auth) CheckLogin(ctx context.Context, username string, password string, now time.Time) (int, error) {
	user, err := s.r.GetUser(ctx, username)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return 0, model.ErrInvalidLogin
		}
		return 0, err
	}

	if user.LockedUntil != "" {
		lockedUntil, err := time.Parse(time.RFC3339, user.LockedUntil)
		if err == nil && now.Before(lockedUntil) {
			return 0, model.ErrLockedOut
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
			return 0, err
		}
		return 0, model.ErrInvalidLogin
	}

	if err := s.r.ResetAttempts(ctx, username); err != nil {
		return 0, err
	}
	return user.ID, nil
}

func (s *Auth) GetCredentials(ctx context.Context) (model.Credentials, error) {
	user, err := s.r.GetFirstUser(ctx)
	if err != nil {
		return model.Credentials{}, err
	}
	return model.Credentials{Username: user.Username, Password: user.Password}, nil
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
