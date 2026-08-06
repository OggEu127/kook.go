package kook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// BaseURL KOOK API 基础URL
	BaseURL = "https://www.kookapp.cn/api"
	// Version API版本
	Version = "v3"
	// SDKVersion SDK语义化版本
	SDKVersion = "1.3.0"
	// UserAgent 用户代理
	UserAgent = "KOOK-Go-SDK/" + SDKVersion
)

// TokenType 鉴权类型
type TokenType string

const (
	// TokenTypeBot 机器人Token
	TokenTypeBot TokenType = "Bot"
	// TokenTypeBearer OAuth2 Token
	TokenTypeBearer TokenType = "Bearer"
)

// Client KOOK API客户端
type Client struct {
	httpClient       *http.Client
	token            string
	tokenType        TokenType
	baseURL          string
	logger           *logrus.Logger
	rateLimiter      *GlobalRateLimiter
	retryConfig      *RetryConfig
	closed           atomic.Bool
	configErr        error
	noRateLimit      bool
	ecosystemOptions *EcosystemOptions

	// API服务
	Gateway       *GatewayService
	User          *UserService
	Guild         *GuildService
	Channel       *ChannelService
	ChannelUser   *ChannelUserService
	Message       *MessageService
	UserChat      *UserChatService
	DirectMessage *DirectMessageService
	GuildRole     *RoleService
	GuildEmoji    *EmojiService
	// Role 与 Emoji 保留用于兼容 v1.1.1；新代码应使用 GuildRole/GuildEmoji。
	Role      *RoleService
	Emoji     *EmojiService
	Blacklist *BlacklistService
	Invite    *InviteService
	Asset     *AssetService
	Intimacy  *IntimacyService
	Friend    *FriendService
	Game      *GameService
	Badge     *BadgeService
	Thread    *ThreadService
	Voice     *VoiceService
	Template  *TemplateService
	Ecosystem *EcosystemService
	// 以下服务仅用于旧代码编译兼容，未确认端点会返回 ErrUnsupportedEndpoint。
	Region   *RegionService
	OAuth    *OAuthService
	Live     *LiveService
	Admin    *AdminService
	Security *SecurityService
	Item     *ItemService
	Order    *OrderService
	Coupon   *CouponService
	Boost    *BoostService
}

// ClientOption 客户端配置选项
type ClientOption func(*Client)

// WithHTTPClient 设置自定义HTTP客户端
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTokenType 设置Token类型
func WithTokenType(tokenType TokenType) ClientOption {
	return func(c *Client) {
		c.tokenType = tokenType
	}
}

// WithBaseURL 设置自定义基础URL
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithLogger 设置自定义日志器
func WithLogger(logger *logrus.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithRateLimiter 设置自定义速率限制器
func WithRateLimiter(rateLimiter *GlobalRateLimiter) ClientOption {
	return func(c *Client) {
		if c.rateLimiter != nil && c.rateLimiter != rateLimiter {
			c.rateLimiter.Close()
		}
		c.rateLimiter = rateLimiter
		c.configErr = nil
		c.noRateLimit = false
	}
}

// WithRateLimitConfig 使用经过校验的全局和端点限流配置。
func WithRateLimitConfig(config RateLimitConfig) ClientOption {
	return func(c *Client) {
		if c.rateLimiter != nil {
			c.rateLimiter.Close()
		}
		c.rateLimiter, c.configErr = NewGlobalRateLimiterWithError(config)
		c.noRateLimit = false
	}
}

// WithoutRateLimit 禁用速率限制
func WithoutRateLimit() ClientOption {
	return func(c *Client) {
		if c.rateLimiter != nil {
			c.rateLimiter.Close()
		}
		c.rateLimiter = nil
		c.configErr = nil
		c.noRateLimit = true
	}
}

// WithRetryConfig 设置自定义重试配置
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		if config == nil {
			c.retryConfig = nil
			return
		}
		cloned := *config
		c.retryConfig = &cloned
	}
}

// WithoutRetry 禁用重试
func WithoutRetry() ClientOption {
	return func(c *Client) {
		c.retryConfig = &RetryConfig{MaxRetries: 0}
	}
}

// WithEcosystem 显式启用KOOK Go SDK生态服务。未配置该选项时SDK不会发送生态请求。
func WithEcosystem(options EcosystemOptions) ClientOption {
	return func(c *Client) {
		cloned := options
		c.ecosystemOptions = &cloned
	}
}

// NewClient 创建新的KOOK客户端
func NewClient(token string, options ...ClientOption) *Client {
	client, err := NewClientWithError(token, options...)
	if err != nil {
		panic(err.Error())
	}
	return client
}

