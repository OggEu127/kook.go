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
func (s *UserChatService) GetUserChatList(ctx context.Context, args ...any) (*UserChatListResponse, error) {
	params, err := compatParams("GetUserChatList", args, func(args []any) (UserChatListParams, bool) {
		if len(args) != 2 {
			return UserChatListParams{}, false
		}
		page, okPage := compatInt(args[0])
		pageSize, okPageSize := compatInt(args[1])
		return UserChatListParams{Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, okPage && okPageSize
	})
	if err != nil {
		return nil, err
	}
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
func (s *UserChatService) GetUserChat(ctx context.Context, args ...any) (*UserChat, error) {
	params, err := compatParams("GetUserChat", args, func(args []any) (UserChatViewParams, bool) {
		if len(args) != 1 {
			return UserChatViewParams{}, false
		}
		chatCode, ok := compatString(args[0])
		return UserChatViewParams{ChatCode: chatCode}, ok
	})
	if err != nil {
		return nil, err
	}
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
func (s *UserChatService) CreateUserChat(ctx context.Context, args ...any) (*UserChat, error) {
	params, err := compatParams("CreateUserChat", args, func(args []any) (UserChatCreateParams, bool) {
		if len(args) != 1 {
			return UserChatCreateParams{}, false
		}
		targetID, ok := compatString(args[0])
		return UserChatCreateParams{TargetID: targetID}, ok
	})
	if err != nil {
		return nil, err
	}
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
func (s *UserChatService) DeleteUserChat(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteUserChat", args, func(args []any) (UserChatDeleteParams, bool) {
		if len(args) != 1 {
			return UserChatDeleteParams{}, false
		}
		chatCode, ok := compatString(args[0])
		return UserChatDeleteParams{ChatCode: chatCode}, ok
	})
	if err != nil {
		return err
	}
	if params.ChatCode == "" {
		return fmt.Errorf("私信会话Code不能为空")
	}

	_, err = s.client.Post(ctx, "user-chat/delete", map[string]interface{}{
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
