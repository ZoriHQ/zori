package middlewares

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"
	"zori/internal/cache"

	"github.com/labstack/echo/v4"
)

type CacheMiddleware struct {
	cacheService *cache.CacheService
}

func NewCacheMiddleware(cacheService *cache.CacheService) *CacheMiddleware {
	return &CacheMiddleware{
		cacheService: cacheService,
	}
}

type CacheConfig struct {
	TTL       time.Duration
	KeyPrefix string
	SkipCache func(c echo.Context) bool
}

func (m *CacheMiddleware) Middleware(config CacheConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.SkipCache != nil && config.SkipCache(c) {
				return next(c)
			}

			cacheKey := m.generateCacheKey(config.KeyPrefix, c)

			cachedValue, err := m.cacheService.Get(c.Request().Context(), cacheKey)
			if err == nil && cachedValue != nil {
				var cachedResponse interface{}
				if err := json.Unmarshal([]byte(*cachedValue), &cachedResponse); err == nil {
					return c.JSON(200, cachedResponse)
				}
			}

			rec := &responseCapture{
				ResponseWriter: c.Response().Writer,
				body:           []byte{},
			}
			c.Response().Writer = rec

			err = next(c)
			if err != nil {
				return err
			}

			if c.Response().Status >= 200 && c.Response().Status < 300 && len(rec.body) > 0 {
				// Probably, not ideal solution.
				// I'll think about it later :)
				var responseObj interface{}
				if unmarshalErr := json.Unmarshal(rec.body, &responseObj); unmarshalErr == nil {
					_ = m.cacheService.Set(c.Request().Context(), cacheKey, responseObj, config.TTL)
				}
			}

			return nil
		}
	}
}

// generateCacheKey creates a unique cache key based on the endpoint and query parameters
func (m *CacheMiddleware) generateCacheKey(prefix string, c echo.Context) string {
	// Get the path
	path := c.Path()

	// Get and sort query parameters for consistent cache keys
	queryParams := c.QueryParams()
	params := make([]string, 0, len(queryParams))
	for key, values := range queryParams {
		for _, value := range values {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}
	sort.Strings(params)

	// Create a deterministic query string
	queryString := ""
	if len(params) > 0 {
		queryString = "?" + url.QueryEscape(fmt.Sprint(params))
	}

	// Hash the full key to keep it reasonable length
	fullKey := fmt.Sprintf("%s:%s%s", prefix, path, queryString)
	hash := md5.Sum([]byte(fullKey))
	hashStr := hex.EncodeToString(hash[:])

	// Return prefix with hash for readability and uniqueness
	projectID := c.QueryParam("project_id")
	if projectID != "" {
		return fmt.Sprintf("%s:project:%s:hash:%s", prefix, projectID, hashStr)
	}

	return fmt.Sprintf("%s:hash:%s", prefix, hashStr)
}

// responseCapture is a custom response writer that captures the response body
type responseCapture struct {
	http.ResponseWriter
	body       []byte
	statusCode int
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body = append(r.body, b...)
	return r.ResponseWriter.Write(b)
}

func (r *responseCapture) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}
