package adminauth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
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
	mu       sync.RWMutex
	sessions map[string]Session
}

func New(cfg config.Config) *Service {
	return &Service{
		cfg:      cfg,
		sessions: map[string]Session{},
	}
}

func (s *Service) Login(username, password string) (Session, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return Session{}, ErrInvalidInput
	}
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
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Service) Current(token string) (Session, error) {
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

func generateToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
