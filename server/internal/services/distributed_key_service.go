package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"tavily-proxy/server/internal/models"

	"gorm.io/gorm"
)

var (
	ErrDistributedKeyNotFound = errors.New("distributed_key_not_found")
	ErrDistributedKeyDisabled = errors.New("distributed_key_disabled")
	ErrDistributedKeyExpired  = errors.New("distributed_key_expired")
	ErrInvalidRateLimit       = errors.New("invalid_rate_limit_per_minute")
)

const (
	distributedKeyPrefixDefault = "uk_live_"
	distributedKeyNameDefault   = "User Key"
)

type DistributedKeyService struct {
	db               *gorm.DB
	logger           *slog.Logger
	cipher           *TokenCipher
	defaultRateLimit int
}

type DistributedKeyCreateInput struct {
	Name               string
	Note               string
	ExpiresAt          *time.Time
	RateLimitPerMinute *int
}

type DistributedKeyUpdateInput struct {
	Name               *string
	Note               *string
	IsActive           *bool
	ExpiresAt          *time.Time
	ClearExpiresAt     bool
	RateLimitPerMinute *int
}

func NewDistributedKeyService(db *gorm.DB, logger *slog.Logger, cipher *TokenCipher, defaultRateLimit int) *DistributedKeyService {
	if defaultRateLimit < 0 {
		defaultRateLimit = 0
	}
	return &DistributedKeyService{
		db:               db,
		logger:           logger,
		cipher:           cipher,
		defaultRateLimit: defaultRateLimit,
	}
}

func (s *DistributedKeyService) List(ctx context.Context) ([]models.DistributedKey, error) {
	var keys []models.DistributedKey
	if err := s.db.WithContext(ctx).Order("id desc").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *DistributedKeyService) FindByID(ctx context.Context, id uint) (*models.DistributedKey, error) {
	var key models.DistributedKey
	if err := s.db.WithContext(ctx).First(&key, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &key, nil
}

func (s *DistributedKeyService) Create(ctx context.Context, in DistributedKeyCreateInput) (*models.DistributedKey, string, error) {
	rateLimit := s.defaultRateLimit
	if in.RateLimitPerMinute != nil {
		rateLimit = *in.RateLimitPerMinute
	}
	if rateLimit < 0 {
		return nil, "", ErrInvalidRateLimit
	}

	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = distributedKeyNameDefault
	}

	for i := 0; i < 3; i++ {
		token, err := generateDistributedKeyToken()
		if err != nil {
			return nil, "", err
		}

		ciphertext, err := s.cipher.Encrypt(token)
		if err != nil {
			return nil, "", err
		}

		record := models.DistributedKey{
			Name:               name,
			Note:               strings.TrimSpace(in.Note),
			TokenHash:          hashToken(token),
			Ciphertext:         ciphertext,
			KeyPrefix:          tokenPrefix(token),
			IsActive:           true,
			ExpiresAt:          in.ExpiresAt,
			RateLimitPerMinute: rateLimit,
		}

		if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
			if isUniqueConstraintError(err) {
				continue
			}
			return nil, "", err
		}

		s.logger.Info("distributed key created", "id", record.ID, "name", record.Name)
		return &record, token, nil
	}

	return nil, "", errors.New("failed to create unique key")
}

func (s *DistributedKeyService) Update(ctx context.Context, id uint, in DistributedKeyUpdateInput) (*models.DistributedKey, error) {
	key, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if key == nil {
		return nil, ErrDistributedKeyNotFound
	}

	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			name = distributedKeyNameDefault
		}
		key.Name = name
	}
	if in.Note != nil {
		key.Note = strings.TrimSpace(*in.Note)
	}
	if in.IsActive != nil {
		key.IsActive = *in.IsActive
	}
	if in.ClearExpiresAt {
		key.ExpiresAt = nil
	} else if in.ExpiresAt != nil {
		key.ExpiresAt = in.ExpiresAt
	}
	if in.RateLimitPerMinute != nil {
		if *in.RateLimitPerMinute < 0 {
			return nil, ErrInvalidRateLimit
		}
		key.RateLimitPerMinute = *in.RateLimitPerMinute
	}

	if err := s.db.WithContext(ctx).Save(key).Error; err != nil {
		return nil, err
	}
	return key, nil
}

func (s *DistributedKeyService) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Delete(&models.DistributedKey{}, id).Error
}

func (s *DistributedKeyService) Rotate(ctx context.Context, id uint) (*models.DistributedKey, string, error) {
	key, err := s.FindByID(ctx, id)
	if err != nil {
		return nil, "", err
	}
	if key == nil {
		return nil, "", ErrDistributedKeyNotFound
	}

	for i := 0; i < 3; i++ {
		token, err := generateDistributedKeyToken()
		if err != nil {
			return nil, "", err
		}

		ciphertext, err := s.cipher.Encrypt(token)
		if err != nil {
			return nil, "", err
		}

		key.TokenHash = hashToken(token)
		key.Ciphertext = ciphertext
		key.KeyPrefix = tokenPrefix(token)
		key.IsActive = true
		key.LastUsedAt = nil

		if err := s.db.WithContext(ctx).Save(key).Error; err != nil {
			if isUniqueConstraintError(err) {
				continue
			}
			return nil, "", err
		}

		s.logger.Info("distributed key rotated", "id", key.ID)
		return key, token, nil
	}

	return nil, "", errors.New("failed to rotate key")
}

func (s *DistributedKeyService) AuthenticateBearer(ctx context.Context, token string, now time.Time) (*models.DistributedKey, error) {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return nil, ErrDistributedKeyNotFound
	}

	var key models.DistributedKey
	if err := s.db.WithContext(ctx).First(&key, "token_hash = ?", hashToken(trimmed)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDistributedKeyNotFound
		}
		return nil, err
	}

	plain, err := s.cipher.Decrypt(key.Ciphertext)
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(plain), []byte(trimmed)) != 1 {
		return nil, ErrDistributedKeyNotFound
	}
	if !key.IsActive {
		return nil, ErrDistributedKeyDisabled
	}
	if key.ExpiresAt != nil && !key.ExpiresAt.After(now) {
		return nil, ErrDistributedKeyExpired
	}
	return &key, nil
}

func (s *DistributedKeyService) TouchLastUsed(ctx context.Context, id uint, when time.Time) error {
	return s.db.WithContext(ctx).
		Model(&models.DistributedKey{}).
		Where("id = ?", id).
		Update("last_used_at", when).
		Error
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func tokenPrefix(token string) string {
	if len(token) <= 14 {
		return token
	}
	return token[:14]
}

func generateDistributedKeyToken() (string, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return distributedKeyPrefixDefault + base64.RawURLEncoding.EncodeToString(raw), nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint") || strings.Contains(msg, "duplicate key")
}

func ParseRFC3339Ptr(raw string) (*time.Time, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, fmt.Errorf("invalid datetime: %w", err)
	}
	return &t, nil
}
