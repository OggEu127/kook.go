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

// GetUser 获取指定用户信息
func (s *UserService) GetUser(ctx context.Context, userID string, guildID string) (*User, error) {
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	query := map[string]string{
		"user_id": userID,
	}

	if guildID != "" {
		query["guild_id"] = guildID
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

// GetUserOnlineStatus 获取用户在线状态
//
// Deprecated: KOOK 官方 user/online 是 Webhook 模式机器人上线接口，
// 不是指定用户在线状态查询。请使用 GetOnlineStatus 查询机器人在线状态。
func (s *UserService) GetUserOnlineStatus(ctx context.Context, userID string) (bool, error) {
	return false, fmt.Errorf("GetUserOnlineStatus 已废弃：KOOK 官方没有指定用户在线状态查询接口")
}

// UpdateUserInfo 更新用户信息
//
// Deprecated: 当前 KOOK 官方用户接口未提供 user/update。
func (s *UserService) UpdateUserInfo(ctx context.Context, params UpdateUserParams) (*User, error) {
	return nil, fmt.Errorf("UpdateUserInfo 已废弃：KOOK 官方没有 user/update 接口")
}

// UpdateUserParams 更新用户信息参数
type UpdateUserParams struct {
	Username string `json:"username,omitempty"` // 用户名
	Avatar   string `json:"avatar,omitempty"`   // 头像（base64或URL）
	Banner   string `json:"banner,omitempty"`   // 横幅图片URL
}

// BlockUser 屏蔽用户
//
// Deprecated: 当前 KOOK 官方用户接口未提供 user/block。
func (s *UserService) BlockUser(ctx context.Context, userID string) error {
	return fmt.Errorf("BlockUser 已废弃：KOOK 官方没有 user/block 接口")
}

// UnblockUser 取消屏蔽用户
//
// Deprecated: 当前 KOOK 官方用户接口未提供 user/unblock。
func (s *UserService) UnblockUser(ctx context.Context, userID string) error {
	return fmt.Errorf("UnblockUser 已废弃：KOOK 官方没有 user/unblock 接口")
}

// GetBlockedUsers 获取被屏蔽的用户列表
//
// Deprecated: 当前 KOOK 官方用户接口未提供 user/blocked。
func (s *UserService) GetBlockedUsers(ctx context.Context) ([]User, error) {
	return nil, fmt.Errorf("GetBlockedUsers 已废弃：KOOK 官方没有 user/blocked 接口")
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
