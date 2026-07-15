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

// GetGuildList 获取当前用户的服务器列表
func (s *GuildService) GetGuildList(ctx context.Context, page, pageSize int, sort string) (*ListGuildsResponse, error) {
	query := make(map[string]string)

	if page > 0 {
		query["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 && pageSize <= 50 {
		query["page_size"] = strconv.Itoa(pageSize)
	}
	if sort != "" {
		query["sort"] = sort
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
func (s *GuildService) GetGuildInfo(ctx context.Context, guildID string) (*Guild, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{
		"guild_id": guildID,
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
func (s *GuildService) LeaveGuild(ctx context.Context, guildID string) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}

	params := map[string]interface{}{
		"guild_id": guildID,
	}

	_, err := s.client.Post(ctx, "guild/leave", params)
	return err
}

// GetGuildMembers 获取服务器成员列表
func (s *GuildService) GetGuildMembers(ctx context.Context, guildID string, page, pageSize int, sort string) (*ListGuildMembersResponse, error) {
	return s.GetGuildMembersWithParams(ctx, GuildMembersParams{
		GuildID:  guildID,
		Page:     page,
		PageSize: pageSize,
	})
}

// GetGuildMembersWithParams 获取服务器成员列表
func (s *GuildService) GetGuildMembersWithParams(ctx context.Context, params GuildMembersParams) (*ListGuildMembersResponse, error) {
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

// GetGuildMember 获取服务器成员信息
func (s *GuildService) GetGuildMember(ctx context.Context, guildID, userID string) (*GuildMember, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	query := map[string]string{
		"guild_id": guildID,
		"user_id":  userID,
	}

	resp, err := s.client.Get(ctx, "user/view", query)
	if err != nil {
		return nil, err
	}

	var member GuildMember
	if err := json.Unmarshal(resp.Data, &member); err != nil {
		return nil, fmt.Errorf("解析服务器成员信息失败: %w", err)
	}

	return &member, nil
}

// KickGuildMember 踢出服务器成员
func (s *GuildService) KickGuildMember(ctx context.Context, guildID, userID string) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	params := map[string]interface{}{
		"guild_id":  guildID,
		"target_id": userID,
	}

	_, err := s.client.Post(ctx, "guild/kickout", params)
	return err
}

// UpdateGuildMemberNickname 修改服务器成员昵称
func (s *GuildService) UpdateGuildMemberNickname(ctx context.Context, guildID, userID, nickname string) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}

	params := map[string]interface{}{
		"guild_id": guildID,
		"nickname": nickname,
	}

	if userID != "" {
		params["user_id"] = userID
	}

	_, err := s.client.Post(ctx, "guild/nickname", params)
	return err
}

// GetRegions 获取可用的服务器区域列表
func (s *GuildService) GetRegions(ctx context.Context) (*ListRegionsResponse, error) {
	resp, err := s.client.Get(ctx, "guild/regions", nil)
	if err != nil {
		return nil, err
	}

	var result ListRegionsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析区域列表失败: %w", err)
	}

	return &result, nil
}

// UpdateNickname 修改用户昵称
func (s *GuildService) UpdateNickname(ctx context.Context, guildID, userID, nickname string) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	params := map[string]interface{}{
		"guild_id": guildID,
		"user_id":  userID,
	}

	if nickname != "" {
		params["nickname"] = nickname
	}

	_, err := s.client.Post(ctx, "guild/nickname", params)
	return err
}

// GetGuildBoostInfo 获取服务器助力信息
func (s *GuildService) GetGuildBoostInfo(ctx context.Context, guildID string) (*GuildBoostInfo, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{
		"guild_id": guildID,
	}

	resp, err := s.client.Get(ctx, "guild-boost/info", query)
	if err != nil {
		return nil, err
	}

	var boostInfo GuildBoostInfo
	if err := json.Unmarshal(resp.Data, &boostInfo); err != nil {
		return nil, fmt.Errorf("解析助力信息失败: %w", err)
	}

	return &boostInfo, nil
}

// GetGuildMuteList 获取服务器静音闭麦列表
func (s *GuildService) GetGuildMuteList(ctx context.Context, guildID string, returnType string) (*GuildMuteList, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{"guild_id": guildID}
	if returnType != "" {
		query["return_type"] = returnType
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
func (s *GuildService) CreateGuildMute(ctx context.Context, guildID, userID string, muteType int) error {
	return s.setGuildMute(ctx, "guild-mute/create", guildID, userID, muteType)
}

// DeleteGuildMute 删除服务器静音或闭麦
func (s *GuildService) DeleteGuildMute(ctx context.Context, guildID, userID string, muteType int) error {
	return s.setGuildMute(ctx, "guild-mute/delete", guildID, userID, muteType)
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
func (s *GuildService) GetGuildBoostHistory(ctx context.Context, guildID string, startTime, endTime int64) (*GuildBoostHistoryResponse, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{"guild_id": guildID}
	if startTime > 0 {
		query["start_time"] = strconv.FormatInt(startTime, 10)
	}
	if endTime > 0 {
		query["end_time"] = strconv.FormatInt(endTime, 10)
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

// ListGuildsResponse 服务器列表响应
type ListGuildsResponse struct {
	Items []Guild        `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// GuildMembersParams 服务器成员列表参数
type GuildMembersParams struct {
	GuildID        string `json:"guild_id,omitempty"`
	ChannelID      string `json:"channel_id,omitempty"`
	Search         string `json:"search,omitempty"`
	RoleID         int    `json:"role_id,omitempty"`
	MobileVerified *int   `json:"mobile_verified,omitempty"`
	ActiveTime     *int   `json:"active_time,omitempty"`
	JoinedAt       *int   `json:"joined_at,omitempty"`
	Page           int    `json:"page,omitempty"`
	PageSize       int    `json:"page_size,omitempty"`
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
	if p.RoleID > 0 {
		query["role_id"] = strconv.Itoa(p.RoleID)
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
	if p.Page > 0 {
		query["page"] = strconv.Itoa(p.Page)
	}
	if p.PageSize > 0 && p.PageSize <= 50 {
		query["page_size"] = strconv.Itoa(p.PageSize)
	}
	if p.FilterUserID != "" {
		query["filter_user_id"] = p.FilterUserID
	}
	return query
}

// ListGuildMembersResponse 服务器成员列表响应
type ListGuildMembersResponse struct {
	Items []GuildMember  `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// ListRegionsResponse 区域列表响应
type ListRegionsResponse struct {
	Items []Region       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// GuildBoostInfo 服务器助力信息
type GuildBoostInfo struct {
	BoostNum       int `json:"boost_num"`        // 助力数量
	BufferBoostNum int `json:"buffer_boost_num"` // 缓冲助力数量
	Level          int `json:"level"`            // 服务器等级
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
	Sort  map[string]int      `json:"sort"`
}