// NewClientWithError 创建新的KOOK客户端，并将配置错误返回给调用方。
func NewClientWithError(token string, options ...ClientOption) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("token不能为空")
	}

	// 默认HTTP客户端
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 默认日志器
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	client := &Client{
		httpClient:  httpClient,
		token:       token,
		tokenType:   TokenTypeBot,
		baseURL:     BaseURL,
		logger:      logger,
		rateLimiter: NewGlobalRateLimiter(),
		retryConfig: DefaultRetryConfig(),
	}

	// 应用选项
	for _, option := range options {
		if option == nil {
			client.rateLimiter.Close()
			return nil, fmt.Errorf("客户端配置选项不能为空")
		}
		option(client)
	}
	if err := client.validate(); err != nil {
		if client.rateLimiter != nil {
			client.rateLimiter.Close()
		}
		return nil, err
	}

	// 初始化API服务
	client.Gateway = &GatewayService{client: client}
	client.User = &UserService{client: client}
	client.Guild = &GuildService{client: client}
	client.Channel = &ChannelService{client: client}
	client.ChannelUser = &ChannelUserService{client: client}
	client.Message = &MessageService{client: client}
	client.UserChat = &UserChatService{client: client}
	client.DirectMessage = &DirectMessageService{client: client}
	client.GuildRole = &RoleService{client: client}
	client.GuildEmoji = &EmojiService{client: client}
	client.Role = client.GuildRole
	client.Emoji = client.GuildEmoji
	client.Blacklist = &BlacklistService{client: client}
	client.Invite = &InviteService{client: client}
	client.Asset = &AssetService{client: client}
	client.Intimacy = &IntimacyService{client: client}
	client.Friend = &FriendService{client: client}
	client.Game = &GameService{client: client}
	client.Badge = &BadgeService{client: client}
	client.Thread = &ThreadService{client: client}
	client.Voice = &VoiceService{client: client}
	client.Template = &TemplateService{client: client}
	client.Region = &RegionService{client: client}
	client.OAuth = &OAuthService{client: client}
	client.Live = &LiveService{client: client}
	client.Admin = &AdminService{client: client}
	client.Security = &SecurityService{client: client}
	client.Item = &ItemService{client: client}
	client.Order = &OrderService{client: client}
	client.Coupon = &CouponService{client: client}
	client.Boost = &BoostService{client: client}
	ecosystemService, ecosystemErr := newEcosystemService(client, client.ecosystemOptions)
	if ecosystemErr != nil {
		if client.rateLimiter != nil {
			client.rateLimiter.Close()
		}
		return nil, ecosystemErr
	}
	client.Ecosystem = ecosystemService

	return client, nil
}

func (c *Client) validate() error {
	if c.configErr != nil {
		return c.configErr
	}
	if c.httpClient == nil {
		return fmt.Errorf("HTTP客户端不能为空")
	}
	if c.logger == nil {
		return fmt.Errorf("日志器不能为空")
	}
	if c.baseURL == "" {
		return fmt.Errorf("基础URL不能为空")
	}
	baseURL, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("基础URL无效: %w", err)
	}
	if baseURL.Host == "" || (baseURL.Scheme != "http" && baseURL.Scheme != "https") {
		return fmt.Errorf("基础URL必须是完整的HTTP或HTTPS URL")
	}
	if c.tokenType != TokenTypeBot && c.tokenType != TokenTypeBearer {
		return fmt.Errorf("不支持的Token类型: %s", c.tokenType)
	}
	if c.rateLimiter == nil && !c.noRateLimit {
		return fmt.Errorf("限流器不能为空；如需禁用请使用WithoutRateLimit")
	}
	return validateRetryConfig(c.retryConfig)
}

func validateRetryConfig(config *RetryConfig) error {
	if config == nil {
		return fmt.Errorf("重试配置不能为空")
	}
	if config.MaxRetries < 0 {
		return fmt.Errorf("最大重试次数不能为负数")
	}
	if config.MaxRetries == 0 {
		return nil
	}
	if config.InitialDelay < 0 || config.MaxDelay < 0 || config.MaxDelay < config.InitialDelay {
		return fmt.Errorf("重试延迟配置无效")
	}
	if config.BackoffFactor <= 0 {
		return fmt.Errorf("退避因子必须大于0")
	}
	return nil
}

// Close 释放客户端内部后台资源。
func (c *Client) Close() error {
	if c == nil || !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	if c.Ecosystem != nil {
		c.Ecosystem.close()
	}
	if c.rateLimiter != nil {
		c.rateLimiter.Close()
	}
	return nil
}

