package studentauth

import (
	"context"
	"crypto/rand"
	crand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/raywong-bitscube/stepup/backend/internal/config"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidInput  = errors.New("invalid_input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrCodeInvalid   = errors.New("code_invalid")
	ErrCodeExpired   = errors.New("code_expired")
	ErrCodeUsed      = errors.New("code_used")
	ErrPasswordUnset = errors.New("password_unset")
)

type student struct {
	Identifier string
	Password   string
	Status     int
}

type verification struct {
	Identifier string
	Code       string
	ExpiresAt  time.Time
	Used       bool
}

type Session struct {
	StudentID  uint64
	Token      string
	Identifier string
	ExpiresAt  time.Time
	LastSeenAt time.Time
}

type Service struct {
	cfg           config.Config
	db            *sql.DB
	mu            sync.RWMutex
	students      map[string]student
	codes         map[string]verification
	sessions      map[string]Session
	codeTTL       time.Duration
	sessionTTL    time.Duration
	defaultStatus int
}

func New(cfg config.Config, db *sql.DB) *Service {
	return &Service{
		cfg:           cfg,
		db:            db,
		students:      map[string]student{},
		codes:         map[string]verification{},
		sessions:      map[string]Session{},
		codeTTL:       5 * time.Minute,
		sessionTTL:    cfg.SessionTTL,
		defaultStatus: 1,
	}
}

func (s *Service) SendCode(identifier string) (string, error) {
	identifier = normalizeIdentifier(identifier)
	if identifier == "" {
		return "", ErrInvalidInput
	}
	if s.db != nil {
		return s.sendCodeDB(identifier)
	}
	return s.sendCodeMemory(identifier)
}

func (s *Service) sendCodeMemory(identifier string) (string, error) {
	v := verification{
		Identifier: identifier,
		Code:       generateCode(),
		ExpiresAt:  time.Now().Add(s.codeTTL),
	}

	s.mu.Lock()
	s.codes[identifier] = v
	s.mu.Unlock()
	return v.Code, nil
}

func (s *Service) VerifyCode(identifier, code string) error {
	identifier = normalizeIdentifier(identifier)
	if identifier == "" || strings.TrimSpace(code) == "" {
		return ErrInvalidInput
	}
	if s.db != nil {
		return s.verifyCodeDB(identifier, code)
	}
	return s.verifyCodeMemory(identifier, code)
}

func (s *Service) verifyCodeMemory(identifier, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	v, ok := s.codes[identifier]
	if !ok || v.Code != code {
		return ErrCodeInvalid
	}
	if v.Used {
		return ErrCodeUsed
	}
	if time.Now().After(v.ExpiresAt) {
		return ErrCodeExpired
	}
	v.Used = true
	s.codes[identifier] = v
	return nil
}

func (s *Service) SetPassword(identifier, password string) error {
	identifier = normalizeIdentifier(identifier)
	password = strings.TrimSpace(password)
	if identifier == "" || password == "" {
		return ErrInvalidInput
	}
	if s.db != nil {
		return s.setPasswordDB(identifier, password)
	}
	return s.setPasswordMemory(identifier, password)
}

func (s *Service) setPasswordMemory(identifier, password string) error {
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.students[identifier]
	if !ok {
		s.students[identifier] = student{
			Identifier: identifier,
			Password:   hashed,
			Status:     s.defaultStatus,
		}
		return nil
	}
	existing.Password = hashed
	s.students[identifier] = existing
	return nil
}

func (s *Service) Login(identifier, password string) (Session, error) {
	identifier = normalizeIdentifier(identifier)
	password = strings.TrimSpace(password)
	if identifier == "" || password == "" {
		return Session{}, ErrInvalidInput
	}
	if s.db != nil {
		return s.loginDB(identifier, password)
	}
	return s.loginMemory(identifier, password)
}

