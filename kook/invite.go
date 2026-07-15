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

// GetInviteList 获取邀请列表
func (s *InviteService) GetInviteList(ctx context.Context, guildID string, page, pageSize int) (*ListInvitesResponse, error) {
	return s.GetInviteListWithParams(ctx, InviteListParams{
		GuildID:  guildID,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetInviteListWithParams 获取邀请列表
func (s *InviteService) GetInviteListWithParams(ctx context.Context, params InviteListParams) (*ListInvitesResponse, error) {
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
	if params.Duration > 0 {
		requestParams["duration"] = params.Duration
	}
	settingTimes := params.SettingTimes
	if settingTimes == 0 {
		settingTimes = params.Setting
	}
	if settingTimes != 0 {
		requestParams["setting_times"] = settingTimes
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
func (s *InviteService) DeleteInvite(ctx context.Context, urlCode string) error {
	if urlCode == "" {
		return fmt.Errorf("邀请码不能为空")
	}

	params := map[string]interface{}{
		"url_code": urlCode,
	}

	_, err := s.client.Post(ctx, "invite/delete", params)
	return err
}

// GetInvitees 获取邀请用户列表
func (s *InviteService) GetInvitees(ctx context.Context, params InviteesParams) (*InviteesResponse, error) {
	if params.Page <= 0 {
		return nil, fmt.Errorf("分页页码不能为空")
	}
	if params.PageSize <= 0 {
		return nil, fmt.Errorf("分页每页数量不能为空")
	}

	resp, err := s.client.Get(ctx, "invite/invitees", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result InviteesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析邀请用户列表失败: %w", err)
	}

	return &result, nil
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
	GuildID   string `json:"guild_id,omitempty"`
	ChannelID string `json:"channel_id,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

func (p InviteListParams) toQuery() map[string]string {
	query := make(map[string]string)
	if p.GuildID != "" {
		query["guild_id"] = p.GuildID
	}
	if p.ChannelID != "" {
		query["channel_id"] = p.ChannelID
	}
	if p.Page > 0 {
		query["page"] = strconv.Itoa(p.Page)
	}
	if p.PageSize > 0 && p.PageSize <= 50 {
		query["page_size"] = strconv.Itoa(p.PageSize)
	}
	return query
}

// CreateInviteParams 创建邀请参数
type CreateInviteParams struct {
	GuildID      string `json:"guild_id,omitempty"`      // 服务器ID
	ChannelID    string `json:"channel_id,omitempty"`    // 频道ID
	Duration     int    `json:"duration,omitempty"`      // 有效期（秒）：0永久，1800半小时，3600一小时，21600六小时，43200十二小时，86400一天，604800七天
	SettingTimes int    `json:"setting_times,omitempty"` // 使用次数限制，默认-1无限制
	Setting      int    `json:"setting,omitempty"`       // Deprecated: 请使用 SettingTimes。
}

// ListInvitesResponse 邀请列表响应
type ListInvitesResponse struct {
	Items []Invite       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// InviteesParams 邀请用户列表参数
type InviteesParams struct {
	ID        string `json:"id,omitempty"`
	InviteURL string `json:"invite_url,omitempty"`
	GuildID   string `json:"guild_id,omitempty"`
	Status    *int   `json:"status,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	EndTime   string `json:"end_time,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

func (p InviteesParams) toQuery() map[string]string {
	query := make(map[string]string)
	if p.ID != "" {
		query["id"] = p.ID
	}
	if p.InviteURL != "" {
		query["invite_url"] = p.InviteURL
	}
	if p.GuildID != "" {
		query["guild_id"] = p.GuildID
	}
	if p.Status != nil {
		query["status"] = strconv.Itoa(*p.Status)
	}
	if p.StartTime != "" {
		query["start_time"] = p.StartTime
	}
	if p.EndTime != "" {
		query["end_time"] = p.EndTime
	}
	if p.Page > 0 {
		query["page"] = strconv.Itoa(p.Page)
	}
	if p.PageSize > 0 {
		query["page_size"] = strconv.Itoa(p.PageSize)
	}
	return query
}

// Invitee 邀请加入用户信息
type Invitee struct {
	Status     int    `json:"status"`
	JoinedTime int64  `json:"joined_time"`
	ActiveTime int64  `json:"active_time"`
	ShowName   string `json:"show_name"`
}

// InviteesResponse 邀请用户列表响应
type InviteesResponse struct {
	Items     []Invitee      `json:"items"`
	Meta      PaginationMeta `json:"meta"`
	Sort      map[string]int `json:"sort"`
	Count     int            `json:"count"`
	KeepCount int            `json:"keep_count"`
	LossCount int            `json:"loss_count"`
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
