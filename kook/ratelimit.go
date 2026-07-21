package kook

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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
	closed     atomic.Bool
}

// NewRateLimiter 创建新的速率限制器
// rate: 令牌补充间隔
// burst: 令牌桶容量
func NewRateLimiter(rate time.Duration, burst int) *RateLimiter {
	rl, err := NewRateLimiterWithError(rate, burst)
	if err != nil {
		panic(err)
	}
	return rl
}

// NewRateLimiterWithError 创建新的速率限制器，并返回配置错误。
func NewRateLimiterWithError(rate time.Duration, burst int) (*RateLimiter, error) {
	if rate <= 0 {
		return nil, fmt.Errorf("rate必须大于0")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("burst必须大于0")
	}
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

	return rl, nil
}

// Wait 等待获取令牌
func (rl *RateLimiter) Wait(ctx context.Context) error {
	if rl == nil || rl.closed.Load() {
		return ErrRateLimiterClosed
	}
	select {
	case <-rl.tokens:
		return nil
	case <-rl.done:
		return ErrRateLimiterClosed
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TryAcquire 尝试获取令牌，不等待
func (rl *RateLimiter) TryAcquire() bool {
	if rl == nil || rl.closed.Load() {
		return false
	}
	select {
	case <-rl.tokens:
		return true
	default:
		return false
	}
}

// Release 归还已获取的令牌。
func (rl *RateLimiter) Release() {
	if rl == nil || rl.closed.Load() {
		return
	}
	select {
	case rl.tokens <- struct{}{}:
	default:
	}
}

// Close 停止令牌补充协程。
func (rl *RateLimiter) Close() {
	if rl == nil {
		return
	}
	rl.closeOnce.Do(func() {
		rl.closed.Store(true)
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
	closed   atomic.Bool
}

// NewEndpointRateLimiter 创建端点级别的速率限制器
func NewEndpointRateLimiter(rate time.Duration, burst int) *EndpointRateLimiter {
	limiter, err := NewEndpointRateLimiterWithError(rate, burst)
	if err != nil {
		panic(err)
	}
	return limiter
}

// NewEndpointRateLimiterWithError 创建经过参数校验的端点限流器。
func NewEndpointRateLimiterWithError(rate time.Duration, burst int) (*EndpointRateLimiter, error) {
	if rate <= 0 {
		return nil, fmt.Errorf("rate必须大于0")
	}
	if burst <= 0 {
		return nil, fmt.Errorf("burst必须大于0")
	}
	return &EndpointRateLimiter{
		limiters: make(map[string]*RateLimiter),
		rate:     rate,
		burst:    burst,
	}, nil
}

// Wait 等待指定端点的令牌
func (erl *EndpointRateLimiter) Wait(ctx context.Context, endpoint string) error {
	limiter, err := erl.getLimiter(endpoint)
	if err != nil {
		return err
	}
	return limiter.Wait(ctx)
}

// TryAcquire 尝试获取指定端点的令牌
func (erl *EndpointRateLimiter) TryAcquire(endpoint string) bool {
	limiter, err := erl.getLimiter(endpoint)
	return err == nil && limiter.TryAcquire()
}

// Close 停止所有端点限流器。
func (erl *EndpointRateLimiter) Close() {
	if erl == nil || !erl.closed.CompareAndSwap(false, true) {
		return
	}
	erl.mu.Lock()
	defer erl.mu.Unlock()
	for _, limiter := range erl.limiters {
		limiter.Close()
	}
}

// getLimiter 获取或创建端点的速率限制器
func (erl *EndpointRateLimiter) getLimiter(endpoint string) (*RateLimiter, error) {
	if erl == nil || erl.closed.Load() {
		return nil, ErrRateLimiterClosed
	}
	erl.mu.RLock()
	limiter, exists := erl.limiters[endpoint]
	erl.mu.RUnlock()

	if exists {
		return limiter, nil
	}

	erl.mu.Lock()
	defer erl.mu.Unlock()
	if erl.closed.Load() {
		return nil, ErrRateLimiterClosed
	}

	// 双重检查
	if limiter, exists := erl.limiters[endpoint]; exists {
		return limiter, nil
	}

	// 创建新的限制器
	limiter = NewRateLimiter(erl.rate, erl.burst)
	erl.limiters[endpoint] = limiter
	return limiter, nil
}

// GlobalRateLimiter 全局速率限制器
type GlobalRateLimiter struct {
	generalLimiter  *RateLimiter
	endpointLimiter *EndpointRateLimiter
	closed          atomic.Bool
}

// RateLimitConfig 配置客户端的全局和端点令牌桶。
type RateLimitConfig struct {
	GlobalRate    time.Duration
	GlobalBurst   int
	EndpointRate  time.Duration
	EndpointBurst int
}

// DefaultRateLimitConfig 返回默认限流配置。
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		GlobalRate:    500 * time.Millisecond,
		GlobalBurst:   10,
		EndpointRate:  200 * time.Millisecond,
		EndpointBurst: 5,
	}
}

// NewGlobalRateLimiter 创建全局速率限制器
func NewGlobalRateLimiter() *GlobalRateLimiter {
	limiter, err := NewGlobalRateLimiterWithError(DefaultRateLimitConfig())
	if err != nil {
		panic(err)
	}
	return limiter
}

// NewGlobalRateLimiterWithError 创建经过完整配置校验的组合限流器。
func NewGlobalRateLimiterWithError(config RateLimitConfig) (*GlobalRateLimiter, error) {
	general, err := NewRateLimiterWithError(config.GlobalRate, config.GlobalBurst)
	if err != nil {
		return nil, fmt.Errorf("全局限流配置无效: %w", err)
	}
	endpoint, err := NewEndpointRateLimiterWithError(config.EndpointRate, config.EndpointBurst)
	if err != nil {
		general.Close()
		return nil, fmt.Errorf("端点限流配置无效: %w", err)
	}
	return &GlobalRateLimiter{generalLimiter: general, endpointLimiter: endpoint}, nil
}

// Wait 等待令牌（同时检查全局和端点限制）
func (grl *GlobalRateLimiter) Wait(ctx context.Context, endpoint string) error {
	if grl == nil || grl.closed.Load() {
		return ErrRateLimiterClosed
	}
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
	if grl == nil || grl.closed.Load() {
		return false
	}
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
	if grl == nil || !grl.closed.CompareAndSwap(false, true) {
		return
	}
	if grl.generalLimiter != nil {
		grl.generalLimiter.Close()
	}
	if grl.endpointLimiter != nil {
		grl.endpointLimiter.Close()
	}
}
