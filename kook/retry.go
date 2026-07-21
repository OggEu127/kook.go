package kook

import (
	"context"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries     int              // 最大重试次数
	InitialDelay   time.Duration    // 初始延迟
	MaxDelay       time.Duration    // 最大延迟
	BackoffFactor  float64          // 退避因子
	RetryableError func(error) bool // 判断错误是否可重试
	// RetryNonIdempotent 允许网络错误和5xx响应触发POST/PATCH重试。
	// 默认关闭，避免响应丢失时重复执行已经成功的写操作。
	RetryNonIdempotent bool
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		RetryableError: func(err error) bool {
			return IsRetryableError(err)
		},
	}
}

// IsRetryableError 判断错误是否可重试
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// 网络相关错误
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}

	// URL 错误
	if urlErr, ok := err.(*url.Error); ok {
		return IsRetryableError(urlErr.Err)
	}

	// 系统调用错误
	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*syscall.Errno); ok {
			switch *syscallErr {
			case syscall.ECONNRESET, syscall.ECONNREFUSED, syscall.ETIMEDOUT:
				return true
			}
		}
	}

	// KOOK API 错误
	if kookErr, ok := IsKOOKError(err); ok {
		return kookErr.IsRetryable()
	}

	return false
}

// IsRateLimitError 判断是否为速率限制错误
func IsRateLimitError(err error) bool {
	if kookErr, ok := IsKOOKError(err); ok {
		return kookErr.IsRateLimited()
	}
	return false
}

// GetRetryDelay 获取重试延迟时间
func GetRetryDelay(attempt int, config *RetryConfig) time.Duration {
	if attempt <= 0 {
		return config.InitialDelay
	}

	// 指数退避算法
	delay := float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt))

	// 限制最大延迟
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	return time.Duration(delay)
}

// RetryableFunc 可重试的函数类型
type RetryableFunc func(ctx context.Context) (*Response, error)

// DoWithRetry 执行带重试的操作
func DoWithRetry(ctx context.Context, fn RetryableFunc, config *RetryConfig, logger Logger) (*Response, error) {
	return doWithRetry(ctx, "", fn, config, logger)
}

func doRequestWithRetry(ctx context.Context, method string, fn RetryableFunc, config *RetryConfig, logger Logger) (*Response, error) {
	return doWithRetry(ctx, method, fn, config, logger)
}

func doWithRetry(ctx context.Context, method string, fn RetryableFunc, config *RetryConfig, logger Logger) (*Response, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}
	retryableError := config.RetryableError
	if retryableError == nil {
		retryableError = IsRetryableError
	}

	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := GetRetryDelay(attempt-1, config)

			if retryAfter := retryAfterFromError(lastErr); retryAfter > delay {
				delay = retryAfter
			}

			if logger != nil {
				if IsRateLimitError(lastErr) {
					logger.Warnf("遇到速率限制错误，等待 %v 后重试 (第 %d 次)", delay, attempt)
				} else {
					logger.Warnf("请求失败，等待 %v 后重试 (第 %d 次)", delay, attempt)
				}
			}

			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return nil, ctx.Err()
			}
		}

		resp, err := fn(ctx)
		if err == nil {
			if attempt > 0 && logger != nil {
				logger.Infof("重试成功 (第 %d 次尝试)", attempt+1)
			}
			return resp, nil
		}

		lastErr = err

		// 检查是否为可重试错误
		if !retryableError(err) || !requestMethodAllowsRetry(method, err, config) {
			if logger != nil {
				logger.Debugf("遇到不可重试错误")
			}
			return resp, err
		}

		if attempt == config.MaxRetries && logger != nil {
			logger.Errorf("重试失败，已达到最大重试次数 (%d)", config.MaxRetries)
		}
	}

	return nil, fmt.Errorf("重试失败: %w", lastErr)
}

func requestMethodAllowsRetry(method string, err error, config *RetryConfig) bool {
	if method == "" || config.RetryNonIdempotent {
		return true
	}
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodPut, http.MethodDelete:
		return true
	case http.MethodPost, http.MethodPatch:
		return IsRateLimitError(err)
	default:
		return false
	}
}

func retryAfterFromError(err error) time.Duration {
	if kookErr, ok := IsKOOKError(err); ok {
		return kookErr.RetryAfter
	}
	return 0
}

// Logger 日志接口
type Logger interface {
	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}

// ExtractRetryAfter 从 HTTP 响应头中提取 Retry-After 值
func ExtractRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	return parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
}

func parseRetryAfter(value string, now time.Time) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(value + "s"); err == nil && seconds > 0 {
		return seconds
	}
	if timestamp, err := http.ParseTime(value); err == nil {
		if duration := timestamp.Sub(now); duration > 0 {
			return duration
		}
	}
	return 0
}
