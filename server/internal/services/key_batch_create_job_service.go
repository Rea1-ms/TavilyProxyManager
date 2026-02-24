package services

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const maxBatchCreateKeys = 5000

var (
	ErrBatchCreateNoKeys   = errors.New("no keys to create")
	ErrBatchCreateTooLarge = errors.New("too many keys in one batch")
)

type KeyBatchCreateFailure struct {
	Key   string `json:"key"`
	Error string `json:"error"`
}

type KeyBatchCreateJobStatus struct {
	ID     string `json:"id"`
	Status string `json:"status"` // idle|running|completed|error
	Error  string `json:"error,omitempty"`

	Total     int `json:"total"`
	Completed int `json:"completed"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`

	Failures []KeyBatchCreateFailure `json:"failures,omitempty"`

	StartedAt *time.Time `json:"started_at,omitempty"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
}

type KeyBatchCreateJobService struct {
	keys   *KeyService
	logger *slog.Logger

	mu  sync.RWMutex
	job *KeyBatchCreateJobStatus
}

func NewKeyBatchCreateJobService(keys *KeyService, logger *slog.Logger) *KeyBatchCreateJobService {
	return &KeyBatchCreateJobService{keys: keys, logger: logger}
}

func (s *KeyBatchCreateJobService) Get() KeyBatchCreateJobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.job == nil {
		return KeyBatchCreateJobStatus{Status: "idle"}
	}
	return cloneKeyBatchCreateJobStatus(*s.job)
}

func (s *KeyBatchCreateJobService) Start(rawKeys []string, alias string, totalQuota int) (KeyBatchCreateJobStatus, bool, error) {
	keys := normalizeBatchKeys(rawKeys)
	if len(keys) == 0 {
		return KeyBatchCreateJobStatus{}, false, ErrBatchCreateNoKeys
	}
	if len(keys) > maxBatchCreateKeys {
		return KeyBatchCreateJobStatus{}, false, ErrBatchCreateTooLarge
	}

	if strings.TrimSpace(alias) == "" {
		alias = "Default"
	}

	s.mu.Lock()
	if s.job != nil && s.job.Status == "running" {
		job := cloneKeyBatchCreateJobStatus(*s.job)
		s.mu.Unlock()
		return job, true, nil
	}

	startedAt := time.Now()
	job := &KeyBatchCreateJobStatus{
		ID:        uuid.NewString(),
		Status:    "running",
		Total:     len(keys),
		StartedAt: &startedAt,
	}
	s.job = job
	s.mu.Unlock()

	go s.runJob(job.ID, keys, alias, totalQuota)

	return cloneKeyBatchCreateJobStatus(*job), false, nil
}

func (s *KeyBatchCreateJobService) runJob(jobID string, keys []string, alias string, totalQuota int) {
	ctx := context.Background()

	for _, key := range keys {
		_, err := s.keys.Create(ctx, key, alias, totalQuota)

		s.mu.Lock()
		if s.job == nil || s.job.ID != jobID {
			s.mu.Unlock()
			return
		}

		s.job.Completed++
		if err != nil {
			s.job.Failed++
			s.job.Failures = append(s.job.Failures, KeyBatchCreateFailure{
				Key:   key,
				Error: err.Error(),
			})
		} else {
			s.job.Succeeded++
		}
		s.mu.Unlock()
	}

	endedAt := time.Now()
	s.mu.Lock()
	if s.job != nil && s.job.ID == jobID {
		s.job.Status = "completed"
		s.job.EndedAt = &endedAt
	}
	s.mu.Unlock()
}

func normalizeBatchKeys(rawKeys []string) []string {
	out := make([]string, 0, len(rawKeys))
	seen := make(map[string]struct{}, len(rawKeys))
	for _, raw := range rawKeys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func cloneKeyBatchCreateJobStatus(in KeyBatchCreateJobStatus) KeyBatchCreateJobStatus {
	out := in
	if in.Failures != nil {
		out.Failures = make([]KeyBatchCreateFailure, len(in.Failures))
		copy(out.Failures, in.Failures)
	}
	if in.StartedAt != nil {
		t := *in.StartedAt
		out.StartedAt = &t
	}
	if in.EndedAt != nil {
		t := *in.EndedAt
		out.EndedAt = &t
	}
	return out
}
