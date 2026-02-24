package models

import "time"

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	Key        string     `gorm:"uniqueIndex;not null" json:"-"`
	Alias      string     `gorm:"not null" json:"alias"`
	TotalQuota int        `gorm:"not null;default:1000" json:"total_quota"`
	UsedQuota  int        `gorm:"not null;default:0" json:"used_quota"`
	IsActive   bool       `gorm:"not null;default:true" json:"is_active"`
	IsInvalid  bool       `gorm:"not null;default:false" json:"is_invalid"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type RequestLog struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	RequestID         string    `gorm:"index;not null" json:"request_id"`
	KeyUsed           uint      `gorm:"column:key_used;index" json:"key_used"`
	KeyAlias          string    `json:"key_alias"`
	Endpoint          string    `gorm:"index;not null" json:"endpoint"`
	StatusCode        int       `json:"status_code"`
	LatencyMs         int64     `json:"latency"`
	RequestBody       string    `gorm:"type:text" json:"request_body,omitempty"`
	RequestTruncated  bool      `gorm:"not null;default:false" json:"request_truncated"`
	ResponseBody      string    `gorm:"type:text" json:"response_body,omitempty"`
	ResponseTruncated bool      `gorm:"not null;default:false" json:"response_truncated"`
	ClientIP          string    `json:"client_ip"`
	CreatedAt         time.Time `gorm:"index" json:"created_at"`
}

type DistributedKey struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Name               string     `gorm:"not null" json:"name"`
	Note               string     `gorm:"type:text" json:"note"`
	TokenHash          string     `gorm:"size:64;uniqueIndex;not null" json:"-"`
	Ciphertext         string     `gorm:"type:text;not null" json:"-"`
	KeyPrefix          string     `gorm:"size:64;not null" json:"key_prefix"`
	IsActive           bool       `gorm:"not null;default:true" json:"is_active"`
	ExpiresAt          *time.Time `json:"expires_at"`
	RateLimitPerMinute int        `gorm:"not null;default:60" json:"rate_limit_per_minute"`
	LastUsedAt         *time.Time `json:"last_used_at"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type DistributedKeyUsageDaily struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	DistributedKeyID uint      `gorm:"not null;index:idx_distributed_key_usage_daily,unique" json:"distributed_key_id"`
	Date             string    `gorm:"size:10;not null;index:idx_distributed_key_usage_daily,unique" json:"date"`
	TotalCount       int64     `gorm:"not null;default:0" json:"total_count"`
	Status2xx        int64     `gorm:"column:status_2xx;not null;default:0" json:"status_2xx"`
	Status4xx        int64     `gorm:"column:status_4xx;not null;default:0" json:"status_4xx"`
	Status5xx        int64     `gorm:"column:status_5xx;not null;default:0" json:"status_5xx"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RequestStat struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Granularity string    `gorm:"not null;index:idx_request_stat_bucket,unique" json:"granularity"`
	Bucket      string    `gorm:"not null;index:idx_request_stat_bucket,unique" json:"bucket"`
	Endpoint    string    `gorm:"not null;default:'';index:idx_request_stat_bucket,unique" json:"endpoint"`
	Count       int64     `gorm:"not null;default:0" json:"count"`
	UpdatedAt   time.Time `gorm:"index" json:"updated_at"`
}

type Setting struct {
	Key       string    `gorm:"primaryKey" json:"key"`
	Value     string    `gorm:"not null" json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}
