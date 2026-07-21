package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// BlacklistService 屏蔽/黑名单相关API服务
type BlacklistService struct {
	client *Client
}

type BlacklistListParams struct {
	GuildID  string
	Page     *int
	PageSize *int
}

// GetBlacklistUsers 获取屏蔽用户列表。
func (s *BlacklistService) GetBlacklistUsers(ctx context.Context, args ...any) (*BlacklistResponse, error) {
	params, err := compatParams("GetBlacklistUsers", args, func(args []any) (BlacklistListParams, bool) {
		if len(args) != 3 {
			return BlacklistListParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		page, okPage := compatInt(args[1])
		pageSize, okPageSize := compatInt(args[2])
		return BlacklistListParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, okGuild && okPage && okPageSize
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

	resp, err := s.client.Get(ctx, "blacklist/list", query)
	if err != nil {
		return nil, err
	}

	var result BlacklistResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析屏蔽用户列表失败: %w", err)
	}

	return &result, nil
}

type BlacklistCreateParams struct {
	GuildID    string
	TargetID   string
	Remark     string
	DelMsgDays *int
}

// CreateBlacklistUser 屏蔽用户。
func (s *BlacklistService) CreateBlacklistUser(ctx context.Context, args ...any) error {
	params, err := compatParams("CreateBlacklistUser", args, func(args []any) (BlacklistCreateParams, bool) {
		if len(args) != 4 {
			return BlacklistCreateParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		userID, okUser := compatString(args[1])
		remark, okRemark := compatString(args[2])
		days, okDays := compatInt(args[3])
		return BlacklistCreateParams{GuildID: guildID, TargetID: userID, Remark: remark, DelMsgDays: &days}, okGuild && okUser && okRemark && okDays
	})
	if err != nil {
		return err
	}
	if params.GuildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if params.TargetID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id":  params.GuildID,
		"target_id": params.TargetID,
	}

	if params.Remark != "" {
		body["remark"] = params.Remark
	}
	if params.DelMsgDays != nil {
		body["del_msg_days"] = *params.DelMsgDays
	}

	_, err = s.client.Post(ctx, "blacklist/create", body)
	return err
}

type BlacklistDeleteParams struct {
	GuildID  string
	TargetID string
}

// DeleteBlacklistUser 取消屏蔽用户。
func (s *BlacklistService) DeleteBlacklistUser(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteBlacklistUser", args, func(args []any) (BlacklistDeleteParams, bool) {
		if len(args) != 2 {
			return BlacklistDeleteParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		userID, okUser := compatString(args[1])
		return BlacklistDeleteParams{GuildID: guildID, TargetID: userID}, okGuild && okUser
	})
	if err != nil {
		return err
	}
	if params.GuildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if params.TargetID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id":  params.GuildID,
		"target_id": params.TargetID,
	}

	_, err = s.client.Post(ctx, "blacklist/delete", body)
	return err
}

// 数据结构定义

// BlacklistUser 屏蔽用户信息
type BlacklistUser struct {
	User        User   `json:"user"`
	Remark      string `json:"remark"`
	UserID      string `json:"user_id"`
	CreatedTime int64  `json:"created_time"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

// BlacklistResponse 屏蔽用户列表响应
type BlacklistResponse struct {
	Items []BlacklistUser `json:"items"`
	Meta  PaginationMeta  `json:"meta"`
	Sort  SortFields      `json:"sort"`
}
