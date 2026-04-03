package adminauth

import (
	"context"
	"database/sql"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"github.com/raywong-bitscube/stepup/backend/internal/database"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalidInput = errors.New("invalid_input")
)

type Session struct {
	Token     string
	Username  string
	Role      string
	ExpiresAt time.Time
	LastSeen  time.Time
}

type Service struct {
	cfg      config.Config
	db       *sql.DB
	mu       sync.RWMutex
	sessions map[string]Session
}

func New(cfg config.Config) *Service {
	svc := &Service{
		cfg:      cfg,
		sessions: map[string]Session{},
	}
	if cfg.DBDSN != "" {
		db, err := database.OpenMySQL(cfg.DBDSN)
		if err == nil {
			svc.db = db
		}
	}
	return svc
}

func (s *Service) Login(username, password string) (Session, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return Session{}, ErrInvalidInput
	}

	if s.db != nil {
		return s.loginDB(username, password)
	}
	return s.loginMemory(username, password)
}

func (s *Service) loginMemory(username, password string) (Session, error) {
	if username != s.cfg.AdminBootstrapUsername || password != s.cfg.AdminBootstrapPassword {
		return Session{}, ErrUnauthorized
	}
	token, err := generateToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	session := Session{
		Token:     token,
		Username:  username,
		Role:      "super_admin",
		ExpiresAt: now.Add(s.cfg.AdminSessionTTL),
		LastSeen:  now,
	}

	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

func (s *Service) Logout(token string) {
	if s.db != nil {
		_ = s.logoutDB(token)
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Service) Current(token string) (Session, error) {
	if s.db != nil {
		return s.currentDB(token)
	}
	s.mu.RLock()
	session, ok := s.sessions[token]
	s.mu.RUnlock()
	if !ok {
		return Session{}, ErrUnauthorized
	}
	if time.Now().After(session.ExpiresAt) {
		s.Logout(token)
		return Session{}, ErrUnauthorized
	}

	session.LastSeen = time.Now()
	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

func (s *Service) loginDB(username, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var (
		adminID uint64
		dbUser  string
		dbPass  string
		role    string
	)
	err := s.db.QueryRowContext(ctx, `
SELECT id, username, password, role
FROM admin
WHERE username = ? AND status = 1 AND is_deleted = 0
LIMIT 1
`, username).Scan(&adminID, &dbUser, &dbPass, &role)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	if dbPass != password {
		return Session{}, ErrUnauthorized
	}

	token, err := generateToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	expiresAt := now.Add(s.cfg.AdminSessionTTL)

	_, err = s.db.ExecContext(ctx, `
INSERT INTO admin_session
  (admin_id, session_token, expires_at, last_seen_at, ip_address, user_agent, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, '', '', 1, ?, ?, ?, ?, 0)
`, adminID, token, expiresAt, now, now, adminID, now, adminID)
	if err != nil {
		return Session{}, fmt.Errorf("create admin_session failed: %w", err)
	}

	return Session{
		Token:     token,
		Username:  dbUser,
		Role:      role,
		ExpiresAt: expiresAt,
		LastSeen:  now,
	}, nil
}

func (s *Service) currentDB(token string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var (
		adminID   uint64
		username  string
		role      string
		expiresAt time.Time
		lastSeen  sql.NullTime
	)
	err := s.db.QueryRowContext(ctx, `
SELECT a.id, a.username, a.role, sess.expires_at, sess.last_seen_at
FROM admin_session sess
JOIN admin a ON a.id = sess.admin_id
WHERE sess.session_token = ?
  AND sess.status = 1
  AND sess.is_deleted = 0
  AND a.status = 1
  AND a.is_deleted = 0
LIMIT 1
`, token).Scan(&adminID, &username, &role, &expiresAt, &lastSeen)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	if time.Now().After(expiresAt) {
		_ = s.logoutDB(token)
		return Session{}, ErrUnauthorized
	}

	now := time.Now()
	_, _ = s.db.ExecContext(ctx, `
UPDATE admin_session
SET last_seen_at = ?, updated_at = ?, updated_by = ?
WHERE session_token = ? AND is_deleted = 0
`, now, now, adminID, token)

	return Session{
		Token:     token,
		Username:  username,
		Role:      role,
		ExpiresAt: expiresAt,
		LastSeen:  now,
	}, nil
}

func (s *Service) logoutDB(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
UPDATE admin_session
SET status = 0, updated_at = ?, updated_by = COALESCE(updated_by, 0)
WHERE session_token = ? AND is_deleted = 0
`, time.Now(), token)
	return err
}

func generateToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
