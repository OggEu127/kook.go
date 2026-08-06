package kook

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	defaultEcosystemHeartbeat = 30 * time.Second
	maxEcosystemBackoff       = 5 * time.Minute
	maxEcosystemResponseBytes = 1 << 20
)

// ErrEcosystemDisabled 表示客户端没有显式配置生态服务。
var ErrEcosystemDisabled = errors.New("KOOK Go SDK生态服务未启用")

// ReleaseChannel SDK发布通道。
type ReleaseChannel string

const (
	// ReleaseChannelStable 稳定发布通道。
	ReleaseChannelStable ReleaseChannel = "stable"
	// ReleaseChannelBeta 测试发布通道。
	ReleaseChannelBeta ReleaseChannel = "beta"
)

// EcosystemTransport 描述当前SDK客户端承载机器人的方式。
type EcosystemTransport string

const (
	// GatewayTransport 表示WebSocket网关机器人，由SDK自动管理。
	GatewayTransport EcosystemTransport = "gateway"
	// WebhookTransport 表示Webhook机器人，由调用方通过Start和Stop管理。
	WebhookTransport EcosystemTransport = "webhook"
)

// EcosystemOptions 配置可选的SDK生态服务。
type EcosystemOptions struct {
	BaseURL               string                `yaml:"base_url"`
	Channel               ReleaseChannel        `yaml:"channel"`
	ContributeToCommunity *bool                 `yaml:"contribute_to_community"`
	NoticeStatePath       string                `yaml:"notice_state_path,omitempty"`
	OnUpdateAvailable     func(SDKUpdateStatus) `yaml:"-"`
}

// SDKUpdateStatus 描述当前SDK相对发布清单的状态。
type SDKUpdateStatus struct {
	CurrentVersion          string         `json:"current_version"`
	LatestVersion           string         `json:"latest_version"`
	MinimumSupportedVersion string         `json:"minimum_supported_version"`
	Channel                 ReleaseChannel `json:"channel"`
	UpdateAvailable         bool           `json:"update_available"`
	Supported               bool           `json:"supported"`
	ReleaseURL              string         `json:"release_url"`
	Message                 string         `json:"message"`
	PublishedAt             time.Time      `json:"published_at"`
	Revision                string         `json:"revision"`
}

// OnlineInstanceStats 是公开的匿名在线实例统计。
type OnlineInstanceStats struct {
	OnlineInstances int64     `json:"online_instances"`
	AsOf            time.Time `json:"as_of"`
	LeaseSeconds    int       `json:"lease_seconds"`
	Definition      string    `json:"definition"`
}

type ecosystemHeartbeatRequest struct {
	InstanceID string             `json:"instance_id"`
	SDKVersion string             `json:"sdk_version"`
	GoVersion  string             `json:"go_version"`
	OS         string             `json:"os"`
	Arch       string             `json:"arch"`
	Channel    ReleaseChannel     `json:"channel"`
	Transport  EcosystemTransport `json:"transport"`
}

type ecosystemHeartbeatResponse struct {
	LeaseSeconds         int             `json:"lease_seconds"`
	NextHeartbeatSeconds int             `json:"next_heartbeat_seconds"`
	Update               SDKUpdateStatus `json:"update"`
}

// EcosystemService 提供版本检查和匿名在线实例上报。
type EcosystemService struct {
	client     *Client
	enabled    bool
	baseURL    string
	channel    ReleaseChannel
	instanceID string
	onUpdate   func(SDKUpdateStatus)
	contribute bool

	mu              sync.RWMutex
	cached          *SDKUpdateStatus
	lastNotified    string
	gatewayRefs     int
	webhookActive   bool
	workerCancel    context.CancelFunc
	workerDone      chan struct{}
	workerTransport EcosystemTransport
	leaseStarted    bool
	closed          bool
}

