package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/submerge/submerge/backend/internal/apiresp"
)

type visitor struct {
	count    int
	windowAt time.Time
}

// rateLimiter 简易内存限流：每 IP 每分钟最多 max 次。
// 过期项由后台定时清理，避免每请求全表扫描带来的锁竞争。
type rateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
	max      int
}

// cleanupInterval 后台清理周期
const cleanupInterval = time.Minute

func (rl *rateLimiter) allow(ip string) bool {
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	v, ok := rl.visitors[ip]
	if !ok || now.Sub(v.windowAt) > time.Minute {
		rl.visitors[ip] = &visitor{count: 1, windowAt: now}
		return true
	}
	if v.count >= rl.max {
		return false
	}
	v.count++
	return true
}

// cleanup 定期移除过期窗口，控制 map 体量
func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for now := range ticker.C {
		rl.mu.Lock()
		for key, entry := range rl.visitors {
			if now.Sub(entry.windowAt) > time.Minute {
				delete(rl.visitors, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit 返回按 IP 限流的 gin 中间件
func RateLimit(max int) gin.HandlerFunc {
	if max <= 0 {
		max = 60
	}
	rl := &rateLimiter{
		visitors: map[string]*visitor{},
		max:      max,
	}
	go rl.cleanup()

	return func(c *gin.Context) {
		if rl.allow(c.ClientIP()) {
			c.Next()
			return
		}
		apiresp.Fail(c, http.StatusTooManyRequests, "rate_limited", "too many requests")
		c.Abort()
	}
}
