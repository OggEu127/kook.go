package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// EmojiService 表情包相关API服务
type EmojiService struct {
	client *Client
}

type EmojiListParams struct {
	GuildID  string
	Page     *int
	PageSize *int
}

// GetEmojiList 获取服务器表情列表。
func (s *EmojiService) GetEmojiList(ctx context.Context, params EmojiListParams) (*EmojiListResponse, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{
		"guild_id": params.GuildID,
	}

	if params.Page != nil {
		query["page"] = strconv.Itoa(*params.Page)
	}
	if params.PageSize != nil {
		query["page_size"] = strconv.Itoa(*params.PageSize)
	}

	resp, err := s.client.Get(ctx, "guild-emoji/list", query)
	if err != nil {
		return nil, err
	}

	var result EmojiListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析表情列表失败: %w", err)
	}

	return &result, nil
}

type EmojiCreateParams struct {
	GuildID  string
	Name     string
	FileName string
	Emoji    []byte
}

// CreateEmoji 创建服务器表情。
func (s *EmojiService) CreateEmoji(ctx context.Context, params EmojiCreateParams) (*Emoji, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.FileName == "" || len(params.Emoji) == 0 {
		return nil, fmt.Errorf("表情文件名和内容不能为空")
	}

	fields := map[string]string{"guild_id": params.GuildID}
	if params.Name != "" {
		fields["name"] = params.Name
	}
	resp, err := s.client.doMultipartRequest(ctx, "guild-emoji/create", fields, map[string]multipartFile{
		"emoji": {FileName: params.FileName, Content: params.Emoji},
	}, nil)
	if err != nil {
		return nil, err
	}

	var result Emoji
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析表情信息失败: %w", err)
	}

	return &result, nil
}

type EmojiUpdateParams struct {
	ID   string
	Name string
}

// UpdateEmoji 更新服务器表情。
func (s *EmojiService) UpdateEmoji(ctx context.Context, params EmojiUpdateParams) error {
	if params.ID == "" {
		return fmt.Errorf("表情ID不能为空")
	}
	if params.Name == "" {
		return fmt.Errorf("表情名称不能为空")
	}

	_, err := s.client.Post(ctx, "guild-emoji/update", map[string]interface{}{
		"id": params.ID, "name": params.Name,
	})
	return err
}

type EmojiDeleteParams struct {
	ID string
}

// DeleteEmoji 删除服务器表情。
func (s *EmojiService) DeleteEmoji(ctx context.Context, params EmojiDeleteParams) error {
	if params.ID == "" {
		return fmt.Errorf("表情ID不能为空")
	}

	body := map[string]interface{}{
		"id": params.ID,
	}

	_, err := s.client.Post(ctx, "guild-emoji/delete", body)
	return err
}

// 数据结构定义

// Emoji 表情信息
type Emoji struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	UserInfo User   `json:"user_info"`
}

// EmojiListResponse 表情列表响应
type EmojiListResponse struct {
	Items []Emoji        `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}
