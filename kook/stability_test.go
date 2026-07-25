package kook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (fn roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

type retryTimeoutError struct{}

func (retryTimeoutError) Error() string   { return "temporary timeout" }
func (retryTimeoutError) Timeout() bool   { return true }
func (retryTimeoutError) Temporary() bool { return true }

func TestWrappedTransportErrorUsesMethodAwareRetryPolicy(t *testing.T) {
	newHTTPClient := func(calls *atomic.Int32) *http.Client {
		return &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
			if calls.Add(1) == 1 {
				transportErr := &url.Error{Op: request.Method, URL: request.URL.String(), Err: retryTimeoutError{}}
				return nil, fmt.Errorf("wrapped transport error: %w", transportErr)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(`{"code":0,"data":{}}`)),
				Request:    request,
			}, nil
		})}
	}

	t.Run("get retries wrapped timeout", func(t *testing.T) {
		var calls atomic.Int32
		client := NewClient("token", WithHTTPClient(newHTTPClient(&calls)), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()

		_, err := client.Get(context.Background(), "read", nil)
		require.NoError(t, err)
		require.Equal(t, int32(2), calls.Load())
	})

	t.Run("post does not retry wrapped timeout by default", func(t *testing.T) {
		var calls atomic.Int32
		client := NewClient("token", WithHTTPClient(newHTTPClient(&calls)), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()

		_, err := client.Post(context.Background(), "write", map[string]interface{}{})
		require.Error(t, err)
		require.Equal(t, int32(1), calls.Load())
	})
}

func TestNonIdempotentRequestDoesNotRetryServerError(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, `{"code":50300,"message":"unavailable"}`)
	}))
	defer server.Close()

	client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
		MaxRetries: 3, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
	}))
	defer func() { _ = client.Close() }()

	_, err := client.Post(context.Background(), "write", map[string]interface{}{"value": 1})
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}

func TestIdempotentRequestRetriesAndPostRetriesRateLimit(t *testing.T) {
	t.Run("get", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"code":50300,"message":"unavailable"}`)
				return
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
		}))
		defer server.Close()
		client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()
		_, err := client.Get(context.Background(), "read", nil)
		require.NoError(t, err)
		require.Equal(t, int32(2), calls.Load())
	})

	t.Run("post-429", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.Header().Set("Retry-After", "0.01")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = io.WriteString(w, `{"code":42900,"message":"limited"}`)
				return
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
		}))
		defer server.Close()
		client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()
		started := time.Now()
		_, err := client.Post(context.Background(), "write", map[string]interface{}{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, time.Since(started), 10*time.Millisecond)
		require.Equal(t, int32(2), calls.Load())
	})
}

func TestExplicitNonIdempotentRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"code":50300,"message":"unavailable"}`)
			return
		}
		_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
		MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1, RetryNonIdempotent: true,
	}))
	defer func() { _ = client.Close() }()

	_, err := client.Post(context.Background(), "write", map[string]interface{}{})
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestBinaryAndMultipartUseRetryPolicy(t *testing.T) {
	t.Run("binary-get", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = io.WriteString(w, `{"code":50300,"message":"unavailable"}`)
				return
			}
			_, _ = io.WriteString(w, "badge")
		}))
		defer server.Close()
		client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()

		response, err := client.Badge.GetGuildBadge(context.Background(), BadgeParams{GuildID: "guild"})
		require.NoError(t, err)
		require.Equal(t, []byte("badge"), response.Data)
		require.Equal(t, int32(2), calls.Load())
	})

	t.Run("multipart-429", func(t *testing.T) {
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.Header().Set("Retry-After", "0.001")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = io.WriteString(w, `{"code":42900,"message":"limited"}`)
				return
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"url":"asset"}}`)
		}))
		defer server.Close()
		client := NewClient("token", WithBaseURL(server.URL), WithoutRateLimit(), WithRetryConfig(&RetryConfig{
			MaxRetries: 1, InitialDelay: time.Millisecond, MaxDelay: time.Millisecond, BackoffFactor: 1,
		}))
		defer func() { _ = client.Close() }()

		asset, err := client.Asset.Create(context.Background(), AssetCreateParams{FileName: "asset", Content: []byte("data")})
		require.NoError(t, err)
		require.Equal(t, "asset", asset.URL)
		require.Equal(t, int32(2), calls.Load())
	})
}

func TestAllHTTPTransportsShareRateLimiter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"data":{}}`)
	}))
	defer server.Close()
	client := NewClient("token", WithBaseURL(server.URL), WithoutRetry(), WithRateLimitConfig(RateLimitConfig{
		GlobalRate: time.Hour, GlobalBurst: 1, EndpointRate: time.Hour, EndpointBurst: 1,
	}))
	defer func() { _ = client.Close() }()

	_, err := client.Get(context.Background(), "first", nil)
	require.NoError(t, err)
	for _, request := range []func(context.Context) error{
		func(ctx context.Context) error {
			_, err := client.Badge.GetGuildBadge(ctx, BadgeParams{GuildID: "guild"})
			return err
		},
		func(ctx context.Context) error {
			_, err := client.Asset.Create(ctx, AssetCreateParams{FileName: "asset", Content: []byte("data")})
			return err
		},
	} {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		err := request(ctx)
		cancel()
		require.ErrorIs(t, err, context.DeadlineExceeded)
	}
}

