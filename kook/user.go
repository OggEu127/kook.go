package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// UserService 用户相关API服务
type UserService struct {
	client *Client
}

// GetMe 获取当前用户信息
func (s *UserService) GetMe(ctx context.Context) (*User, error) {
	resp, err := s.client.Get(ctx, "user/me", nil)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}

	return &user, nil
}

// UserViewParams 获取目标用户参数。
type UserViewParams struct {
	UserID  string
	GuildID string
}

// GetUser 获取指定用户信息
func (s *UserService) GetUser(ctx context.Context, params UserViewParams) (*User, error) {
	if params.UserID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	query := map[string]string{
		"user_id": params.UserID,
	}

	if params.GuildID != "" {
		query["guild_id"] = params.GuildID
	}

	resp, err := s.client.Get(ctx, "user/view", query)
	if err != nil {
		return nil, err
	}

	var user User
	if err := json.Unmarshal(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("解析用户信息失败: %w", err)
	}

	return &user, nil
}

// SetOnline 上线机器人（仅限Webhook使用）
func (s *UserService) SetOnline(ctx context.Context) error {
	_, err := s.client.Post(ctx, "user/online", nil)
	return err
}

// SetOffline 下线机器人（仅限Webhook使用）
func (s *UserService) SetOffline(ctx context.Context) error {
	_, err := s.client.Post(ctx, "user/offline", nil)
	return err
}

// GetOnlineStatus 获取机器人在线状态
func (s *UserService) GetOnlineStatus(ctx context.Context) (*OnlineStatus, error) {
	resp, err := s.client.Get(ctx, "user/get-online-status", nil)
	if err != nil {
		return nil, err
	}

	var status OnlineStatus
	if err := json.Unmarshal(resp.Data, &status); err != nil {
		return nil, fmt.Errorf("解析在线状态失败: %w", err)
	}

	return &status, nil
}

// OnlineStatus 在线状态信息
type OnlineStatus struct {
	Online   bool     `json:"online"`    // 是否在线
	OnlineOS []string `json:"online_os"` // 在线的平台列表
}