// buildURL 构建完整的API URL
func (c *Client) buildURL(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "/")
	return fmt.Sprintf("%s/%s/%s", strings.TrimRight(c.baseURL, "/"), Version, endpoint)
}

func buildUnversionedURL(baseURL, endpoint string) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(endpoint, "/"))
}

// doRequest 执行HTTP请求
func (c *Client) doRequest(ctx context.Context, method, endpoint string, params map[string]interface{}, query map[string]string) (*Response, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}
	// 使用重试机制执行请求
	return doRequestWithRetry(ctx, method, func(ctx context.Context) (*Response, error) {
		return c.doSingleRequest(ctx, method, endpoint, params, query)
	}, c.retryConfig, c.logger)
}

// BinaryResponse 表示图片等非 JSON API 的响应。
type BinaryResponse struct {
	StatusCode  int
	ContentType string
	ETag        string
	Data        []byte
}

func (c *Client) doBinaryRequest(ctx context.Context, method, endpoint string, query map[string]string) (*BinaryResponse, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}
	var result *BinaryResponse
	_, err := doRequestWithRetry(ctx, method, func(ctx context.Context) (*Response, error) {
		response, requestErr := c.doSingleBinaryRequest(ctx, method, endpoint, query)
		if requestErr == nil {
			result = response
		}
		return nil, requestErr
	}, c.retryConfig, c.logger)
	return result, err
}

func (c *Client) doSingleBinaryRequest(ctx context.Context, method, endpoint string, query map[string]string) (*BinaryResponse, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, endpoint); err != nil {
			return nil, fmt.Errorf("等待速率限制失败: %w", err)
		}
	}
	requestURL := c.buildURL(endpoint)
	if len(query) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("解析URL失败: %w", err)
		}
		q := u.Query()
		for key, value := range query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.tokenType, c.token))
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept-Language", "zh-cn")
	c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL)}).Debug("发送二进制API请求")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL)}).Error("二进制API请求失败")
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode}).Error("读取二进制API响应失败")
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	fields := logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(body)}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		c.logger.WithFields(fields).Error("二进制API返回错误")
		return nil, NewKOOKErrorFromResponse(resp, body).WithContext(method, endpoint)
	}
	c.logger.WithFields(fields).Info("二进制API请求成功")
	return &BinaryResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        resp.Header.Get("ETag"),
		Data:        body,
	}, nil
}

func (c *Client) doMultipartRequest(ctx context.Context, endpoint string, fields map[string]string, files map[string]multipartFile, query map[string]string) (*Response, error) {
	if c.closed.Load() {
		return nil, ErrClientClosed
	}
	return doRequestWithRetry(ctx, http.MethodPost, func(ctx context.Context) (*Response, error) {
		return c.doSingleMultipartRequest(ctx, endpoint, fields, files, query)
	}, c.retryConfig, c.logger)
}

type multipartFile struct {
	FileName string
	Content  []byte
}

