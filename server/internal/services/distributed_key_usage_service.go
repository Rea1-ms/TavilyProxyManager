package services

import (
	"context"
	"time"

	"tavily-proxy/server/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DistributedKeyUsageService struct {
	db *gorm.DB
}

type DistributedKeyUsageTotals struct {
	TotalCount int64 `gorm:"column:total_count" json:"total_count"`
	Status2xx  int64 `gorm:"column:status_2xx" json:"status_2xx"`
	Status4xx  int64 `gorm:"column:status_4xx" json:"status_4xx"`
	Status5xx  int64 `gorm:"column:status_5xx" json:"status_5xx"`
}

type DistributedKeyUsagePoint struct {
	Date       string `gorm:"column:date" json:"date"`
	TotalCount int64  `gorm:"column:total_count" json:"total_count"`
	Status2xx  int64  `gorm:"column:status_2xx" json:"status_2xx"`
	Status4xx  int64  `gorm:"column:status_4xx" json:"status_4xx"`
	Status5xx  int64  `gorm:"column:status_5xx" json:"status_5xx"`
}

type DistributedKeyAggregatedRow struct {
	DistributedKeyID uint  `gorm:"column:distributed_key_id"`
	TotalCount       int64 `gorm:"column:total_count"`
	Status2xx        int64 `gorm:"column:status_2xx"`
	Status4xx        int64 `gorm:"column:status_4xx"`
	Status5xx        int64 `gorm:"column:status_5xx"`
}

func NewDistributedKeyUsageService(db *gorm.DB) *DistributedKeyUsageService {
	return &DistributedKeyUsageService{db: db}
}

func (s *DistributedKeyUsageService) Record(ctx context.Context, distributedKeyID uint, statusCode int, when time.Time) error {
	if distributedKeyID == 0 {
		return nil
	}

	row := models.DistributedKeyUsageDaily{
		DistributedKeyID: distributedKeyID,
		Date:             when.UTC().Format("2006-01-02"),
		TotalCount:       1,
	}
	switch {
	case statusCode >= 200 && statusCode < 300:
		row.Status2xx = 1
	case statusCode >= 400 && statusCode < 500:
		row.Status4xx = 1
	case statusCode >= 500:
		row.Status5xx = 1
	}

	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "distributed_key_id"},
				{Name: "date"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"total_count": gorm.Expr("total_count + EXCLUDED.total_count"),
				"status_2xx":  gorm.Expr("status_2xx + EXCLUDED.status_2xx"),
				"status_4xx":  gorm.Expr("status_4xx + EXCLUDED.status_4xx"),
				"status_5xx":  gorm.Expr("status_5xx + EXCLUDED.status_5xx"),
				"updated_at":  when.UTC(),
			}),
		}).
		Create(&row).
		Error
}

func (s *DistributedKeyUsageService) Totals(ctx context.Context, distributedKeyID uint) (DistributedKeyUsageTotals, error) {
	var out DistributedKeyUsageTotals
	err := s.db.WithContext(ctx).
		Model(&models.DistributedKeyUsageDaily{}).
		Select(
			"COALESCE(SUM(total_count), 0) AS total_count, "+
				"COALESCE(SUM(status_2xx), 0) AS status_2xx, "+
				"COALESCE(SUM(status_4xx), 0) AS status_4xx, "+
				"COALESCE(SUM(status_5xx), 0) AS status_5xx",
		).
		Where("distributed_key_id = ?", distributedKeyID).
		Scan(&out).
		Error
	return out, err
}

func (s *DistributedKeyUsageService) Series(ctx context.Context, distributedKeyID uint, days int) ([]DistributedKeyUsagePoint, error) {
	if days <= 0 {
		days = 30
	}
	start := time.Now().UTC().AddDate(0, 0, -days+1).Format("2006-01-02")

	var rows []DistributedKeyUsagePoint
	if err := s.db.WithContext(ctx).
		Model(&models.DistributedKeyUsageDaily{}).
		Select("date, total_count, status_2xx, status_4xx, status_5xx").
		Where("distributed_key_id = ? AND date >= ?", distributedKeyID, start).
		Order("date asc").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *DistributedKeyUsageService) AggregateByKey(ctx context.Context) (map[uint]DistributedKeyUsageTotals, error) {
	var rows []DistributedKeyAggregatedRow
	if err := s.db.WithContext(ctx).
		Model(&models.DistributedKeyUsageDaily{}).
		Select(
			"distributed_key_id, " +
				"COALESCE(SUM(total_count), 0) AS total_count, " +
				"COALESCE(SUM(status_2xx), 0) AS status_2xx, " +
				"COALESCE(SUM(status_4xx), 0) AS status_4xx, " +
				"COALESCE(SUM(status_5xx), 0) AS status_5xx",
		).
		Group("distributed_key_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	out := make(map[uint]DistributedKeyUsageTotals, len(rows))
	for _, row := range rows {
		out[row.DistributedKeyID] = DistributedKeyUsageTotals{
			TotalCount: row.TotalCount,
			Status2xx:  row.Status2xx,
			Status4xx:  row.Status4xx,
			Status5xx:  row.Status5xx,
		}
	}
	return out, nil
}
