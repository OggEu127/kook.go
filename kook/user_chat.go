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

// GetUserChatList 获取私信聊天会话列表
func (s *UserChatService) GetUserChatList(ctx context.Context, page, pageSize int) (*UserChatListResponse, error) {
	query := make(map[string]string)
	if page > 0 {
		query["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 {
		query["page_size"] = strconv.Itoa(pageSize)
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

// GetUserChat 获取私信聊天会话详情
func (s *UserChatService) GetUserChat(ctx context.Context, chatCode string) (*UserChat, error) {
	if chatCode == "" {
		return nil, fmt.Errorf("私信会话Code不能为空")
	}

	resp, err := s.client.Get(ctx, "user-chat/view", map[string]string{
		"chat_code": chatCode,
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

// CreateUserChat 创建私信聊天会话
func (s *UserChatService) CreateUserChat(ctx context.Context, targetID string) (*UserChat, error) {
	if targetID == "" {
		return nil, fmt.Errorf("目标用户ID不能为空")
	}

	resp, err := s.client.Post(ctx, "user-chat/create", map[string]interface{}{
		"target_id": targetID,
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

// DeleteUserChat 删除私信聊天会话
func (s *UserChatService) DeleteUserChat(ctx context.Context, chatCode string) error {
	if chatCode == "" {
		return fmt.Errorf("私信会话Code不能为空")
	}

	_, err := s.client.Post(ctx, "user-chat/delete", map[string]interface{}{
		"chat_code": chatCode,
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
	Sort  map[string]int `json:"sort"`
}
