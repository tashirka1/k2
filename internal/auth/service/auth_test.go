package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tashirka1/k2/internal/auth/model"
	"github.com/tashirka1/k2/internal/auth/storage"
)

var _ storage.AuthStorage = (*mockStorage)(nil)

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

func TestCheckLogin(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		userErr  error
		wantErr  error
		password string
		user     model.AuthUser
		wantID   int
	}{
		{
			name:     "success returns user id",
			user:     model.AuthUser{ID: 7, Username: "admin", Password: "secret"},
			password: "secret",
			wantID:   7,
		},
		{
			name:     "wrong password",
			user:     model.AuthUser{ID: 7, Username: "admin", Password: "secret"},
			password: "wrong",
			wantErr:  model.ErrInvalidLogin,
		},
		{
			name:     "user not found",
			userErr:  model.ErrNotFound,
			password: "secret",
			wantErr:  model.ErrInvalidLogin,
		},
		{
			name:     "locked out",
			user:     model.AuthUser{ID: 7, Username: "admin", Password: "secret", LockedUntil: now.Add(5 * time.Minute).Format(time.RFC3339), LoginAttempts: 3},
			password: "secret",
			wantErr:  model.ErrLockedOut,
		},
		{
			name:     "lockout expired",
			user:     model.AuthUser{ID: 7, Username: "admin", Password: "secret", LockedUntil: now.Add(-1 * time.Minute).Format(time.RFC3339), LoginAttempts: 3},
			password: "secret",
			wantID:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockStorage{user: tt.user, userErr: tt.userErr}
			s := NewAuth(r)

			id, err := s.CheckLogin(context.Background(), "admin", tt.password, now)

			assert.Equal(t, tt.wantID, id)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestCheckLogin_FailedAttemptsIncremented(t *testing.T) {
	r := &mockStorage{
		user: model.AuthUser{Username: "admin", Password: "secret", LoginAttempts: 0},
	}
	s := NewAuth(r)

	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	_, _ = s.CheckLogin(context.Background(), "admin", "wrong", time.Now())
	id, err := s.CheckLogin(context.Background(), "admin", "wrong", time.Now())

	assert.Equal(t, 0, id)
	assert.ErrorIs(t, err, model.ErrLockedOut)
}

func TestGetCredentials(t *testing.T) {
	tests := []struct {
		want        model.Credentials
		name        string
		wantErr     error
		firstUser   model.AuthUser
		firstUserOK bool
	}{
		{
			name:        "success",
			firstUser:   model.AuthUser{Username: "admin", Password: "pass"},
			firstUserOK: true,
			want:        model.Credentials{Username: "admin", Password: "pass"},
		},
		{
			name:    "not found",
			wantErr: model.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockStorage{firstUser: tt.firstUser, firstUserOK: tt.firstUserOK}
			s := NewAuth(r)

			creds, err := s.GetCredentials(context.Background())

			assert.Equal(t, tt.want, creds)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEnsureCredentials(t *testing.T) {
	tests := []struct {
		want          model.Credentials
		name          string
		cfgUsername   string
		cfgPassword   string
		upsertErr     error
		createErr     error
		wantErr       error
		firstUser     model.AuthUser
		existsOK      bool
		firstUserOK   bool
		wantGenerated bool
	}{
		{
			name:        "configured uses env credentials",
			cfgUsername: "operator",
			cfgPassword: "s3cret",
			want:        model.Credentials{Username: "operator", Password: "s3cret"},
		},
		{
			name:        "configured upsert error",
			cfgUsername: "operator",
			cfgPassword: "s3cret",
			upsertErr:   assert.AnError,
			wantErr:     assert.AnError,
		},
		{
			name:          "first start generates credentials",
			wantGenerated: true,
		},
		{
			name:      "create error on first start",
			createErr: assert.AnError,
			wantErr:   assert.AnError,
		},
		{
			name:        "already exists returns stored credentials",
			existsOK:    true,
			firstUser:   model.AuthUser{Username: "admin", Password: "pass"},
			firstUserOK: true,
			want:        model.Credentials{Username: "admin", Password: "pass"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &mockStorage{
				existsOK:    tt.existsOK,
				firstUser:   tt.firstUser,
				firstUserOK: tt.firstUserOK,
				upsertErr:   tt.upsertErr,
				createErr:   tt.createErr,
			}
			s := NewAuth(r)

			creds, err := s.EnsureCredentials(context.Background(), tt.cfgUsername, tt.cfgPassword)

			if tt.wantGenerated {
				assert.NoError(t, err)
				assert.Contains(t, creds.Username, "-")
				assert.Len(t, creds.Password, 16)
				return
			}
			assert.Equal(t, tt.want, creds)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
