package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// GuildService 服务器相关API服务
type GuildService struct {
	client *Client
}

// GetGuildMember 是 v1.1.1 兼容入口。当前官方契约未确认 guild/user。
func (s *GuildService) GetGuildMember(context.Context, string, string) (*GuildMember, error) {
	return nil, unsupportedEndpoint("guild/user")
}

// GetRegions 是 v1.1.1 兼容入口。当前官方契约未确认 guild/regions。
func (s *GuildService) GetRegions(context.Context) (*ListRegionsResponse, error) {
	return nil, unsupportedEndpoint("guild/regions")
}

// UpdateNickname 复用当前官方 guild/nickname 端点。
func (s *GuildService) UpdateNickname(ctx context.Context, guildID, userID, nickname string) error {
	return s.UpdateGuildMemberNickname(ctx, guildID, userID, nickname)
}

// GetGuildBoostInfo 是 v1.1.1 兼容入口。当前官方契约未确认 guild-boost/info。
func (s *GuildService) GetGuildBoostInfo(context.Context, string) (*GuildBoostInfo, error) {
	return nil, unsupportedEndpoint("guild-boost/info")
}

// GetGuildList 获取当前用户的服务器列表
func (s *GuildService) GetGuildList(ctx context.Context, args ...any) (*ListGuildsResponse, error) {
	params, err := compatParams("GetGuildList", args, func(args []any) (GuildListParams, bool) {
		if len(args) != 3 {
			return GuildListParams{}, false
		}
		page, ok1 := compatInt(args[0])
		pageSize, ok2 := compatInt(args[1])
		sortValue, ok3 := compatString(args[2])
		return GuildListParams{Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize), Sort: sortValue}, ok1 && ok2 && ok3
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
	if params.Sort != "" {
		query["sort"] = params.Sort
	}

	resp, err := s.client.Get(ctx, "guild/list", query)
	if err != nil {
		return nil, err
	}

	var result ListGuildsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析服务器列表失败: %w", err)
	}

	return &result, nil
}

