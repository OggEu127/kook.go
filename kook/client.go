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
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// BaseURL KOOK API 基础URL
	BaseURL = "https://www.kookapp.cn/api"
	// Version API版本
	Version = "v3"
	// UserAgent 用户代理
	UserAgent = "KOOK-Go-SDK/1.2.0"
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
	httpClient  *http.Client
	token       string
	tokenType   TokenType
	baseURL     string
	logger      *logrus.Logger
	rateLimiter *GlobalRateLimiter
	retryConfig *RetryConfig

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
	Blacklist     *BlacklistService
	Invite        *InviteService
	Asset         *AssetService
	Intimacy      *IntimacyService
	Friend        *FriendService
	Game          *GameService
	Badge         *BadgeService
	Thread        *ThreadService
	Voice         *VoiceService
	Template      *TemplateService
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
		c.rateLimiter = rateLimiter
	}
}

// WithoutRateLimit 禁用速率限制
func WithoutRateLimit() ClientOption {
	return func(c *Client) {
		c.rateLimiter = nil
	}
}

// WithRetryConfig 设置自定义重试配置
func WithRetryConfig(config *RetryConfig) ClientOption {
	return func(c *Client) {
		c.retryConfig = config
	}
}

// WithoutRetry 禁用重试
func WithoutRetry() ClientOption {
	return func(c *Client) {
		c.retryConfig = &RetryConfig{MaxRetries: 0}
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
	if token == "" {
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
		option(client)
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

	return client, nil
}

// Close 释放客户端内部后台资源。
func (c *Client) Close() error {
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
	// 使用重试机制执行请求
	return DoWithRetry(ctx, func(ctx context.Context) (*Response, error) {
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, NewKOOKErrorFromResponse(resp, body).WithContext(method, endpoint)
	}
	return &BinaryResponse{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		ETag:        resp.Header.Get("ETag"),
		Data:        body,
	}, nil
}

func (c *Client) doMultipartRequest(ctx context.Context, endpoint string, fields map[string]string, files map[string]multipartFile, query map[string]string) (*Response, error) {
	return DoWithRetry(ctx, func(ctx context.Context) (*Response, error) {
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
		"method":  http.MethodPost,
		"url":     requestURL,
		"headers": redactedHeaders(req.Header),
	}).Debugf("发送multipart API请求")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("请求失败")
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Errorf("读取响应失败")
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debugf("收到multipart API响应")

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := NewKOOKErrorFromResponse(resp, respBody).WithContext(http.MethodPost, endpoint)
		c.logger.WithError(err).Errorf("HTTP返回错误")
		return nil, err
	}

	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.WithError(err).Errorf("解析响应失败")
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}
	if response.Code != 0 {
		err := NewKOOKError(response.Code, response.Message).WithContext(http.MethodPost, endpoint)
		err.HTTPStatus = resp.StatusCode
		c.logger.WithError(err).Errorf("API返回错误")
		return &response, err
	}

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
		c.logger.WithField("params", string(jsonData)).Debugf("请求参数")
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
		"method":  method,
		"url":     requestURL,
		"headers": redactedHeaders(req.Header),
	}).Debugf("发送API请求")

	// 执行请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Errorf("请求失败")
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.WithError(err).Errorf("读取响应失败")
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   string(respBody),
	}).Debugf("收到API响应")

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		err := NewKOOKErrorFromResponse(resp, respBody).WithContext(method, endpoint)
		c.logger.WithError(err).Errorf("HTTP返回错误")
		return nil, err
	}

	// 解析响应
	var response Response
	if err := json.Unmarshal(respBody, &response); err != nil {
		c.logger.WithError(err).Errorf("解析响应失败")
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
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			if seconds, parseErr := time.ParseDuration(retryAfter + "s"); parseErr == nil {
				err = err.WithRetryAfter(seconds)
			}
		}

		err.HTTPStatus = resp.StatusCode

		c.logger.WithError(err).Errorf("API返回错误")
		return &response, err
	}

	c.logger.Infof("API请求成功: %s %s", method, requestURL)
	return &response, nil
}

func redactedHeaders(headers http.Header) http.Header {
	redacted := headers.Clone()
	if redacted.Get("Authorization") != "" {
		redacted.Set("Authorization", "[REDACTED]")
	}
	return redacted
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
