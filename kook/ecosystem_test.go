package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestEcosystemIsOptInAndValidatesEndpoint(t *testing.T) {
	client := NewClient("token", WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	_, err := client.Ecosystem.CheckVersion(context.Background())
	require.ErrorIs(t, err, ErrEcosystemDisabled)

	_, err = NewClientWithError("token", WithEcosystem(EcosystemOptions{BaseURL: "http://example.com"}))
	require.ErrorContains(t, err, "HTTPS")
	_, err = NewClientWithError("token", WithEcosystem(EcosystemOptions{
		BaseURL: "https://example.com", Channel: ReleaseChannel("nightly"),
	}))
	require.ErrorContains(t, err, "stable或beta")
}

func TestEcosystemVersionCheckCacheCallbackAndPrivacy(t *testing.T) {
	var callbackCount atomic.Int32
	callbackValue := make(chan SDKUpdateStatus, 1)
	var requestMu sync.Mutex
	var heartbeatBody map[string]interface{}
	var authorization string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestMu.Lock()
		authorization = r.Header.Get("Authorization")
		requestMu.Unlock()
		switch {
		case r.URL.Path == "/v1/sdk/releases/latest":
			require.Equal(t, "beta", r.URL.Query().Get("channel"))
			require.Equal(t, SDKVersion, r.URL.Query().Get("current_version"))
			writeSDKUpdate(t, w, "revision-1")
		case r.URL.Path == "/v1/instances/heartbeat":
			var decoded map[string]interface{}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&decoded))
			requestMu.Lock()
			heartbeatBody = decoded
			requestMu.Unlock()
			_, _ = io.WriteString(w, `{
				"lease_seconds":90,
				"next_heartbeat_seconds":300,
				"update":{
					"current_version":"1.3.0",
					"latest_version":"1.4.0-beta.1",
					"minimum_supported_version":"1.2.0",
					"channel":"beta",
					"update_available":true,
					"supported":true,
					"release_url":"https://example.test/releases/v1.4.0-beta.1",
					"message":"beta",
					"published_at":"2026-07-26T00:00:00Z",
					"revision":"revision-1"
				}
			}`)
		case strings.HasPrefix(r.URL.Path, "/v1/instances/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient("super-secret-token",
		WithEcosystem(EcosystemOptions{
			BaseURL:         server.URL,
			Channel:         ReleaseChannelBeta,
			NoticeStatePath: filepath.Join(t.TempDir(), "notice"),
			OnUpdateAvailable: func(status SDKUpdateStatus) {
				callbackCount.Add(1)
				callbackValue <- status
			},
		}),
		WithoutRateLimit(),
		WithoutRetry(),
	)
	defer func() { _ = client.Close() }()

	status, err := client.Ecosystem.CheckVersion(context.Background())
	require.NoError(t, err)
	require.True(t, status.UpdateAvailable)
	require.Equal(t, "1.4.0-beta.1", status.LatestVersion)
	require.Equal(t, "revision-1", status.Revision)
	select {
	case callback := <-callbackValue:
		require.Equal(t, status.Revision, callback.Revision)
	case <-time.After(time.Second):
		t.Fatal("更新回调未执行")
	}
	_, err = client.Ecosystem.CheckVersion(context.Background())
	require.NoError(t, err)
	require.Eventually(t, func() bool { return callbackCount.Load() == 1 }, time.Second, time.Millisecond)
	cached, ok := client.Ecosystem.CachedVersionStatus()
	require.True(t, ok)
	require.Equal(t, "revision-1", cached.Revision)

	require.NoError(t, client.Ecosystem.Start(context.Background(), WebhookTransport))
	require.Eventually(t, func() bool {
		requestMu.Lock()
		defer requestMu.Unlock()
		return heartbeatBody != nil
	}, time.Second, time.Millisecond)
	requestMu.Lock()
	bodySnapshot := heartbeatBody
	authorizationSnapshot := authorization
	requestMu.Unlock()
	require.Empty(t, authorizationSnapshot)
	encoded, err := json.Marshal(bodySnapshot)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "super-secret-token")
	require.NotContains(t, string(encoded), "bot_id")
	require.NotContains(t, string(encoded), "guild")
	require.Len(t, bodySnapshot["instance_id"], 32)
	require.Equal(t, SDKVersion, bodySnapshot["sdk_version"])
	require.Equal(t, "webhook", bodySnapshot["transport"])
	require.NoError(t, client.Ecosystem.Stop(context.Background()))
}