func newEcosystemService(client *Client, options *EcosystemOptions) (*EcosystemService, error) {
	service := &EcosystemService{client: client, channel: ReleaseChannelStable}
	if options == nil {
		return service, nil
	}
	baseURL, err := validateEcosystemBaseURL(options.BaseURL)
	if err != nil {
		return nil, err
	}
	channel := options.Channel
	if channel == "" {
		channel = ReleaseChannelStable
	}
	if !validReleaseChannel(channel) {
		return nil, fmt.Errorf("生态发布通道必须是stable或beta")
	}
	contribute := true
	if options.ContributeToCommunity != nil {
		contribute = *options.ContributeToCommunity
	}
	instanceID := ""
	if contribute {
		instanceID, err = randomInstanceID()
		if err != nil {
			return nil, fmt.Errorf("生成生态实例ID失败: %w", err)
		}
	}
	service.enabled = true
	service.baseURL = baseURL
	service.channel = channel
	service.instanceID = instanceID
	service.onUpdate = options.OnUpdateAvailable
	service.contribute = contribute
	if contribute {
		service.showCommunityContributionNotice(options.NoticeStatePath)
	}
	return service, nil
}

func validateEcosystemBaseURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("生态服务地址不能为空")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "", fmt.Errorf("生态服务地址无效")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("生态服务地址不能包含凭据、查询参数或片段")
	}
	host := parsed.Hostname()
	if parsed.Scheme != "https" && (parsed.Scheme != "http" || !isLoopbackHost(host)) {
		return "", fmt.Errorf("生态服务地址必须使用HTTPS；本地回环地址可使用HTTP")
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func validReleaseChannel(channel ReleaseChannel) bool {
	return channel == ReleaseChannelStable || channel == ReleaseChannelBeta
}

func randomInstanceID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(value[:]), nil
}

// CommunityContribution 返回可用于EcosystemOptions的显式社区贡献设置。
func CommunityContribution(enabled bool) *bool {
	return &enabled
}

// ContributionEnabled 报告当前客户端是否会发送匿名在线心跳。
func (s *EcosystemService) ContributionEnabled() bool {
	return s != nil && s.enabled && s.contribute
}

// CheckVersion 主动查询当前发布通道的SDK版本状态。
func (s *EcosystemService) CheckVersion(ctx context.Context) (*SDKUpdateStatus, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	query := url.Values{}
	query.Set("channel", string(s.channel))
	query.Set("current_version", SDKVersion)
	var status SDKUpdateStatus
	if err := s.requestJSON(ctx, http.MethodGet, "/v1/sdk/releases/latest?"+query.Encode(), nil, &status); err != nil {
		return nil, err
	}
	s.acceptUpdate(status)
	return &status, nil
}