// GetGuildInfo 获取服务器信息
func (s *GuildService) GetGuildInfo(ctx context.Context, args ...any) (*Guild, error) {
	params, err := compatParams("GetGuildInfo", args, func(args []any) (GuildViewParams, bool) {
		if len(args) != 1 {
			return GuildViewParams{}, false
		}
		value, ok := compatString(args[0])
		return GuildViewParams{GuildID: value}, ok
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

	resp, err := s.client.Get(ctx, "guild/view", query)
	if err != nil {
		return nil, err
	}

	var guild Guild
	if err := json.Unmarshal(resp.Data, &guild); err != nil {
		return nil, fmt.Errorf("解析服务器信息失败: %w", err)
	}

	return &guild, nil
}

// LeaveGuild 离开服务器
func (s *GuildService) LeaveGuild(ctx context.Context, args ...any) error {
	params, err := compatParams("LeaveGuild", args, func(args []any) (GuildLeaveParams, bool) {
		if len(args) != 1 {
			return GuildLeaveParams{}, false
		}
		value, ok := compatString(args[0])
		return GuildLeaveParams{GuildID: value}, ok
	})
	if err != nil {
		return err
	}
	if params.GuildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id": params.GuildID,
	}

	_, err = s.client.Post(ctx, "guild/leave", body)
	return err
}

// GetGuildMembers 获取服务器成员列表
func (s *GuildService) GetGuildMembers(ctx context.Context, args ...any) (*ListGuildMembersResponse, error) {
	params, err := compatParams("GetGuildMembers", args, func(args []any) (GuildMembersParams, bool) {
		if len(args) != 4 {
			return GuildMembersParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		page, ok2 := compatInt(args[1])
		pageSize, ok3 := compatInt(args[2])
		_, ok4 := compatString(args[3])
		return GuildMembersParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, ok1 && ok2 && ok3 && ok4
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	resp, err := s.client.Get(ctx, "guild/user-list", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result ListGuildMembersResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析服务器成员列表失败: %w", err)
	}

	return &result, nil
}

// KickGuildMember 踢出服务器成员
func (s *GuildService) KickGuildMember(ctx context.Context, args ...any) error {
	params, err := compatParams("KickGuildMember", args, func(args []any) (GuildKickoutParams, bool) {
		if len(args) != 2 {
			return GuildKickoutParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		return GuildKickoutParams{GuildID: guildID, TargetID: userID}, ok1 && ok2
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

	_, err = s.client.Post(ctx, "guild/kickout", body)
	return err
}

// UpdateGuildMemberNickname 修改服务器成员昵称
func (s *GuildService) UpdateGuildMemberNickname(ctx context.Context, args ...any) error {
	params, err := compatParams("UpdateGuildMemberNickname", args, func(args []any) (GuildNicknameParams, bool) {
		if len(args) != 3 {
			return GuildNicknameParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		nickname, ok3 := compatString(args[2])
		return GuildNicknameParams{GuildID: guildID, UserID: userID, Nickname: stringPointer(nickname)}, ok1 && ok2 && ok3
	})
	if err != nil {
		return err
	}
	if params.GuildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}

	requestParams := map[string]interface{}{
		"guild_id": params.GuildID,
	}
	if params.Nickname != nil {
		requestParams["nickname"] = *params.Nickname
	}
	if params.UserID != "" {
		requestParams["user_id"] = params.UserID
	}

	_, err = s.client.Post(ctx, "guild/nickname", requestParams)
	return err
}

// GetGuildMuteList 获取服务器静音闭麦列表
func (s *GuildService) GetGuildMuteList(ctx context.Context, args ...any) (*GuildMuteList, error) {
	params, err := compatParams("GetGuildMuteList", args, func(args []any) (GuildMuteListParams, bool) {
		if len(args) != 2 {
			return GuildMuteListParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		returnType, ok2 := compatString(args[1])
		return GuildMuteListParams{GuildID: guildID, ReturnType: returnType}, ok1 && ok2
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{"guild_id": params.GuildID}
	if params.ReturnType != "" {
		query["return_type"] = params.ReturnType
	}

	resp, err := s.client.Get(ctx, "guild-mute/list", query)
	if err != nil {
		return nil, err
	}

	var result GuildMuteList
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析服务器静音闭麦列表失败: %w", err)
	}

	return &result, nil
}

// CreateGuildMute 添加服务器静音或闭麦
func (s *GuildService) CreateGuildMute(ctx context.Context, args ...any) error {
	params, err := compatParams("CreateGuildMute", args, func(args []any) (GuildMuteCreateParams, bool) {
		if len(args) != 3 {
			return GuildMuteCreateParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		muteType, ok3 := compatInt(args[2])
		return GuildMuteCreateParams{GuildID: guildID, UserID: userID, Type: muteType}, ok1 && ok2 && ok3
	})
	if err != nil {
		return err
	}
	return s.setGuildMute(ctx, "guild-mute/create", params.GuildID, params.UserID, params.Type)
}

// DeleteGuildMute 删除服务器静音或闭麦
func (s *GuildService) DeleteGuildMute(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteGuildMute", args, func(args []any) (GuildMuteDeleteParams, bool) {
		if len(args) != 3 {
			return GuildMuteDeleteParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		muteType, ok3 := compatInt(args[2])
		return GuildMuteDeleteParams{GuildID: guildID, UserID: userID, Type: muteType}, ok1 && ok2 && ok3
	})
	if err != nil {
		return err
	}
	return s.setGuildMute(ctx, "guild-mute/delete", params.GuildID, params.UserID, params.Type)
}

func (s *GuildService) setGuildMute(ctx context.Context, endpoint, guildID, userID string, muteType int) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}
	if muteType != GuildMuteTypeMic && muteType != GuildMuteTypeHeadset {
		return fmt.Errorf("静音类型必须为1或2")
	}

	_, err := s.client.Post(ctx, endpoint, map[string]interface{}{
		"guild_id": guildID,
		"user_id":  userID,
		"type":     muteType,
	})
	return err
}

// GetGuildBoostHistory 查询服务器助力历史
func (s *GuildService) GetGuildBoostHistory(ctx context.Context, args ...any) (*GuildBoostHistoryResponse, error) {
	params, err := compatParams("GetGuildBoostHistory", args, func(args []any) (GuildBoostHistoryParams, bool) {
		if len(args) != 3 {
			return GuildBoostHistoryParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		start, ok2 := args[1].(int64)
		end, ok3 := args[2].(int64)
		return GuildBoostHistoryParams{GuildID: guildID, StartTime: optionalPositiveInt64(start), EndTime: optionalPositiveInt64(end)}, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{"guild_id": params.GuildID}
	if params.StartTime != nil {
		query["start_time"] = strconv.FormatInt(*params.StartTime, 10)
	}
	if params.EndTime != nil {
		query["end_time"] = strconv.FormatInt(*params.EndTime, 10)
	}

	resp, err := s.client.Get(ctx, "guild-boost/history", query)
	if err != nil {
		return nil, err
	}

	var result GuildBoostHistoryResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析服务器助力历史失败: %w", err)
	}

	return &result, nil
}

// 数据结构定义

type GuildListParams struct {
	Page     *int
	PageSize *int
	Sort     string
}

type GuildViewParams struct {
	GuildID string
}

type GuildLeaveParams struct {
	GuildID string
}

type GuildKickoutParams struct {
	GuildID  string
	TargetID string
}

type GuildNicknameParams struct {
	GuildID  string
	Nickname *string
	UserID   string
}

type GuildMuteListParams struct {
	GuildID    string
	ReturnType string
}

type GuildMuteCreateParams struct {
	GuildID string
	UserID  string
	Type    int
}

type GuildMuteDeleteParams struct {
	GuildID string
	UserID  string
	Type    int
}

type GuildBoostHistoryParams struct {
	GuildID   string
	StartTime *int64
	EndTime   *int64
}

// ListGuildsResponse 服务器列表响应
type ListGuildsResponse struct {
	Items []Guild        `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// GuildMembersParams 服务器成员列表参数
type GuildMembersParams struct {
	GuildID        string `json:"guild_id,omitempty"`
	ChannelID      string `json:"channel_id,omitempty"`
	Search         string `json:"search,omitempty"`
	RoleID         *int   `json:"role_id,omitempty"`
	MobileVerified *int   `json:"mobile_verified,omitempty"`
	ActiveTime     *int   `json:"active_time,omitempty"`
	JoinedAt       *int   `json:"joined_at,omitempty"`
	Page           *int   `json:"page,omitempty"`
	PageSize       *int   `json:"page_size,omitempty"`
	FilterUserID   string `json:"filter_user_id,omitempty"`
}

func (p GuildMembersParams) toQuery() map[string]string {
	query := map[string]string{
		"guild_id": p.GuildID,
	}
	if p.ChannelID != "" {
		query["channel_id"] = p.ChannelID
	}
	if p.Search != "" {
		query["search"] = p.Search
	}
	if p.RoleID != nil {
		query["role_id"] = strconv.Itoa(*p.RoleID)
	}
	if p.MobileVerified != nil {
		query["mobile_verified"] = strconv.Itoa(*p.MobileVerified)
	}
	if p.ActiveTime != nil {
		query["active_time"] = strconv.Itoa(*p.ActiveTime)
	}
	if p.JoinedAt != nil {
		query["joined_at"] = strconv.Itoa(*p.JoinedAt)
	}
	if p.Page != nil {
		query["page"] = strconv.Itoa(*p.Page)
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	if p.FilterUserID != "" {
		query["filter_user_id"] = p.FilterUserID
	}
	return query
}

// ListGuildMembersResponse 服务器成员列表响应
type ListGuildMembersResponse struct {
	Items        []User         `json:"items"`
	Meta         PaginationMeta `json:"meta"`
	Sort         SortFields     `json:"sort"`
	UserCount    int            `json:"user_count"`
	OnlineCount  int            `json:"online_count"`
	OfflineCount int            `json:"offline_count"`
}

// GuildMember 保留 v1.1.1 的成员数据类型。
type GuildMember struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Nickname    string `json:"nickname"`
	IdentifyNum string `json:"identify_num"`
	Online      bool   `json:"online"`
	Bot         bool   `json:"bot"`
	Status      int    `json:"status"`
	Avatar      string `json:"avatar"`
	VipAvatar   string `json:"vip_avatar"`
	Roles       []int  `json:"roles"`
	JoinedAt    int64  `json:"joined_at"`
	ActiveTime  int64  `json:"active_time"`
	IsVip       bool   `json:"is_vip"`
	VipAmp      bool   `json:"vip_amp"`
}

// ListRegionsResponse 保留 v1.1.1 的区域列表响应。
type ListRegionsResponse struct {
	Items []Region       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// GuildBoostInfo 保留 v1.1.1 的助力概要类型。
type GuildBoostInfo struct {
	BoostNum       int `json:"boost_num"`
	BufferBoostNum int `json:"buffer_boost_num"`
	Level          int `json:"level"`
}

const (
	// GuildMuteTypeMic 麦克风闭麦
	GuildMuteTypeMic = 1
	// GuildMuteTypeHeadset 耳机静音
	GuildMuteTypeHeadset = 2
)

// GuildMuteList 服务器静音闭麦列表
type GuildMuteList struct {
	Mic     GuildMuteState `json:"mic"`
	Headset GuildMuteState `json:"headset"`
}

// GuildMuteState 静音闭麦分组
type GuildMuteState struct {
	Type    int      `json:"type"`
	UserIDs []string `json:"user_ids"`
}

// GuildBoostHistory 助力历史
type GuildBoostHistory struct {
	UserID    string `json:"user_id"`
	GuildID   string `json:"guild_id"`
	StartTime int64  `json:"start_time"`
	EndTime   int64  `json:"end_time"`
	User      User   `json:"user"`
}

// GuildBoostHistoryResponse 助力历史分页响应
type GuildBoostHistoryResponse struct {
	Items []GuildBoostHistory `json:"items"`
	Meta  PaginationMeta      `json:"meta"`
	Sort  SortFields          `json:"sort"`
}
