package kook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OAuthClient 调用不需要 Bot Token 的 OAuth 接口。
type OAuthClient struct {
	httpClient *http.Client
	baseURL    string
}

// OAuthClientOption 配置 OAuthClient。
type OAuthClientOption func(*OAuthClient)

// WithOAuthHTTPClient 设置 OAuth HTTP 客户端。
func WithOAuthHTTPClient(client *http.Client) OAuthClientOption {
	return func(c *OAuthClient) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithOAuthBaseURL 设置 OAuth API 基础地址，默认与 BaseURL 相同。
func WithOAuthBaseURL(baseURL string) OAuthClientOption {
	return func(c *OAuthClient) {
		if strings.TrimSpace(baseURL) != "" {
			c.baseURL = strings.TrimRight(baseURL, "/")
		}
	}
}

// NewOAuthClient 创建 OAuth 客户端。
func NewOAuthClient(options ...OAuthClientOption) *OAuthClient {
	client := &OAuthClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    BaseURL,
	}
	for _, option := range options {
		option(client)
	}
	return client
}

// OAuthTokenParams 使用授权码换取 Access Token 的参数。
type OAuthTokenParams struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
	RedirectURI  string `json:"redirect_uri"`
}

// ExchangeToken 使用授权码换取 OAuth Access Token。
func (c *OAuthClient) ExchangeToken(ctx context.Context, params OAuthTokenParams) (*OAuthTokenResponse, error) {
	if params.GrantType == "" {
		params.GrantType = "authorization_code"
	}
	if params.GrantType != "authorization_code" {
		return nil, fmt.Errorf("grant_type 仅支持 authorization_code")
	}
	if params.ClientID == "" || params.ClientSecret == "" || params.Code == "" || params.RedirectURI == "" {
		return nil, fmt.Errorf("client_id、client_secret、code 和 redirect_uri 均不能为空")
	}

	payload, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("序列化OAuth参数失败: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, buildUnversionedURL(c.baseURL, "oauth2/token"), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("创建OAuth请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OAuth请求失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取OAuth响应失败: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, NewKOOKErrorFromResponse(resp, body).WithContext(http.MethodPost, "oauth2/token")
	}

	var result OAuthTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析OAuth Token失败: %w", err)
	}
	return &result, nil
}

// OAuthTokenResponse OAuth Token 响应。
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// UnmarshalJSON 兼容官方文档中的 expire_in 与示例中的 expires_in。
func (r *OAuthTokenResponse) UnmarshalJSON(data []byte) error {
	type responseAlias OAuthTokenResponse
	var value struct {
		responseAlias
		ExpireIn int `json:"expire_in"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = OAuthTokenResponse(value.responseAlias)
	if r.ExpiresIn == 0 {
		r.ExpiresIn = value.ExpireIn
	}
	return nil
}
