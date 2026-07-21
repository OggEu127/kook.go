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
func (s *EmojiService) GetEmojiList(ctx context.Context, args ...any) (*EmojiListResponse, error) {
	params, err := compatParams("GetEmojiList", args, func(args []any) (EmojiListParams, bool) {
		if len(args) != 3 {
			return EmojiListParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		page, ok2 := compatInt(args[1])
		pageSize, ok3 := compatInt(args[2])
		return EmojiListParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
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
func (s *EmojiService) CreateEmoji(ctx context.Context, args ...any) (*Emoji, error) {
	params, err := compatParams("CreateEmoji", args, func(args []any) (EmojiCreateParams, bool) {
		if len(args) != 3 {
			return EmojiCreateParams{}, false
		}
		name, ok1 := compatString(args[0])
		guildID, ok2 := compatString(args[1])
		content, ok3 := args[2].([]byte)
		return EmojiCreateParams{GuildID: guildID, Name: name, FileName: "emoji", Emoji: content}, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
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
func (s *EmojiService) UpdateEmoji(ctx context.Context, args ...any) error {
	params, err := compatParams("UpdateEmoji", args, func(args []any) (EmojiUpdateParams, bool) {
		if len(args) != 2 {
			return EmojiUpdateParams{}, false
		}
		id, ok1 := compatString(args[0])
		name, ok2 := compatString(args[1])
		return EmojiUpdateParams{ID: id, Name: name}, ok1 && ok2
	})
	if err != nil {
		return err
	}
	if params.ID == "" {
		return fmt.Errorf("表情ID不能为空")
	}
	if params.Name == "" {
		return fmt.Errorf("表情名称不能为空")
	}

	_, err = s.client.Post(ctx, "guild-emoji/update", map[string]interface{}{
		"id": params.ID, "name": params.Name,
	})
	return err
}

type EmojiDeleteParams struct {
	ID string
}

// DeleteEmoji 删除服务器表情。
func (s *EmojiService) DeleteEmoji(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteEmoji", args, func(args []any) (EmojiDeleteParams, bool) {
		if len(args) != 1 {
			return EmojiDeleteParams{}, false
		}
		id, ok := compatString(args[0])
		return EmojiDeleteParams{ID: id}, ok
	})
	if err != nil {
		return err
	}
	if params.ID == "" {
		return fmt.Errorf("表情ID不能为空")
	}

	body := map[string]interface{}{
		"id": params.ID,
	}

	_, err = s.client.Post(ctx, "guild-emoji/delete", body)
	return err
}

// UpdateEmojiLegacy 提供 v1 返回值形状；新代码应使用 UpdateEmoji。
func (s *EmojiService) UpdateEmojiLegacy(ctx context.Context, id, name string) (*Emoji, error) {
	if err := s.UpdateEmoji(ctx, id, name); err != nil {
		return nil, err
	}
	return &Emoji{ID: id, Name: name}, nil
}

// 数据结构定义

// Emoji 表情信息
type Emoji struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	UserID   string `json:"user_id"`
	UserInfo User   `json:"user_info"`
}

// EmojiListResponse 表情列表响应
type EmojiListResponse struct {
	Items []Emoji        `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}
