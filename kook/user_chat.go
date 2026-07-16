package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// UserChatService 私信聊天会话相关API服务
type UserChatService struct {
	client *Client
}

type UserChatListParams struct {
	Page     *int
	PageSize *int
}

// GetUserChatList 获取私信聊天会话列表。
func (s *UserChatService) GetUserChatList(ctx context.Context, params UserChatListParams) (*UserChatListResponse, error) {
	query := make(map[string]string)
	if params.Page != nil {
		query["page"] = strconv.Itoa(*params.Page)
	}
	if params.PageSize != nil {
		query["page_size"] = strconv.Itoa(*params.PageSize)
	}

	resp, err := s.client.Get(ctx, "user-chat/list", query)
	if err != nil {
		return nil, err
	}

	var result UserChatListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析私信会话列表失败: %w", err)
	}

	return &result, nil
}

type UserChatViewParams struct {
	ChatCode string
}

// GetUserChat 获取私信聊天会话详情。
func (s *UserChatService) GetUserChat(ctx context.Context, params UserChatViewParams) (*UserChat, error) {
	if params.ChatCode == "" {
		return nil, fmt.Errorf("私信会话Code不能为空")
	}

	resp, err := s.client.Get(ctx, "user-chat/view", map[string]string{
		"chat_code": params.ChatCode,
	})
	if err != nil {
		return nil, err
	}

	var result UserChat
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析私信会话详情失败: %w", err)
	}

	return &result, nil
}

type UserChatCreateParams struct {
	TargetID string
}

// CreateUserChat 创建私信聊天会话。
func (s *UserChatService) CreateUserChat(ctx context.Context, params UserChatCreateParams) (*UserChat, error) {
	if params.TargetID == "" {
		return nil, fmt.Errorf("目标用户ID不能为空")
	}

	resp, err := s.client.Post(ctx, "user-chat/create", map[string]interface{}{
		"target_id": params.TargetID,
	})
	if err != nil {
		return nil, err
	}

	var result UserChat
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析私信会话创建结果失败: %w", err)
	}

	return &result, nil
}

type UserChatDeleteParams struct {
	ChatCode string
}

// DeleteUserChat 删除私信聊天会话。
func (s *UserChatService) DeleteUserChat(ctx context.Context, params UserChatDeleteParams) error {
	if params.ChatCode == "" {
		return fmt.Errorf("私信会话Code不能为空")
	}

	_, err := s.client.Post(ctx, "user-chat/delete", map[string]interface{}{
		"chat_code": params.ChatCode,
	})
	return err
}

// UserChat 私信聊天会话
type UserChat struct {
	Code            string `json:"code"`
	LastReadTime    int64  `json:"last_read_time"`
	LatestMsgTime   int64  `json:"latest_msg_time"`
	UnreadCount     int    `json:"unread_count"`
	IsFriend        bool   `json:"is_friend"`
	IsBlocked       bool   `json:"is_blocked"`
	IsTargetBlocked bool   `json:"is_target_blocked"`
	TargetInfo      User   `json:"target_info"`
}

// UserChatListResponse 私信聊天会话分页响应
type UserChatListResponse struct {
	Items []UserChat     `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}