func (s *Service) loginMemory(identifier, password string) (Session, error) {
	s.mu.RLock()
	stu, ok := s.students[identifier]
	s.mu.RUnlock()
	if !ok {
		return Session{}, ErrUnauthorized
	}
	if stu.Status != 1 {
		return Session{}, ErrUnauthorized
	}
	if stu.Password == "" {
		return Session{}, ErrPasswordUnset
	}
	if !verifyPassword(stu.Password, password) {
		return Session{}, ErrUnauthorized
	}

	token, err := generateToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	session := Session{
		StudentID:  0,
		Token:      token,
		Identifier: identifier,
		ExpiresAt:  now.Add(s.sessionTTL),
		LastSeenAt: now,
	}

	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

func (s *Service) Logout(token string) {
	token = strings.TrimSpace(token)
	if s.db != nil {
		_ = s.logoutStudentDB(token)
		return
	}
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

func (s *Service) Current(token string) (Session, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Session{}, ErrUnauthorized
	}
	if s.db != nil {
		return s.currentStudentDB(token)
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

	session.LastSeenAt = time.Now()
	s.mu.Lock()
	s.sessions[token] = session
	s.mu.Unlock()
	return session, nil
}

func normalizeIdentifier(identifier string) string {
	return strings.TrimSpace(strings.ToLower(identifier))
}

func generateToken() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func generateCode() string {
	n, err := crand.Int(crand.Reader, big.NewInt(1000000))
	if err != nil {
		return "123456"
	}
	return fmt.Sprintf("%06d", n.Int64())
}

func (s *Service) sendCodeDB(identifier string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	code := generateCode()
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO verification_code
  (identifier, code, type, expires_at, is_used, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, 'login', ?, 0, ?, 0, ?, 0, 0)
`, identifier, code, now.Add(s.codeTTL), now, now)
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *Service) verifyCodeDB(identifier, code string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var (
		id        uint64
		dbCode    string
		expiresAt time.Time
		isUsed    bool
	)
	err := s.db.QueryRowContext(ctx, `
SELECT id, code, expires_at, is_used
FROM verification_code
WHERE identifier = ? AND type = 'login' AND is_deleted = 0
ORDER BY id DESC
LIMIT 1
`, identifier).Scan(&id, &dbCode, &expiresAt, &isUsed)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCodeInvalid
		}
		return err
	}
	if isUsed {
		return ErrCodeUsed
	}
	if time.Now().After(expiresAt) {
		return ErrCodeExpired
	}
	if dbCode != code {
		return ErrCodeInvalid
	}

	_, err = s.db.ExecContext(ctx, `
UPDATE verification_code
SET is_used = 1, updated_at = ?, updated_by = 0
WHERE id = ?
`, time.Now(), id)
	return err
}

func (s *Service) setPasswordDB(identifier, password string) error {
	hashed, err := hashPassword(password)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id uint64
	err = s.db.QueryRowContext(ctx, `
SELECT id FROM student
WHERE (phone = ? OR email = ?) AND is_deleted = 0
LIMIT 1
`, identifier, identifier).Scan(&id)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	now := time.Now()
	if errors.Is(err, sql.ErrNoRows) {
		phone := sql.NullString{}
		email := sql.NullString{}
		if strings.Contains(identifier, "@") {
			email = sql.NullString{String: identifier, Valid: true}
		} else {
			phone = sql.NullString{String: identifier, Valid: true}
		}
		_, err = s.db.ExecContext(ctx, `
INSERT INTO student
  (phone, email, password, name, stage_id, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, 1, 1, ?, 0, ?, 0, 0)
`, phone, email, hashed, identifier, now, now)
		return err
	}

	_, err = s.db.ExecContext(ctx, `
UPDATE student
SET password = ?, updated_at = ?, updated_by = 0
WHERE id = ?
`, hashed, now, id)
	return err
}

func (s *Service) loginDB(identifier, password string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var (
		studentID uint64
		phone     sql.NullString
		email     sql.NullString
		dbPass    string
		status    int
	)
	err := s.db.QueryRowContext(ctx, `
SELECT id, phone, email, password, status
FROM student
WHERE (phone = ? OR email = ?) AND is_deleted = 0
LIMIT 1
`, identifier, identifier).Scan(&studentID, &phone, &email, &dbPass, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	if status != 1 || !verifyPassword(dbPass, password) {
		return Session{}, ErrUnauthorized
	}

	token, err := generateToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now()
	expiresAt := now.Add(s.sessionTTL)

	_, err = s.db.ExecContext(ctx, `
INSERT INTO student_session
  (student_id, session_token, expires_at, last_seen_at, ip_address, user_agent, status, created_at, created_by, updated_at, updated_by, is_deleted)
VALUES (?, ?, ?, ?, '', '', 1, ?, ?, ?, ?, 0)
`, studentID, token, expiresAt, now, now, studentID, now, studentID)
	if err != nil {
		return Session{}, fmt.Errorf("create student_session: %w", err)
	}

	return Session{
		StudentID:  studentID,
		Token:      token,
		Identifier: identifier,
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}, nil
}

func (s *Service) currentStudentDB(token string) (Session, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var (
		studentID uint64
		phone     sql.NullString
		email     sql.NullString
		expiresAt time.Time
	)
	err := s.db.QueryRowContext(ctx, `
SELECT s.id, s.phone, s.email, sess.expires_at
FROM student_session sess
JOIN student s ON s.id = sess.student_id
WHERE sess.session_token = ?
  AND sess.status = 1
  AND sess.is_deleted = 0
  AND s.status = 1
  AND s.is_deleted = 0
LIMIT 1
`, token).Scan(&studentID, &phone, &email, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Session{}, ErrUnauthorized
		}
		return Session{}, err
	}
	if time.Now().After(expiresAt) {
		_ = s.logoutStudentDB(token)
		return Session{}, ErrUnauthorized
	}

	now := time.Now()
	_, _ = s.db.ExecContext(ctx, `
UPDATE student_session
SET last_seen_at = ?, updated_at = ?, updated_by = ?
WHERE session_token = ? AND is_deleted = 0
`, now, now, studentID, token)

	var idf string
	if phone.Valid && strings.TrimSpace(phone.String) != "" {
		idf = normalizeIdentifier(phone.String)
	} else if email.Valid && strings.TrimSpace(email.String) != "" {
		idf = normalizeIdentifier(email.String)
	}

	return Session{
		StudentID:  studentID,
		Token:      token,
		Identifier: idf,
		ExpiresAt:  expiresAt,
		LastSeenAt: now,
	}, nil
}

func (s *Service) logoutStudentDB(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := s.db.ExecContext(ctx, `
UPDATE student_session
SET status = 0, updated_at = NOW(), updated_by = student_id
WHERE session_token = ? AND is_deleted = 0
`, token)
	return err
}

func hashPassword(raw string) (string, error) {
	out, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func verifyPassword(stored, input string) bool {
	// Backward compatibility: allow plain-text value during transition.
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(input)) == nil
	}
	return stored == input
}
