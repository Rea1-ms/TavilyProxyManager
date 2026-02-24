package httpserver

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"tavily-proxy/server/internal/services"
)

func handleGetBatchCreateKeys(c *gin.Context, jobs *services.KeyBatchCreateJobService) {
	if jobs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "batch_create_unavailable"})
		return
	}
	c.JSON(http.StatusOK, jobs.Get())
}

func handleStartBatchCreateKeys(c *gin.Context, jobs *services.KeyBatchCreateJobService) {
	if jobs == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "batch_create_unavailable"})
		return
	}

	var body struct {
		Keys       []string `json:"keys"`
		Alias      string   `json:"alias"`
		TotalQuota int      `json:"total_quota"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	job, alreadyRunning, err := jobs.Start(body.Keys, body.Alias, body.TotalQuota)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrBatchCreateNoKeys):
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing_keys"})
		case errors.Is(err, services.ErrBatchCreateTooLarge):
			c.JSON(http.StatusBadRequest, gin.H{"error": "too_many_keys"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "batch_create_failed"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"job":             job,
		"already_running": alreadyRunning,
	})
}
