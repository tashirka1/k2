package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/admin/model"
)

type mockStorage struct {
	user        model.AdminUser
	userErr     error
	createErr   error
	updateErr   error
	resetErr    error
	firstUser   model.AdminUser
	firstUserOK bool
	existsOK    bool
}

func (m *mockStorage) GetUser(_ context.Context, username string) (model.AdminUser, error) {
	if m.userErr != nil {
		return model.AdminUser{}, m.userErr
	}
	u := m.user
	u.Username = username
	return u, nil
}

func (m *mockStorage) GetFirstUser(_ context.Context) (model.AdminUser, error) {
	if !m.firstUserOK {
		return model.AdminUser{}, model.ErrNotFound
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

func TestCheckLogin_Success(t *testing.T) {
	r := &mockStorage{
		user: model.AdminUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAdmin(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckLogin_WrongPassword(t *testing.T) {
	r := &mockStorage{
		user: model.AdminUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAdmin(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "wrong", time.Now())

	assert.ErrorIs(t, err, model.ErrInvalidLogin)
	assert.False(t, ok)
}

func TestCheckLogin_UserNotFound(t *testing.T) {
	r := &mockStorage{userErr: model.ErrNotFound}
	s := NewAdmin(r)

	ok, err := s.CheckLogin(context.Background(), "nobody", "pwd", time.Now())

	assert.ErrorIs(t, err, model.ErrInvalidLogin)
	assert.False(t, ok)
}

func TestCheckLogin_Lockout(t *testing.T) {
	lockedUntil := time.Now().Add(5 * time.Minute)
	r := &mockStorage{
		user: model.AdminUser{Username: "admin", Password: "secret", LockedUntil: lockedUntil.Format(time.RFC3339), LoginAttempts: 3},
	}
	s := NewAdmin(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.ErrorIs(t, err, model.ErrLockedOut)
	assert.False(t, ok)
}

func TestCheckLogin_LockoutExpired(t *testing.T) {
	lockedUntil := time.Now().Add(-1 * time.Minute)
	r := &mockStorage{
		user: model.AdminUser{Username: "admin", Password: "secret", LockedUntil: lockedUntil.Format(time.RFC3339), LoginAttempts: 3},
	}
	s := NewAdmin(r)

	ok, err := s.CheckLogin(context.Background(), "admin", "secret", time.Now())

	assert.NoError(t, err)
	assert.True(t, ok)
}

func TestCheckLogin_FailedAttemptsIncremented(t *testing.T) {
	r := &mockStorage{
		user: model.AdminUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAdmin(r)

	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	ok, err := s.CheckLogin(context.Background(), "admin", "wrong", time.Now())

	assert.ErrorIs(t, err, model.ErrLockedOut)
	assert.False(t, ok)
}

func TestGetCredentials_Success(t *testing.T) {
	r := &mockStorage{
		firstUser:   model.AdminUser{Username: "admin", Password: "pass"},
		firstUserOK: true,
	}
	s := NewAdmin(r)

	user, pass, err := s.GetCredentials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pass", pass)
}

func TestGetCredentials_NotFound(t *testing.T) {
	r := &mockStorage{firstUserOK: false}
	s := NewAdmin(r)

	_, _, err := s.GetCredentials(context.Background())

	assert.ErrorIs(t, err, model.ErrNotFound)
}

func TestEnsureCredentials_FirstStart(t *testing.T) {
	r := &mockStorage{existsOK: false}
	s := NewAdmin(r)

	username, password, err := s.EnsureCredentials(context.Background())

	assert.NoError(t, err)
	assert.Contains(t, username, "-")
	assert.Len(t, password, 16)
}

func TestEnsureCredentials_AlreadyExists(t *testing.T) {
	r := &mockStorage{
		existsOK:    true,
		firstUser:   model.AdminUser{Username: "admin", Password: "pass"},
		firstUserOK: true,
	}
	s := NewAdmin(r)

	user, pass, err := s.EnsureCredentials(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, "admin", user)
	assert.Equal(t, "pass", pass)
}
