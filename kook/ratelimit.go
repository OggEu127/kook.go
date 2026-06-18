package kook

import (
	"context"
	"sync"
	"time"
)

// RateLimiter 速率限制器
type RateLimiter struct {
	tokens     chan struct{}
	refillMu   sync.Mutex
	lastRefill time.Time
	rate       time.Duration
	burst      int
	done       chan struct{}
	closeOnce  sync.Once
}

// NewRateLimiter 创建新的速率限制器
// rate: 令牌补充间隔
// burst: 令牌桶容量
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	rl := &RateLimiter{
		tokens:     make(chan struct{}, burst),
		lastRefill: time.Now(),
		rate:       rate,
		burst:      burst,
		done:       make(chan struct{}),
	}

	// 初始填满令牌桶
	for i := 0; i < burst; i++ {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}

	// 启动令牌补充协程
	go rl.refillLoop()

	return rl
}

// Wait 等待获取令牌
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-rl.tokens:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire 尝试获取令牌，不等待
func (rl *RateLimiter) TryAcquire() bool {
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Release 归还已获取的令牌。
func (rl *RateLimiter) Release() {
	select {
	case rl.tokens <- struct{}{}:
	default:
	}
}

// Close 停止令牌补充协程。
func (rl *RateLimiter) Close() {
	rl.closeOnce.Do(func() {
		close(rl.done)
	})
}

// refillLoop 令牌补充循环
func (rl *RateLimiter) refillLoop() {
	ticker := time.NewTicker(rl.rate)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.refillMu.Lock()
			// 尝试添加一个令牌
			select {
			case rl.tokens <- struct{}{}:
				// 成功添加令牌
			default:
				// 令牌桶已满
			}
			rl.lastRefill = time.Now()
			rl.refillMu.Unlock()
		case <-rl.done:
			return
		}
	}
}

// EndpointRateLimiter 端点级别的速率限制器
type EndpointRateLimiter struct {
	limiters map[string]*RateLimiter
	mu       sync.RWMutex
	rate     time.Duration
	burst    int
}

// NewEndpointRateLimiter 创建端点级别的速率限制器
func NewEndpointRateLimiter(rate time.Duration, burst int) *EndpointRateLimiter {
	return &EndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rate:     rate,
		burst:    burst,
	}
}

// Wait 等待指定端点的令牌
func (erl *EndpointRateLimiter) Wait(ctx context.Context, endpoint string) error {
	return erl.getLimiter(endpoint).Wait(ctx)
}

// TryAcquire 尝试获取指定端点的令牌
func (erl *EndpointRateLimiter) TryAcquire(endpoint string) bool {
	return erl.getLimiter(endpoint).TryAcquire()
}

// Close 停止所有端点限流器。
func (erl *EndpointRateLimiter) Close() {
	erl.mu.RLock()
	defer erl.mu.RUnlock()
	for _, limiter := range erl.limiters {
		limiter.Close()
	}
}

// getLimiter 获取或创建端点的速率限制器
func (erl *EndpointRateLimiter) getLimiter(endpoint string) *RateLimiter {
	erl.mu.RLock()
	limiter, exists := erl.limiters[endpoint]
	erl.mu.RUnlock()

	if exists {
		return limiter
	}

	erl.mu.Lock()
	defer erl.mu.Unlock()

	// 双重检查
	if limiter, exists := erl.limiters[endpoint]; exists {
		return limiter
	}

	// 创建新的限制器
	limiter = NewRateLimiter(erl.rate, erl.burst)
	erl.limiters[endpoint] = limiter
	return limiter
}

// GlobalRateLimiter 全局速率限制器
type GlobalRateLimiter struct {
	generalLimiter  *RateLimiter
	endpointLimiter *EndpointRateLimiter
}

// NewGlobalRateLimiter 创建全局速率限制器
func NewGlobalRateLimiter() *GlobalRateLimiter {
	return &GlobalRateLimiter{
		// KOOK API 全局限制：120 requests per minute
		generalLimiter: NewRateLimiter(500*time.Millisecond, 10),
		// 端点级别限制：更宽松一些
		endpointLimiter: NewEndpointRateLimiter(200*time.Millisecond, 5),
	}
}

// Wait 等待令牌（同时检查全局和端点限制）
func (grl *GlobalRateLimiter) Wait(ctx context.Context, endpoint string) error {
	// 先等待全局限制
	if err := grl.generalLimiter.Wait(ctx); err != nil {
		return err
	}
	// 再等待端点限制
	if err := grl.endpointLimiter.Wait(ctx, endpoint); err != nil {
		grl.generalLimiter.Release()
		return err
	}
	return nil
}

// TryAcquire 尝试获取令牌
func (grl *GlobalRateLimiter) TryAcquire(endpoint string) bool {
	// 需要同时满足全局和端点限制
	if !grl.generalLimiter.TryAcquire() {
		return false
	}
	if !grl.endpointLimiter.TryAcquire(endpoint) {
		// 如果端点限制失败，需要把全局令牌还回去
		grl.generalLimiter.Release()
		return false
	}
	return true
}

// Close 停止全局和端点限流器。
func (grl *GlobalRateLimiter) Close() {
	if grl == nil {
		return
	}
	if grl.generalLimiter != nil {
		grl.generalLimiter.Close()
	}
	if grl.endpointLimiter != nil {
		grl.endpointLimiter.Close()
	}
}
