package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/danglnh07/zola/db"
	"github.com/danglnh07/zola/service/security"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const claimsKey = "claims-key"

// Auth middleware: verify token, check token version, check if the role is allowed for that endpoint
func (server *Server) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Get the token from request header
		token := strings.TrimSpace(strings.TrimPrefix(ctx.Request.Header.Get("Authorization"), "Bearer"))
		if token == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"Missing Bearer token"})
			return
		}

		// Verify token
		claims, err := server.jwtService.VerifyToken(token)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"Invalid token: " + err.Error()})
			return
		}

		// Check if the token version is match with database
		var account db.Account
		result := server.queries.DB.First(&account, claims.ID)
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"Invalid token: ID not exists"})
			return
		}

		if claims.Version != int(account.TokenVersion) {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{"Invalid token: token version not match"})
			return
		}

		// Check token type
		path := ctx.FullPath()
		tokenType := security.TokenType(claims.TokenType)
		if tokenType != security.AccessToken && tokenType != security.RefreshToken {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{"Invalid token: invalid token type"})
			return
		}

		// Only the refresh endpoint need refresh token, all endpoint that need authentication need access token
		if path == "/api/auth/token/refresh" && tokenType == security.RefreshToken ||
			path != "/api/auth/token/refresh" && tokenType != security.RefreshToken {
			ctx.Set(claimsKey, claims)
			ctx.Next()
			return
		}

		ctx.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{"This token type is not suitable for this endpoint"})
	}
}

// CORS middleware
func (server *Server) CORSMiddlware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", fmt.Sprintf("http://%s", server.config.BaseURL))
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
		ctx.Next()
	}
}

// Rate limiter struct, used Token Bucket strategy
type RateLimiter struct {
	tokens     int
	maxToken   int
	refillRate time.Duration
	lastRefill time.Time
	mutex      sync.Mutex
}

// Constructor method for RateLimiter
func NewRateLimiter(maxToken int, refillRate time.Duration) *RateLimiter {
	return &RateLimiter{
		tokens:     maxToken,
		maxToken:   maxToken,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Method to check if the current request can pass on, by checking the available token
// while refill token if needed
func (limiter *RateLimiter) Allow() bool {
	// Use mutex to avoid race condition
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	// Refill token
	elapsed := time.Since(limiter.lastRefill)
	refill := int(elapsed / limiter.refillRate)
	if refill > 0 {
		limiter.tokens += refill
		// If tokens exceed max token, we flatten it down
		if limiter.tokens > limiter.maxToken {
			limiter.tokens = limiter.maxToken
		}
		limiter.lastRefill = time.Now()
	}

	// Consume token
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}

	// If no token available, simply refuse
	return false
}

// Rate limiting middleware
func (server *Server) RateLimitingMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !server.limiter.Allow() {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{"Too many request at a time"})
			return
		}

		ctx.Next()
	}
}