func (c *Client) doSingleMultipartRequest(ctx context.Context, endpoint string, fields map[string]string, files map[string]multipartFile, query map[string]string) (*Response, error) {
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, endpoint); err != nil {
			return nil, fmt.Errorf("等待速率限制失败: %w", err)
		}
	}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for name, value := range fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, fmt.Errorf("写入表单字段失败: %w", err)
		}
	}
	for field, file := range files {
		part, err := writer.CreateFormFile(field, file.FileName)
		if err != nil {
			return nil, fmt.Errorf("创建表单文件失败: %w", err)
		}
		if _, err := part.Write(file.Content); err != nil {
			return nil, fmt.Errorf("写入文件内容失败: %w", err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("关闭multipart表单失败: %w", err)
	}

	requestURL := c.buildURL(endpoint)
	if len(query) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("解析URL失败: %w", err)
		}
		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.tokenType, c.token))
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept-Language", "zh-cn")

	c.logger.WithFields(logrus.Fields{
		"method": http.MethodPost,
		"url":    sanitizedURL(requestURL),
		"bytes":  buf.Len(),
	}).Debugf("发送multipart API请求")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "bytes": buf.Len()}).Errorf("multipart API请求失败")
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "status": resp.StatusCode}).Errorf("读取multipart API响应失败")
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"method":     http.MethodPost,
		"url":        sanitizedURL(requestURL),
		"status":     resp.StatusCode,
		"request_id": resp.Header.Get("X-Request-Id"),
		"bytes":      len(respBody),
	}).Debugf("收到multipart API响应")

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := NewKOOKErrorFromResponse(resp, respBody).WithContext(http.MethodPost, endpoint)
		c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("multipart API返回错误")
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("解析multipart API响应失败")
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if response.Code != 0 {
		err := NewKOOKError(response.Code, response.Message).WithContext(http.MethodPost, endpoint)
		err.HTTPStatus = resp.StatusCode
		if requestID := resp.Header.Get("X-Request-Id"); requestID != "" {
			err = err.WithRequestID(requestID)
		}
		if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); retryAfter > 0 {
			err = err.WithRetryAfter(retryAfter)
		}
		c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("multipart API返回错误")
		return &response, err
	}

	c.logger.WithFields(logrus.Fields{"method": http.MethodPost, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Info("multipart API请求成功")
	return &response, nil
}

// doSingleRequest 执行单次HTTP请求
func (c *Client) doSingleRequest(ctx context.Context, method, endpoint string, params map[string]interface{}, query map[string]string) (*Response, error) {
	// 应用速率限制
	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx, endpoint); err != nil {
			return nil, fmt.Errorf("等待速率限制失败: %w", err)
		}
	}

	requestURL := c.buildURL(endpoint)

	// 添加查询参数
	if len(query) > 0 {
		u, err := url.Parse(requestURL)
		if err != nil {
			return nil, fmt.Errorf("解析URL失败: %w", err)
		}

		q := u.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		requestURL = u.String()
	}

	var body io.Reader
	if params != nil {
		jsonData, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("序列化请求参数失败: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
		c.logger.WithField("body_bytes", len(jsonData)).Debugf("请求参数已序列化")
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", c.tokenType, c.token))
	req.Header.Set("User-Agent", UserAgent)
	if params != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept-Language", "zh-cn")

	c.logger.WithFields(logrus.Fields{
		"method": method,
		"url":    sanitizedURL(requestURL),
	}).Debugf("发送API请求")

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL)}).Errorf("请求失败")
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode}).Errorf("读取响应失败")
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"method":     method,
		"url":        sanitizedURL(requestURL),
		"status":     resp.StatusCode,
		"bytes":      len(respBody),
		"request_id": resp.Header.Get("X-Request-Id"),
	}).Debugf("收到API响应")

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := NewKOOKErrorFromResponse(resp, respBody).WithContext(method, endpoint)
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("HTTP返回错误")
		return nil, err
	}

	// 解析响应
	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("解析响应失败")
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 检查API错误
	if response.Code != 0 {
		err := NewKOOKError(response.Code, response.Message).
			WithContext(method, endpoint)

		// 从响应头中提取请求ID
		if requestID := resp.Header.Get("X-Request-Id"); requestID != "" {
			err = err.WithRequestID(requestID)
		}

		// 从响应头中提取重试延迟
		if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now()); retryAfter > 0 {
			err = err.WithRetryAfter(retryAfter)
		}

		err.HTTPStatus = resp.StatusCode

		c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Errorf("API返回错误")
		return &response, err
	}

	c.logger.WithFields(logrus.Fields{"method": method, "url": sanitizedURL(requestURL), "status": resp.StatusCode, "request_id": resp.Header.Get("X-Request-Id"), "bytes": len(respBody)}).Infof("API请求成功")
	return &response, nil
}

func sanitizedURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "[INVALID URL]"
	}
	query := u.Query()
	for key := range query {
		switch strings.ToLower(key) {
		case "access_token", "token", "code", "client_secret", "ticket", "session_id", "gateway_token", "authorization":
			query.Set(key, "[REDACTED]")
		}
	}
	u.RawQuery = query.Encode()
	if u.User != nil {
		u.User = url.User("[REDACTED]")
	}
	u.Fragment = ""
	return u.String()
}

// Get 发送GET请求
func (c *Client) Get(ctx context.Context, endpoint string, query map[string]string) (*Response, error) {
	return c.doRequest(ctx, "GET", endpoint, nil, query)
}

// Post 发送POST请求
func (c *Client) Post(ctx context.Context, endpoint string, params map[string]interface{}) (*Response, error) {
	return c.doRequest(ctx, "POST", endpoint, params, nil)
}

// Put 发送PUT请求
func (c *Client) Put(ctx context.Context, endpoint string, params map[string]interface{}) (*Response, error) {
	return c.doRequest(ctx, "PUT", endpoint, params, nil)
}

// Delete 发送DELETE请求
func (c *Client) Delete(ctx context.Context, endpoint string, params map[string]interface{}) (*Response, error) {
	return c.doRequest(ctx, "DELETE", endpoint, params, nil)
}

// Response API响应结构
type Response struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}
