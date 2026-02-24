package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tavily-proxy/server/internal/services"
)

func handleListDistributedKeys(c *gin.Context, distributedKeys *services.DistributedKeyService, usage *services.DistributedKeyUsageService) {
	if distributedKeys == nil || usage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	keys, err := distributedKeys.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	aggregate, err := usage.AggregateByKey(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	type item struct {
		ID                 uint    `json:"id"`
		Name               string  `json:"name"`
		Note               string  `json:"note"`
		KeyPrefix          string  `json:"key_prefix"`
		IsActive           bool    `json:"is_active"`
		ExpiresAt          *string `json:"expires_at"`
		RateLimitPerMinute int     `json:"rate_limit_per_minute"`
		LastUsedAt         *string `json:"last_used_at"`
		CreatedAt          string  `json:"created_at"`
		TotalCount         int64   `json:"total_count"`
		Status2xx          int64   `json:"status_2xx"`
		Status4xx          int64   `json:"status_4xx"`
		Status5xx          int64   `json:"status_5xx"`
	}

	items := make([]item, 0, len(keys))
	for _, key := range keys {
		stats := aggregate[key.ID]
		items = append(items, item{
			ID:                 key.ID,
			Name:               key.Name,
			Note:               key.Note,
			KeyPrefix:          key.KeyPrefix,
			IsActive:           key.IsActive,
			ExpiresAt:          formatTimePtr(key.ExpiresAt),
			RateLimitPerMinute: key.RateLimitPerMinute,
			LastUsedAt:         formatTimePtr(key.LastUsedAt),
			CreatedAt:          key.CreatedAt.Format(time.RFC3339),
			TotalCount:         stats.TotalCount,
			Status2xx:          stats.Status2xx,
			Status4xx:          stats.Status4xx,
			Status5xx:          stats.Status5xx,
		})
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func handleCreateDistributedKey(c *gin.Context, distributedKeys *services.DistributedKeyService) {
	if distributedKeys == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	var body struct {
		Name               string `json:"name"`
		Note               string `json:"note"`
		ExpiresAt          string `json:"expires_at"`
		RateLimitPerMinute *int   `json:"rate_limit_per_minute"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	expiresAt, err := services.ParseRFC3339Ptr(body.ExpiresAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_expires_at"})
		return
	}

	created, plain, err := distributedKeys.Create(c.Request.Context(), services.DistributedKeyCreateInput{
		Name:               body.Name,
		Note:               body.Note,
		ExpiresAt:          expiresAt,
		RateLimitPerMinute: body.RateLimitPerMinute,
	})
	if err != nil {
		if errors.Is(err, services.ErrInvalidRateLimit) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_rate_limit_per_minute"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plain_key": plain,
		"item": gin.H{
			"id":                    created.ID,
			"name":                  created.Name,
			"note":                  created.Note,
			"key_prefix":            created.KeyPrefix,
			"is_active":             created.IsActive,
			"expires_at":            formatTimePtr(created.ExpiresAt),
			"rate_limit_per_minute": created.RateLimitPerMinute,
			"last_used_at":          formatTimePtr(created.LastUsedAt),
			"created_at":            created.CreatedAt.Format(time.RFC3339),
		},
	})
}

func handleUpdateDistributedKey(c *gin.Context, distributedKeys *services.DistributedKeyService, idStr string) {
	if distributedKeys == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	id, err := parseUintParam(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return
	}

	var body struct {
		Name               *string `json:"name"`
		Note               *string `json:"note"`
		IsActive           *bool   `json:"is_active"`
		ExpiresAt          *string `json:"expires_at"`
		ClearExpiresAt     bool    `json:"clear_expires_at"`
		RateLimitPerMinute *int    `json:"rate_limit_per_minute"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	if body.Name == nil &&
		body.Note == nil &&
		body.IsActive == nil &&
		body.ExpiresAt == nil &&
		!body.ClearExpiresAt &&
		body.RateLimitPerMinute == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_fields"})
		return
	}

	var expiresAt *time.Time
	if body.ExpiresAt != nil {
		parsed, err := services.ParseRFC3339Ptr(strings.TrimSpace(*body.ExpiresAt))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_expires_at"})
			return
		}
		expiresAt = parsed
	}

	updated, err := distributedKeys.Update(c.Request.Context(), uint(id), services.DistributedKeyUpdateInput{
		Name:               body.Name,
		Note:               body.Note,
		IsActive:           body.IsActive,
		ExpiresAt:          expiresAt,
		ClearExpiresAt:     body.ClearExpiresAt,
		RateLimitPerMinute: body.RateLimitPerMinute,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrDistributedKeyNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		case errors.Is(err, services.ErrInvalidRateLimit):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_rate_limit_per_minute"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "update_failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": gin.H{
			"id":                    updated.ID,
			"name":                  updated.Name,
			"note":                  updated.Note,
			"key_prefix":            updated.KeyPrefix,
			"is_active":             updated.IsActive,
			"expires_at":            formatTimePtr(updated.ExpiresAt),
			"rate_limit_per_minute": updated.RateLimitPerMinute,
			"last_used_at":          formatTimePtr(updated.LastUsedAt),
			"created_at":            updated.CreatedAt.Format(time.RFC3339),
		},
	})
}

func handleRotateDistributedKey(c *gin.Context, distributedKeys *services.DistributedKeyService, idStr string) {
	if distributedKeys == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	id, err := parseUintParam(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return
	}

	updated, plain, err := distributedKeys.Rotate(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, services.ErrDistributedKeyNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "rotate_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"plain_key": plain,
		"item": gin.H{
			"id":                    updated.ID,
			"name":                  updated.Name,
			"note":                  updated.Note,
			"key_prefix":            updated.KeyPrefix,
			"is_active":             updated.IsActive,
			"expires_at":            formatTimePtr(updated.ExpiresAt),
			"rate_limit_per_minute": updated.RateLimitPerMinute,
			"last_used_at":          formatTimePtr(updated.LastUsedAt),
			"created_at":            updated.CreatedAt.Format(time.RFC3339),
		},
	})
}

