package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"tavily-proxy/server/internal/db"
	"tavily-proxy/server/internal/services"
)

func TestProxy_DistributedKey_BearerOnlyAndUsageRecorded(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	master := services.NewMasterKeyService(database, logger)
	if err := master.LoadOrCreate(ctx); err != nil {
		t.Fatalf("master init: %v", err)
	}

	pool := services.NewKeyService(database, logger)
	if _, err := pool.Create(ctx, "tvly-pool", "pool", 1000); err != nil {
		t.Fatalf("create pool key: %v", err)
	}
	proxy := services.NewTavilyProxy(upstream.URL, 5*time.Second, pool, nil, nil, logger)

	cipher, err := services.NewTokenCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("token cipher: %v", err)
	}
	distributedKeys := services.NewDistributedKeyService(database, logger, cipher, 60)
	usage := services.NewDistributedKeyUsageService(database)
	limiter := services.NewDistributedRateLimiter(time.Minute)

	_, plain, err := distributedKeys.Create(ctx, services.DistributedKeyCreateInput{
		Name:               "client-a",
		RateLimitPerMinute: ptrInt(10),
	})
	if err != nil {
		t.Fatalf("create distributed key: %v", err)
	}

	router := NewRouter(Dependencies{
		MasterKeyService:           master,
		DistributedKeyService:      distributedKeys,
		DistributedKeyUsageService: usage,
		DistributedRateLimiter:     limiter,
		TavilyProxy:                proxy,
	})

	// Distributed key can call proxy endpoint.
	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	req.Header.Set("Accept", "*/*")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected proxy status: got %d want %d", w.Code, http.StatusOK)
	}

	// Distributed key cannot call admin endpoint.
	adminReq := httptest.NewRequest(http.MethodGet, "/api/stats", nil)
	adminReq.Header.Set("Authorization", "Bearer "+plain)
	adminRes := httptest.NewRecorder()
	router.ServeHTTP(adminRes, adminReq)
	if adminRes.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected admin status: got %d want %d", adminRes.Code, http.StatusUnauthorized)
	}

	keys, err := distributedKeys.List(ctx)
	if err != nil || len(keys) != 1 {
		t.Fatalf("list distributed keys: keys=%d err=%v", len(keys), err)
	}

	totals, err := usage.Totals(ctx, keys[0].ID)
	if err != nil {
		t.Fatalf("usage totals: %v", err)
	}
	if totals.TotalCount != 1 || totals.Status2xx != 1 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestProxy_DistributedKey_RateLimitPerKey(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	master := services.NewMasterKeyService(database, logger)
	if err := master.LoadOrCreate(ctx); err != nil {
		t.Fatalf("master init: %v", err)
	}

	pool := services.NewKeyService(database, logger)
	if _, err := pool.Create(ctx, "tvly-pool", "pool", 1000); err != nil {
		t.Fatalf("create pool key: %v", err)
	}
	proxy := services.NewTavilyProxy(upstream.URL, 5*time.Second, pool, nil, nil, logger)

	cipher, err := services.NewTokenCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("token cipher: %v", err)
	}
	distributedKeys := services.NewDistributedKeyService(database, logger, cipher, 60)
	usage := services.NewDistributedKeyUsageService(database)
	limiter := services.NewDistributedRateLimiter(time.Minute)

	key, plain, err := distributedKeys.Create(ctx, services.DistributedKeyCreateInput{
		Name:               "limited",
		RateLimitPerMinute: ptrInt(1),
	})
	if err != nil {
		t.Fatalf("create distributed key: %v", err)
	}

	router := NewRouter(Dependencies{
		MasterKeyService:           master,
		DistributedKeyService:      distributedKeys,
		DistributedKeyUsageService: usage,
		DistributedRateLimiter:     limiter,
		TavilyProxy:                proxy,
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/usage", nil)
		req.Header.Set("Authorization", "Bearer "+plain)
		req.Header.Set("Accept", "*/*")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		expected := http.StatusOK
		if i == 1 {
			expected = http.StatusTooManyRequests
		}
		if w.Code != expected {
			t.Fatalf("request %d status: got %d want %d", i+1, w.Code, expected)
		}
	}

	totals, err := usage.Totals(ctx, key.ID)
	if err != nil {
		t.Fatalf("usage totals: %v", err)
	}
	if totals.TotalCount != 2 || totals.Status4xx != 1 || totals.Status2xx != 1 {
		t.Fatalf("unexpected totals: %+v", totals)
	}
}

func TestProxy_DistributedKey_ExpiredAndDisabled(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(upstream.Close)

	database, err := db.Open(filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatalf("db handle: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	ctx := context.Background()
	master := services.NewMasterKeyService(database, logger)
	if err := master.LoadOrCreate(ctx); err != nil {
		t.Fatalf("master init: %v", err)
	}
	pool := services.NewKeyService(database, logger)
	if _, err := pool.Create(ctx, "tvly-pool", "pool", 1000); err != nil {
		t.Fatalf("create pool key: %v", err)
	}
	proxy := services.NewTavilyProxy(upstream.URL, 5*time.Second, pool, nil, nil, logger)

	cipher, err := services.NewTokenCipher("0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatalf("token cipher: %v", err)
	}
	distributedKeys := services.NewDistributedKeyService(database, logger, cipher, 60)
	usage := services.NewDistributedKeyUsageService(database)
	limiter := services.NewDistributedRateLimiter(time.Minute)

	expiredAt := time.Now().UTC().Add(-time.Minute)
	_, expiredPlain, err := distributedKeys.Create(ctx, services.DistributedKeyCreateInput{
		Name:      "expired",
		ExpiresAt: &expiredAt,
	})
	if err != nil {
		t.Fatalf("create expired key: %v", err)
	}

	disabledKey, disabledPlain, err := distributedKeys.Create(ctx, services.DistributedKeyCreateInput{
		Name: "disabled",
	})
	if err != nil {
		t.Fatalf("create disabled key: %v", err)
	}
	isActive := false
	if _, err := distributedKeys.Update(ctx, disabledKey.ID, services.DistributedKeyUpdateInput{IsActive: &isActive}); err != nil {
		t.Fatalf("disable key: %v", err)
	}

	router := NewRouter(Dependencies{
		MasterKeyService:           master,
		DistributedKeyService:      distributedKeys,
		DistributedKeyUsageService: usage,
		DistributedRateLimiter:     limiter,
		TavilyProxy:                proxy,
	})

	assertUnauthorizedWithError(t, router, expiredPlain, "key_expired")
	assertUnauthorizedWithError(t, router, disabledPlain, "key_disabled")
}

func assertUnauthorizedWithError(t *testing.T, router http.Handler, plain, wantError string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/usage", nil)
	req.Header.Set("Authorization", "Bearer "+plain)
	req.Header.Set("Accept", "*/*")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", w.Code, http.StatusUnauthorized)
	}
	var out map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got, _ := out["error"].(string); got != wantError {
		t.Fatalf("unexpected error: got %q want %q", got, wantError)
	}
}

func ptrInt(v int) *int { return &v }