func writeSDKUpdate(t *testing.T, w http.ResponseWriter, revision string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_, err := fmt.Fprintf(w, `{
		"current_version":"1.3.0",
		"latest_version":"1.4.0-beta.1",
		"minimum_supported_version":"1.2.0",
		"channel":"beta",
		"update_available":true,
		"supported":true,
		"release_url":"https://example.test/releases/v1.4.0-beta.1",
		"message":"beta",
		"published_at":"2026-07-26T00:00:00Z",
		"revision":%q
	}`, revision)
	require.NoError(t, err)
}

func TestEcosystemGatewayLifecycleAndFailureIsolation(t *testing.T) {
	var heartbeats atomic.Int32
	var deletes atomic.Int32
	ecosystemServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			heartbeats.Add(1)
			http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
		case http.MethodDelete:
			deletes.Add(1)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer ecosystemServer.Close()

	var gatewayURL string
	keepOpen := make(chan struct{})
	var closeKeepOpen sync.Once
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	kookServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v3/gateway/index" {
			_, _ = fmt.Fprintf(w, `{"code":0,"data":{"url":%q}}`, gatewayURL)
			return
		}
		connection, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() { _ = connection.Close() }()
		hello, _ := json.Marshal(HelloMessage{Code: 0, SessionID: "session"})
		require.NoError(t, connection.WriteJSON(WebSocketMessage{S: SignalHello, D: hello}))
		<-keepOpen
	}))
	defer kookServer.Close()
	gatewayURL = "ws" + strings.TrimPrefix(kookServer.URL, "http") + "/gateway"

	client := NewClient("token",
		WithBaseURL(kookServer.URL+"/api"),
		WithEcosystem(EcosystemOptions{
			BaseURL: ecosystemServer.URL, NoticeStatePath: filepath.Join(t.TempDir(), "notice"),
		}),
		WithoutRateLimit(),
		WithoutRetry(),
	)
	defer func() { _ = client.Close() }()
	require.Zero(t, heartbeats.Load())
	ws := NewWebSocketClient(client, false)
	require.NoError(t, ws.Connect())
	require.Eventually(t, func() bool { return heartbeats.Load() == 1 }, time.Second, time.Millisecond)
	require.NoError(t, ws.Close())
	closeKeepOpen.Do(func() { close(keepOpen) })
	require.Eventually(t, func() bool { return deletes.Load() == 1 }, time.Second, time.Millisecond)
}

func TestEcosystemOnlineStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/stats/online", r.URL.Path)
		_, _ = io.WriteString(w, `{
			"online_instances":12,
			"as_of":"2026-07-26T00:00:00Z",
			"lease_seconds":90,
			"definition":"active_sdk_client_instances_within_lease_window"
		}`)
	}))
	defer server.Close()
	client := NewClient("token", WithEcosystem(EcosystemOptions{
		BaseURL: server.URL, NoticeStatePath: filepath.Join(t.TempDir(), "notice"),
	}), WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	stats, err := client.Ecosystem.GetOnlineStats(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(12), stats.OnlineInstances)
	require.Equal(t, 90, stats.LeaseSeconds)
}

func TestEcosystemBackoffNeverExceedsMaximum(t *testing.T) {
	for failures := 1; failures < 20; failures++ {
		require.LessOrEqual(t, ecosystemBackoff(failures), maxEcosystemBackoff)
	}
}