func handleDeleteDistributedKey(c *gin.Context, distributedKeys *services.DistributedKeyService, idStr string) {
	if distributedKeys == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	id, err := parseUintParam(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return
	}

	if err := distributedKeys.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func handleDistributedKeyStats(c *gin.Context, distributedKeys *services.DistributedKeyService, usage *services.DistributedKeyUsageService, idStr string) {
	if distributedKeys == nil || usage == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service_not_configured"})
		return
	}

	id, err := parseUintParam(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id"})
		return
	}

	key, err := distributedKeys.FindByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}
	if key == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	days := 30
	if rawDays := strings.TrimSpace(c.Query("days")); rawDays != "" {
		parsed, err := strconv.Atoi(rawDays)
		if err != nil || parsed < 1 || parsed > 365 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_days"})
			return
		}
		days = parsed
	}

	totals, err := usage.Totals(c.Request.Context(), key.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}
	series, err := usage.Series(c.Request.Context(), key.ID, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"item": gin.H{
			"id":                    key.ID,
			"name":                  key.Name,
			"note":                  key.Note,
			"key_prefix":            key.KeyPrefix,
			"is_active":             key.IsActive,
			"expires_at":            formatTimePtr(key.ExpiresAt),
			"rate_limit_per_minute": key.RateLimitPerMinute,
			"last_used_at":          formatTimePtr(key.LastUsedAt),
			"created_at":            key.CreatedAt.Format(time.RFC3339),
		},
		"totals": totals,
		"series": series,
		"days":   days,
	})
}

func formatTimePtr(v *time.Time) *string {
	if v == nil {
		return nil
	}
	out := v.UTC().Format(time.RFC3339)
	return &out
}
