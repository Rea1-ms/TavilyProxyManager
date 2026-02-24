package services

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"tavily-proxy/server/internal/models"

	"gorm.io/gorm"
)

const masterKeySettingKey = "master_key"

type MasterKeyService struct {
	db     *gorm.DB
	logger *slog.Logger

	mu  sync.RWMutex
	key string
}

func NewMasterKeyService(db *gorm.DB, logger *slog.Logger) *MasterKeyService {
	return &MasterKeyService{db: db, logger: logger}
}

func (s *MasterKeyService) LoadOrCreate(ctx context.Context) error {
	return s.LoadOrCreateWithDefault(ctx, "")
}

func (s *MasterKeyService) LoadOrCreateWithDefault(ctx context.Context, preferred string) error {
	var setting models.Setting
	err := s.db.WithContext(ctx).First(&setting, "key = ?", masterKeySettingKey).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	preferred = strings.TrimSpace(preferred)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		masterKey := preferred
		if masterKey == "" {
			var genErr error
			masterKey, genErr = generateSecret(32)
			if genErr != nil {
				return genErr
			}
			s.logger.Info("generated master key", "master_key", masterKey)
		} else {
			s.logger.Info("initialized master key from environment")
		}
		setting = models.Setting{Key: masterKeySettingKey, Value: masterKey}
		if err := s.db.WithContext(ctx).Create(&setting).Error; err != nil {
			return err
		}
	} else if preferred != "" && preferred != setting.Value {
		s.logger.Info("MASTER_KEY ignored because master key already exists in database")
	}

	s.mu.Lock()
	s.key = setting.Value
	s.mu.Unlock()
	return nil
}

func (s *MasterKeyService) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.key
}

func (s *MasterKeyService) Authenticate(token string) bool {
	current := s.Get()
	if current == "" || token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(current), []byte(token)) == 1
}

func (s *MasterKeyService) Reset(ctx context.Context) (string, error) {
	newKey, err := generateSecret(32)
	if err != nil {
		return "", err
	}
	if err := s.db.WithContext(ctx).Save(&models.Setting{Key: masterKeySettingKey, Value: newKey}).Error; err != nil {
		return "", err
	}

	s.mu.Lock()
	s.key = newKey
	s.mu.Unlock()
	return newKey, nil
}

func generateSecret(bytes int) (string, error) {
	if bytes <= 0 {
		return "", errors.New("invalid secret length")
	}
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
