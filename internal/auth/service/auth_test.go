package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/auth/model"
)

type mockStorage struct {
	user        model.AuthUser
	userErr     error
	createErr   error
	updateErr   error
	resetErr    error
	upsertErr   error
	firstUser   model.AuthUser
	firstUserOK bool
	existsOK    bool
}

func (m *mockStorage) GetUser(_ context.Context, username string) (model.AuthUser, error) {
	if m.userErr != nil {
		return model.AuthUser{}, m.userErr
	}
	u := m.user
	u.Username = username
	return u, nil
}

func (m *mockStorage) GetFirstUser(_ context.Context) (model.AuthUser, error) {
	if !m.firstUserOK {
		return model.AuthUser{}, model.ErrNotFound
	}
	return m.firstUser, nil
}

func (m *mockStorage) UserExists(_ context.Context) (bool, error) {
	return m.existsOK, nil
}

func (m *mockStorage) UpdateAttempts(_ context.Context, _ string, attempts int, lockedUntil *time.Time) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.user.LoginAttempts = attempts
	if lockedUntil != nil {
		m.user.LockedUntil = lockedUntil.Format(time.RFC3339)
	}
	return nil
}

func (m *mockStorage) ResetAttempts(_ context.Context, _ string) error {
	if m.resetErr != nil {
		return m.resetErr
	}
	m.user.LoginAttempts = 0
	return nil
}

func (m *mockStorage) CreateInitialUser(_ context.Context, _, _ string) error {
	return m.createErr
}

func (m *mockStorage) UpsertUser(_ context.Context, username, password string) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.user = model.AuthUser{Username: username, Password: password}
	return nil
}

func TestCheckLogin_Success(t *testing.T) {
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAuth(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckLogin_WrongPassword(t *testing.T) {
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAuth(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "wrong", time.Now())

	assert.ErrorIs(t, err, model.ErrInvalidLogin)
	assert.False(t, ok)
}

func TestCheckLogin_UserNotFound(t *testing.T) {
	r := &mockStorage{userErr: model.ErrNotFound}
	s := NewAuth(r)

	ok, err := s.CheckLogin(context.Background(), "nobody", "pwd", time.Now())

	assert.ErrorIs(t, err, model.ErrInvalidLogin)
	assert.False(t, ok)
}

func TestCheckLogin_Lockout(t *testing.T) {
	lockedUntil := time.Now().Add(5 * time.Minute)
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LockedUntil: lockedUntil.Format(time.RFC3339), LoginAttempts: 3},
	}
	s := NewAuth(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.ErrorIs(t, err, model.ErrLockedOut)
	assert.False(t, ok)
}

func TestCheckLogin_LockoutExpired(t *testing.T) {
	lockedUntil := time.Now().Add(-1 * time.Minute)
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LockedUntil: lockedUntil.Format(time.RFC3339), LoginAttempts: 3},
	}
	s := NewAuth(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckLogin_FailedAttemptsIncremented(t *testing.T) {
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAuth(r)

	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	ok, err := s.CheckLogin(context.Background(), "admin", "wrong", time.Now())

	assert.ErrorIs(t, err, model.ErrLockedOut)
	assert.False(t, ok)
}

func TestGetCredentials_Success(t *testing.T) {
	r := &mockStorage{
		firstUser:   model.AuthUser{Username: "admin", Password: "pass"},
		firstUserOK: true,
	}
	s := NewAuth(r)

	user, pass, err := s.GetCredentials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pass", pass)
}

func TestGetCredentials_NotFound(t *testing.T) {
	r := &mockStorage{firstUserOK: false}
	s := NewAuth(r)

	_, _, err := s.GetCredentials(context.Background())

	assert.ErrorIs(t, err, model.ErrNotFound)
}

func TestEnsureCredentials_FirstStart(t *testing.T) {
	r := &mockStorage{existsOK: false}
	s := NewAuth(r)

	username, password, err := s.EnsureCredentials(context.Background(), "", "")

	assert.NoError(t, err)
	assert.Contains(t, username, "-")
	assert.Len(t, password, 16)
}

func TestEnsureCredentials_AlreadyExists(t *testing.T) {
	r := &mockStorage{
		existsOK:    true,
		firstUser:   model.AuthUser{Username: "admin", Password: "pass"},
		firstUserOK: true,
	}
	s := NewAuth(r)

	user, pass, err := s.EnsureCredentials(context.Background(), "", "")

	assert.NoError(t, err)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pass", pass)
}

func TestEnsureCredentials_Configured(t *testing.T) {
	r := &mockStorage{}
	s := NewAuth(r)

	user, pass, err := s.EnsureCredentials(context.Background(), "operator", "s3cret")

	assert.NoError(t, err)
	assert.Equal(t, "operator", user)
	assert.Equal(t, "s3cret", pass)
	assert.Equal(t, "operator", r.user.Username)
	assert.Equal(t, "s3cret", r.user.Password)
}

func TestEnsureCredentials_ConfiguredError(t *testing.T) {
	r := &mockStorage{upsertErr: assert.AnError}
	s := NewAuth(r)

	_, _, err := s.EnsureCredentials(context.Background(), "operator", "s3cret")

	assert.ErrorIs(t, err, assert.AnError)
	assert.Empty(t, r.user.Username)
}
