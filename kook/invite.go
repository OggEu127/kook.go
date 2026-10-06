package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// InviteService 邀请相关API服务
type InviteService struct {
	client *Client
}

// GetInvitees 获取被邀请用户列表及邀请留存统计。
func (s *InviteService) GetInvitees(ctx context.Context, params InviteeListParams) (*ListInviteesResponse, error) {
	if params.Page <= 0 || params.PageSize <= 0 {
		return nil, fmt.Errorf("页码和每页数量必须大于0")
	}
	resp, err := s.client.Get(ctx, "invite/invitees", params.toQuery())
	if err != nil {
		return nil, err
	}
	var result ListInviteesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析被邀请用户列表失败: %w", err)
	}
	return &result, nil
}

// GetInviteList 获取邀请列表。
func (s *InviteService) GetInviteList(ctx context.Context, args ...any) (*ListInvitesResponse, error) {
	params, err := compatParams("GetInviteList", args, func(args []any) (InviteListParams, bool) {
		if len(args) != 3 {
			return InviteListParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		page, okPage := compatInt(args[1])
		pageSize, okPageSize := compatInt(args[2])
		return InviteListParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, okGuild && okPage && okPageSize
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" && params.ChannelID == "" {
		return nil, fmt.Errorf("服务器ID或频道ID不能为空")
	}

	resp, err := s.client.Get(ctx, "invite/list", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result ListInvitesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析邀请列表失败: %w", err)
	}

	return &result, nil
}

// CreateInvite 创建邀请
func (s *InviteService) CreateInvite(ctx context.Context, params CreateInviteParams) (*Invite, error) {
	if params.GuildID == "" && params.ChannelID == "" {
		return nil, fmt.Errorf("服务器ID或频道ID不能为空")
	}

	requestParams := make(map[string]interface{})

	if params.GuildID != "" {
		requestParams["guild_id"] = params.GuildID
	}
	if params.ChannelID != "" {
		requestParams["channel_id"] = params.ChannelID
	}
	if params.Duration != nil {
		requestParams["duration"] = *params.Duration
	}
	if params.SettingTimes != nil {
		requestParams["setting_times"] = *params.SettingTimes
	} else if params.Setting != 0 {
		requestParams["setting_times"] = params.Setting
	}

	resp, err := s.client.Post(ctx, "invite/create", requestParams)
	if err != nil {
		return nil, err
	}

	var invite Invite
	if err := json.Unmarshal(resp.Data, &invite); err != nil {
		return nil, fmt.Errorf("解析邀请信息失败: %w", err)
	}

	return &invite, nil
}

// DeleteInvite 删除邀请
func (s *InviteService) DeleteInvite(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteInvite", args, func(args []any) (DeleteInviteParams, bool) {
		if len(args) != 1 {
			return DeleteInviteParams{}, false
		}
		urlCode, ok := compatString(args[0])
		return DeleteInviteParams{URLCode: urlCode}, ok
	})
	if err != nil {
		return err
	}
	if params.URLCode == "" {
		return fmt.Errorf("邀请码不能为空")
	}
	body := map[string]interface{}{"url_code": params.URLCode}
	if params.GuildID != "" {
		body["guild_id"] = params.GuildID
	}
	if params.ChannelID != "" {
		body["channel_id"] = params.ChannelID
	}
	_, err = s.client.Post(ctx, "invite/delete", body)
	return err
}

// 数据结构定义

// Invite 邀请信息
type Invite struct {
	GuildID     string `json:"guild_id"`     // 服务器ID
	ChannelID   string `json:"channel_id"`   // 频道ID
	URLCode     string `json:"url_code"`     // 邀请码
	URL         string `json:"url"`          // 邀请链接
	User        User   `json:"user"`         // 创建者信息
	CreatedAt   int64  `json:"created_at"`   // 创建时间
	UpdatedAt   int64  `json:"updated_at"`   // 更新时间
	ExpiredAt   int64  `json:"expired_at"`   // 过期时间
	Duration    int    `json:"duration"`     // 有效期（秒）
	Setting     int    `json:"setting"`      // 设置
	RemainTimes int    `json:"remain_times"` // 剩余使用次数
}

// InviteListParams 邀请列表参数
type InviteListParams struct {
	GuildID   string
	ChannelID string
	Page      *int
	PageSize  *int
}

func (p InviteListParams) toQuery() map[string]string {
	query := make(map[string]string)
	if p.GuildID != "" {
		query["guild_id"] = p.GuildID
	}
	if p.ChannelID != "" {
		query["channel_id"] = p.ChannelID
	}
	if p.Page != nil {
		query["page"] = strconv.Itoa(*p.Page)
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	return query
}

// InviteeListParams 被邀请用户列表参数。
type InviteeListParams struct {
	ID        string // 邀请码
	InviteURL string // 邀请链接
	GuildID   string // 服务器ID
	Status    *int   // 0未退出；254已退出；-1全部；nil使用服务端默认值-1
	StartTime string // 加入开始时间，例如2026-06-01 12:00:00
	EndTime   string // 加入结束时间，例如2026-06-02 12:00:00
	Page      int    // 必填，页码
	PageSize  int    // 必填，每页数量
}

func (p InviteeListParams) toQuery() map[string]string {
	query := map[string]string{
		"page": strconv.Itoa(p.Page), "page_size": strconv.Itoa(p.PageSize),
	}
	for key, value := range map[string]string{
		"id": p.ID, "invite_url": p.InviteURL, "guild_id": p.GuildID,
		"start_time": p.StartTime, "end_time": p.EndTime,
	} {
		if value != "" {
			query[key] = value
		}
	}
	if p.Status != nil {
		query["status"] = strconv.Itoa(*p.Status)
	}
	return query
}

// Invitee 被邀请用户信息。
type Invitee struct {
	Status     int    `json:"status"`
	JoinedTime int64  `json:"joined_time"` // 毫秒时间戳
	ActiveTime int64  `json:"active_time"` // 毫秒时间戳
	ShowName   string `json:"show_name"`
}

// ListInviteesResponse 被邀请用户列表及留存统计。
type ListInviteesResponse struct {
	Items     []Invitee      `json:"items"`
	Meta      PaginationMeta `json:"meta"`
	Sort      SortFields     `json:"sort"`
	Count     int            `json:"count"`
	KeepCount int            `json:"keep_count"`
	LossCount int            `json:"loss_count"`
}

// CreateInviteParams 创建邀请参数
type CreateInviteParams struct {
	GuildID      string `json:"guild_id,omitempty"`      // 服务器ID
	ChannelID    string `json:"channel_id,omitempty"`    // 频道ID
	Duration     *int   `json:"duration,omitempty"`      // 0永久；nil使用服务端默认值
	SettingTimes *int   `json:"setting_times,omitempty"` // -1无限制；nil使用服务端默认值
	Setting      int    `json:"setting,omitempty"`       // v1.1.1兼容字段；发送时映射为setting_times
}

// DeleteInviteParams 删除邀请参数。
type DeleteInviteParams struct {
	URLCode   string
	GuildID   string
	ChannelID string
}

// ListInvitesResponse 邀请列表响应
type ListInvitesResponse struct {
	Items []Invite       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// 邀请有效期常量
const (
	InviteDurationForever     = 0      // 永久
	InviteDurationHalfHour    = 1800   // 半小时
	InviteDurationOneHour     = 3600   // 一小时
	InviteDurationSixHours    = 21600  // 六小时
	InviteDurationTwelveHours = 43200  // 十二小时
	InviteDurationOneDay      = 86400  // 一天
	InviteDurationOneWeek     = 604800 // 七天
)
