package kook

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestLoadEcosystemOptionsDefaultsContributionToTrue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ecosystem.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
base_url: https://ecosystem.example.test
channel: beta
`), 0o600))

	options, err := LoadEcosystemOptions(path)
	require.NoError(t, err)
	require.Equal(t, "https://ecosystem.example.test", options.BaseURL)
	require.Equal(t, ReleaseChannelBeta, options.Channel)
	require.NotNil(t, options.ContributeToCommunity)
	require.True(t, *options.ContributeToCommunity)
}

func TestCommunityContributionFalseDisablesOnlyHeartbeats(t *testing.T) {
	var versionRequests atomic.Int32
	var heartbeatRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/sdk/releases/latest":
			versionRequests.Add(1)
			_, _ = io.WriteString(w, `{
				"current_version":"1.3.0",
				"latest_version":"1.3.0",
				"minimum_supported_version":"1.2.0",
				"channel":"stable",
				"update_available":false,
				"supported":true,
				"release_url":"https://example.test/releases/v1.3.0",
				"published_at":"2026-07-26T00:00:00Z",
				"revision":"stable-r1"
			}`)
		case "/v1/instances/heartbeat":
			heartbeatRequests.Add(1)
			http.Error(w, "unexpected heartbeat", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "ecosystem.yaml")
	require.NoError(t, os.WriteFile(path, []byte(
		"base_url: "+server.URL+"\ncontribute_to_community: false\n",
	), 0o600))
	options, err := LoadEcosystemOptions(path)
	require.NoError(t, err)

	client := NewClient("token", WithEcosystem(options), WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	require.False(t, client.Ecosystem.ContributionEnabled())
	require.NoError(t, client.Ecosystem.Start(context.Background(), WebhookTransport))
	time.Sleep(50 * time.Millisecond)
	require.Zero(t, heartbeatRequests.Load())

	status, err := client.Ecosystem.CheckVersion(context.Background())
	require.NoError(t, err)
	require.Equal(t, "1.3.0", status.LatestVersion)
	require.Equal(t, int32(1), versionRequests.Load())
	require.NoError(t, client.Ecosystem.Stop(context.Background()))
}

func TestLoadEcosystemOptionsRejectsUnknownAndMultipleDocuments(t *testing.T) {
	for name, content := range map[string]string{
		"unknown":   "base_url: https://example.test\nunknown: true\n",
		"documents": "base_url: https://example.test\n---\nbase_url: https://other.test\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "ecosystem.yaml")
			require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
			_, err := LoadEcosystemOptions(path)
			require.Error(t, err)
		})
	}
}

func TestProgrammaticCommunityContributionSetting(t *testing.T) {
	disabled := CommunityContribution(false)
	require.NotNil(t, disabled)
	require.False(t, *disabled)
}

func TestCommunityContributionNoticeIsShownOnlyOnce(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "ecosystem.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
base_url: https://ecosystem.example.test
contribute_to_community: true
`), 0o600))

	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	options, err := LoadEcosystemOptions(configPath)
	require.NoError(t, err)
	first := NewClient("token", WithLogger(logger), WithEcosystem(options), WithoutRateLimit(), WithoutRetry())
	require.NoError(t, first.Close())
	require.Contains(t, logs.String(), "匿名在线实例贡献默认开启")
	require.FileExists(t, configPath+".community-notice-v1")

	logs.Reset()
	options, err = LoadEcosystemOptions(configPath)
	require.NoError(t, err)
	second := NewClient("token", WithLogger(logger), WithEcosystem(options), WithoutRateLimit(), WithoutRetry())
	require.NoError(t, second.Close())
	require.NotContains(t, logs.String(), "匿名在线实例贡献默认开启")
}

func TestDisabledContributionDoesNotShowNotice(t *testing.T) {
	directory := t.TempDir()
	configPath := filepath.Join(directory, "ecosystem.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(`
base_url: https://ecosystem.example.test
contribute_to_community: false
`), 0o600))
	options, err := LoadEcosystemOptions(configPath)
	require.NoError(t, err)

	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	client := NewClient("token", WithLogger(logger), WithEcosystem(options), WithoutRateLimit(), WithoutRetry())
	require.NoError(t, client.Close())
	require.NotContains(t, logs.String(), "匿名在线实例贡献默认开启")
	_, err = os.Stat(configPath + ".community-notice-v1")
	require.ErrorIs(t, err, os.ErrNotExist)
}