func TestSensitiveQueryIsRedactedFromLogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"code":0,"data":{"msg_id":"m"}}`)
	}))
	defer server.Close()

	var output bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&output)
	logger.SetLevel(logrus.DebugLevel)
	client := NewClient("bot-secret", WithBaseURL(server.URL), WithLogger(logger), WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()

	_, err := client.Message.SendPipe(context.Background(), SendPipeMessageParams{AccessToken: "pipe-secret", Content: "private-message"})
	require.NoError(t, err)
	logs := output.String()
	require.NotContains(t, logs, "pipe-secret")
	require.NotContains(t, logs, "bot-secret")
	require.NotContains(t, logs, "private-message")
	require.Contains(t, logs, "%5BREDACTED%5D")
}

func TestClientAndRateLimiterCloseWakeWaiters(t *testing.T) {
	limiter := NewRateLimiter(time.Hour, 1)
	require.NoError(t, limiter.Wait(context.Background()))
	waitResult := make(chan error, 1)
	go func() { waitResult <- limiter.Wait(context.Background()) }()
	limiter.Close()
	select {
	case err := <-waitResult:
		require.ErrorIs(t, err, ErrRateLimiterClosed)
	case <-time.After(time.Second):
		t.Fatal("closed rate limiter did not wake waiter")
	}

	client := NewClient("token", WithoutRateLimit(), WithoutRetry())
	require.NoError(t, client.Close())
	require.NoError(t, client.Close())
	_, err := client.Get(context.Background(), "user/me", nil)
	require.ErrorIs(t, err, ErrClientClosed)
}

func TestClientConfigurationValidation(t *testing.T) {
	_, err := NewClientWithError("token", WithHTTPClient(nil))
	require.ErrorContains(t, err, "HTTP客户端")
	_, err = NewClientWithError("token", WithLogger(nil))
	require.ErrorContains(t, err, "日志器")
	_, err = NewClientWithError("token", WithRetryConfig(&RetryConfig{MaxRetries: -1}))
	require.ErrorContains(t, err, "最大重试")
	_, err = NewClientWithError("token", WithRateLimiter(nil))
	require.ErrorContains(t, err, "限流器")
	_, err = NewClientWithError("token", WithRateLimitConfig(RateLimitConfig{}))
	require.ErrorContains(t, err, "限流")
	_, err = NewEndpointRateLimiterWithError(0, 1)
	require.Error(t, err)
	require.Panics(t, func() { NewRateLimiter(0, 1) })
}

func TestRetryAfterHTTPDate(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	value := now.Add(2 * time.Second).Format(http.TimeFormat)
	delay := parseRetryAfter(value, now)
	require.Equal(t, 2*time.Second, delay)
	require.Zero(t, parseRetryAfter("invalid", now))
}

func TestRetryWaitCanBeCanceled(t *testing.T) {
	var calls atomic.Int32
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	started := time.Now()
	_, err := doRequestWithRetry(ctx, http.MethodPost, func(context.Context) (*Response, error) {
		calls.Add(1)
		return nil, (&KOOKError{
			Code:       int(ErrorCodeTooManyRequests),
			HTTPStatus: http.StatusTooManyRequests,
		}).WithRetryAfter(time.Minute)
	}, &RetryConfig{
		MaxRetries: 1, InitialDelay: time.Minute, MaxDelay: time.Minute, BackoffFactor: 1,
	}, nil)

	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, int32(1), calls.Load())
	require.Less(t, time.Since(started), time.Second)
}

func TestSanitizedURLHandlesSensitiveKeysCaseInsensitively(t *testing.T) {
	result := sanitizedURL("https://user:password@example.test/path?ACCESS_TOKEN=secret&session_id=opaque-session-secret&safe=value#token")
	require.False(t, strings.Contains(result, "secret"))
	require.NotContains(t, result, "password")
	require.NotContains(t, result, "opaque-session-secret")
	require.NotContains(t, result, "#token")
	require.True(t, strings.Contains(result, "safe=value"))
	require.True(t, errors.Is(ErrUnsupportedEndpoint, ErrUnsupportedEndpoint))
}

func TestSanitizedURLRedactsInviteCredentials(t *testing.T) {
	result := sanitizedURL("https://www.kookapp.cn/api/v3/invite/invitees?id=invite-secret&invite_url=https%3A%2F%2Fkook.vip%2Finvite-url-secret&guild_id=guild")

	require.NotContains(t, result, "invite-secret")
	require.NotContains(t, result, "invite-url-secret")
	require.Contains(t, result, "guild_id=guild")
	require.Contains(t, result, "%5BREDACTED%5D")
	require.Contains(t, sanitizedURL("https://www.kookapp.cn/api/v3/channel/view?id=resource-id"), "id=resource-id")
}