// CachedVersionStatus 返回最近一次成功获取的版本状态。
func (s *EcosystemService) CachedVersionStatus() (SDKUpdateStatus, bool) {
	if s == nil {
		return SDKUpdateStatus{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil {
		return SDKUpdateStatus{}, false
	}
	return *s.cached, true
}

// GetOnlineStats 获取公开的近似在线SDK客户端实例数。
func (s *EcosystemService) GetOnlineStats(ctx context.Context) (*OnlineInstanceStats, error) {
	if err := s.available(); err != nil {
		return nil, err
	}
	var stats OnlineInstanceStats
	if err := s.requestJSON(ctx, http.MethodGet, "/v1/stats/online", nil, &stats); err != nil {
		return nil, err
	}
	return &stats, nil
}

// Start 为Webhook机器人启动匿名在线租约。Gateway连接会自动调用对应生命周期。
func (s *EcosystemService) Start(ctx context.Context, transport EcosystemTransport) error {
	if err := s.available(); err != nil {
		return err
	}
	if !s.contribute {
		return nil
	}
	if transport != WebhookTransport {
		return fmt.Errorf("Start仅接受WebhookTransport；Gateway连接由SDK自动管理")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrClientClosed
	}
	s.webhookActive = true
	s.leaseStarted = true
	s.ensureWorkerLocked(WebhookTransport)
	s.mu.Unlock()
	return nil
}

// Stop 停止Webhook租约；仍有Gateway连接时不会注销共享实例。
func (s *EcosystemService) Stop(ctx context.Context) error {
	if err := s.available(); err != nil {
		return err
	}
	if !s.contribute {
		return nil
	}
	s.mu.Lock()
	wasWebhookActive := s.webhookActive
	s.webhookActive = false
	shouldDelete := wasWebhookActive && s.gatewayRefs == 0
	var done <-chan struct{}
	if shouldDelete {
		done = s.stopWorkerLocked()
	}
	s.mu.Unlock()
	if shouldDelete {
		if err := waitEcosystemWorker(ctx, done); err != nil {
			return err
		}
		err := s.deleteLease(ctx)
		if err == nil {
			s.mu.Lock()
			if s.gatewayRefs == 0 && !s.webhookActive {
				s.leaseStarted = false
			}
			s.mu.Unlock()
		}
		return err
	}
	return nil
}

func (s *EcosystemService) available() error {
	if s == nil || !s.enabled {
		return ErrEcosystemDisabled
	}
	s.mu.RLock()
	closed := s.closed
	s.mu.RUnlock()
	if closed {
		return ErrClientClosed
	}
	return nil
}

func (s *EcosystemService) gatewayConnected() {
	if s == nil || !s.enabled || !s.contribute {
		return
	}
	s.mu.Lock()
	if !s.closed {
		s.gatewayRefs++
		s.leaseStarted = true
		s.ensureWorkerLocked(GatewayTransport)
	}
	s.mu.Unlock()
}

func (s *EcosystemService) gatewayDisconnected() {
	if s == nil || !s.enabled || !s.contribute {
		return
	}
	s.mu.Lock()
	if s.gatewayRefs > 0 {
		s.gatewayRefs--
	}
	idle := s.gatewayRefs == 0 && !s.webhookActive
	if idle {
		s.stopWorkerLocked()
	} else if s.gatewayRefs == 0 && s.webhookActive {
		s.workerTransport = WebhookTransport
	}
	s.mu.Unlock()
}

func (s *EcosystemService) unregisterIfIdle() {
	if s == nil || !s.enabled || !s.contribute {
		return
	}
	s.mu.RLock()
	idle := s.gatewayRefs == 0 && !s.webhookActive && !s.closed && s.leaseStarted
	done := s.workerDone
	s.mu.RUnlock()
	if idle {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = waitEcosystemWorker(ctx, done)
		if s.deleteLease(ctx) == nil {
			s.mu.Lock()
			if s.gatewayRefs == 0 && !s.webhookActive {
				s.leaseStarted = false
			}
			s.mu.Unlock()
		}
	}
}

func (s *EcosystemService) ensureWorkerLocked(transport EcosystemTransport) {
	if s.workerCancel != nil {
		if transport == GatewayTransport {
			s.workerTransport = transport
		}
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.workerCancel = cancel
	s.workerDone = done
	s.workerTransport = transport
	go s.runReporter(ctx, done)
}

func (s *EcosystemService) stopWorkerLocked() <-chan struct{} {
	done := s.workerDone
	if s.workerCancel != nil {
		s.workerCancel()
		s.workerCancel = nil
	}
	return done
}

func waitEcosystemWorker(ctx context.Context, done <-chan struct{}) error {
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *EcosystemService) runReporter(ctx context.Context, done chan struct{}) {
	defer close(done)
	delay := time.Duration(0)
	failures := 0
	for {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
		s.mu.RLock()
		transport := s.workerTransport
		s.mu.RUnlock()
		next, err := s.heartbeat(ctx, transport)
		if err != nil {
			failures++
			delay = ecosystemBackoff(failures)
			s.client.logger.WithError(err).Debug("SDK生态心跳失败")
			continue
		}
		failures = 0
		delay = next
	}
}

func ecosystemBackoff(failures int) time.Duration {
	if failures < 1 {
		return defaultEcosystemHeartbeat
	}
	shift := failures - 1
	if shift > 4 {
		shift = 4
	}
	base := defaultEcosystemHeartbeat * time.Duration(1<<shift)
	if base > maxEcosystemBackoff {
		base = maxEcosystemBackoff
	}
	var jitterByte [1]byte
	if _, err := rand.Read(jitterByte[:]); err == nil {
		jitter := time.Duration(int(jitterByte[0])%21) * base / 100
		base -= base / 10
		base += jitter
	}
	if base > maxEcosystemBackoff {
		base = maxEcosystemBackoff
	}
	return base
}

func (s *EcosystemService) heartbeat(ctx context.Context, transport EcosystemTransport) (time.Duration, error) {
	payload := ecosystemHeartbeatRequest{
		InstanceID: s.instanceID,
		SDKVersion: SDKVersion,
		GoVersion:  runtime.Version(),
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		Channel:    s.channel,
		Transport:  transport,
	}
	var response ecosystemHeartbeatResponse
	if err := s.requestJSON(ctx, http.MethodPost, "/v1/instances/heartbeat", payload, &response); err != nil {
		return 0, err
	}
	s.acceptUpdate(response.Update)
	next := time.Duration(response.NextHeartbeatSeconds) * time.Second
	if next < 10*time.Second || next > maxEcosystemBackoff {
		next = defaultEcosystemHeartbeat
	}
	return next, nil
}

func (s *EcosystemService) acceptUpdate(status SDKUpdateStatus) {
	s.mu.Lock()
	copyStatus := status
	s.cached = &copyStatus
	notify := status.UpdateAvailable && status.Revision != "" && status.Revision != s.lastNotified
	if notify {
		s.lastNotified = status.Revision
	}
	callback := s.onUpdate
	s.mu.Unlock()
	if !notify {
		return
	}
	s.client.logger.WithFields(map[string]interface{}{
		"current_version": status.CurrentVersion,
		"latest_version":  status.LatestVersion,
		"channel":         status.Channel,
		"supported":       status.Supported,
		"release_url":     status.ReleaseURL,
	}).Warn("发现新的KOOK Go SDK版本")
	if callback != nil {
		go func() {
			defer func() {
				if recover() != nil {
					s.client.logger.Error("SDK生态更新回调发生panic")
				}
			}()
			callback(status)
		}()
	}
}

func (s *EcosystemService) deleteLease(ctx context.Context) error {
	return s.requestJSON(ctx, http.MethodDelete, "/v1/instances/"+url.PathEscape(s.instanceID), nil, nil)
}

func (s *EcosystemService) requestJSON(ctx context.Context, method, path string, requestBody, responseBody interface{}) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("序列化生态请求失败: %w", err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("创建生态请求失败: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("生态请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	limited := io.LimitReader(resp.Body, maxEcosystemResponseBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("读取生态响应失败: %w", err)
	}
	if len(data) > maxEcosystemResponseBytes {
		return fmt.Errorf("生态响应超过1 MiB限制")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("生态服务返回HTTP %d", resp.StatusCode)
	}
	if responseBody == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, responseBody); err != nil {
		return fmt.Errorf("解析生态响应失败: %w", err)
	}
	return nil
}

func (s *EcosystemService) close() {
	if s == nil || !s.enabled || !s.contribute {
		return
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	leaseStarted := s.leaseStarted
	s.closed = true
	s.gatewayRefs = 0
	s.webhookActive = false
	done := s.stopWorkerLocked()
	s.mu.Unlock()
	if leaseStarted {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = waitEcosystemWorker(ctx, done)
		_ = s.deleteLease(ctx)
	}
}
