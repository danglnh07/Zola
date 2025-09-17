package api

import (
	"sync"
	"time"
)

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
