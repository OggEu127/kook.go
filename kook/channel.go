package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ChannelService 频道相关API服务
type ChannelService struct {
	client *Client
}

// GetChannelList 获取频道列表
func (s *ChannelService) GetChannelList(ctx context.Context, guildID string, page, pageSize int, sort string) (*ListChannelsResponse, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	query := map[string]string{
		"guild_id": guildID,
	}

	if page > 0 {
		query["page"] = strconv.Itoa(page)
	}
	if pageSize > 0 && pageSize <= 50 {
		query["page_size"] = strconv.Itoa(pageSize)
	}
	if sort != "" {
		query["sort"] = sort
	}

	resp, err := s.client.Get(ctx, "channel/list", query)
	if err != nil {
		return nil, err
	}

	var result ListChannelsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道列表失败: %w", err)
	}

	return &result, nil
}

// GetChannelInfo 获取频道信息
func (s *ChannelService) GetChannelInfo(ctx context.Context, channelID string) (*Channel, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"target_id": channelID,
	}

	resp, err := s.client.Get(ctx, "channel/view", query)
	if err != nil {
		return nil, err
	}

	var channel Channel
	if err := json.Unmarshal(resp.Data, &channel); err != nil {
		return nil, fmt.Errorf("解析频道信息失败: %w", err)
	}

	return &channel, nil
}

// CreateChannel 创建频道
func (s *ChannelService) CreateChannel(ctx context.Context, guildID string, params CreateChannelParams) (*Channel, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("频道名称不能为空")
	}

	requestParams := map[string]interface{}{
		"guild_id": guildID,
		"name":     params.Name,
	}

	if params.Type > 0 {
		requestParams["type"] = params.Type
	} else {
		requestParams["type"] = 1 // 默认为文字频道
	}

	if params.ParentID != "" {
		requestParams["parent_id"] = params.ParentID
	}
	if params.LimitAmount > 0 {
		requestParams["limit_amount"] = params.LimitAmount
	}
	if params.VoiceQuality > 0 {
		requestParams["voice_quality"] = params.VoiceQuality
	}
	if params.IsCategory {
		requestParams["is_category"] = 1
	}

	resp, err := s.client.Post(ctx, "channel/create", requestParams)
	if err != nil {
		return nil, err
	}

	var channel Channel
	if err := json.Unmarshal(resp.Data, &channel); err != nil {
		return nil, fmt.Errorf("解析频道信息失败: %w", err)
	}

	return &channel, nil
}

// UpdateChannel 更新频道信息
func (s *ChannelService) UpdateChannel(ctx context.Context, channelID string, params UpdateChannelParams) (*Channel, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := map[string]interface{}{
		"channel_id": channelID,
	}

	if params.Name != "" {
		requestParams["name"] = params.Name
	}
	if params.Topic != "" {
		requestParams["topic"] = params.Topic
	}
	if params.SlowMode >= 0 {
		requestParams["slow_mode"] = params.SlowMode
	}
	if params.LimitAmount > 0 {
		requestParams["limit_amount"] = params.LimitAmount
	}
	if params.VoiceQuality > 0 {
		requestParams["voice_quality"] = params.VoiceQuality
	}
	if params.Password != "" {
		requestParams["password"] = params.Password
	}

	resp, err := s.client.Post(ctx, "channel/update", requestParams)
	if err != nil {
		return nil, err
	}

	var channel Channel
	if err := json.Unmarshal(resp.Data, &channel); err != nil {
		return nil, fmt.Errorf("解析频道信息失败: %w", err)
	}

	return &channel, nil
}

// DeleteChannel 删除频道
func (s *ChannelService) DeleteChannel(ctx context.Context, channelID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}

	_, err := s.client.Post(ctx, "channel/delete", params)
	return err
}

// MoveChannel 移动频道位置
func (s *ChannelService) MoveChannel(ctx context.Context, guildID string, channelIDs []string) error {
	if guildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if len(channelIDs) == 0 {
		return fmt.Errorf("频道ID列表不能为空")
	}

	params := map[string]interface{}{
		"guild_id":    guildID,
		"channel_ids": channelIDs,
	}

	_, err := s.client.Post(ctx, "channel/move", params)
	return err
}

// KickoutFromVoiceChannel 从语音频道踢出用户
func (s *ChannelService) KickoutFromVoiceChannel(ctx context.Context, channelID, userID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
		"user_id":    userID,
	}

	_, err := s.client.Post(ctx, "channel/kickout", params)
	return err
}

// MoveUser 移动单个用户到语音频道
func (s *ChannelService) MoveUser(ctx context.Context, channelID, userID string) error {
	return s.MoveUsers(ctx, channelID, []string{userID})
}

// MoveUsers 移动用户到语音频道
func (s *ChannelService) MoveUsers(ctx context.Context, channelID string, userIDs []string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if len(userIDs) == 0 {
		return fmt.Errorf("用户ID列表不能为空")
	}
	for _, userID := range userIDs {
		if userID == "" {
			return fmt.Errorf("用户ID不能为空")
		}
	}

	params := map[string]interface{}{
		"target_id": channelID,
		"user_ids":  userIDs,
	}

	_, err := s.client.Post(ctx, "channel/move-user", params)
	return err
}

// GetChannelRole 获取频道角色权限详情
func (s *ChannelService) GetChannelRole(ctx context.Context, channelID string) (*ChannelRoleResponse, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"channel_id": channelID,
	}

	resp, err := s.client.Get(ctx, "channel-role/index", query)
	if err != nil {
		return nil, err
	}

	var result ChannelRoleResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道权限详情失败: %w", err)
	}

	return &result, nil
}

// KickoutUser 踢出语音频道用户
func (s *ChannelService) KickoutUser(ctx context.Context, channelID, userID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if userID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
		"user_id":    userID,
	}

	_, err := s.client.Post(ctx, "channel/kickout", params)
	return err
}

// GetChannelUserList 获取频道内用户列表
func (s *ChannelService) GetChannelUserList(ctx context.Context, channelID string) ([]User, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"channel_id": channelID,
	}

	resp, err := s.client.Get(ctx, "channel/user-list", query)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := json.Unmarshal(resp.Data, &users); err != nil {
		return nil, fmt.Errorf("解析频道用户列表失败: %w", err)
	}

	return users, nil
}

// GetJoinedChannels 获取用户加入的语音频道列表
func (s *ChannelService) GetJoinedChannels(ctx context.Context, guildID, userID string) ([]Channel, error) {
	if guildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if userID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	resp, err := s.client.Get(ctx, "channel-user/get-joined-channel", map[string]string{
		"guild_id": guildID,
		"user_id":  userID,
	})
	if err != nil {
		return nil, err
	}

	var channels []Channel
	if err := json.Unmarshal(resp.Data, &channels); err != nil {
		return nil, fmt.Errorf("解析用户加入频道列表失败: %w", err)
	}

	return channels, nil
}

// SyncChannelRole 同步频道权限
func (s *ChannelService) SyncChannelRole(ctx context.Context, channelID string) (*ChannelRoleResponse, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}

	resp, err := s.client.Post(ctx, "channel-role/sync", params)
	if err != nil {
		return nil, err
	}

	var result ChannelRoleResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道权限同步结果失败: %w", err)
	}

	return &result, nil
}

// CreateChannelRole 创建频道角色权限
func (s *ChannelService) CreateChannelRole(ctx context.Context, params ChannelRoleParams) (*ChannelPermissionOverwrite, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := params.toMap(false)
	resp, err := s.client.Post(ctx, "channel-role/create", requestParams)
	if err != nil {
		return nil, err
	}

	var result ChannelPermissionOverwrite
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道权限创建结果失败: %w", err)
	}

	return &result, nil
}

// UpdateChannelRole 更新频道角色权限
func (s *ChannelService) UpdateChannelRole(ctx context.Context, params ChannelRoleParams) (*ChannelPermissionOverwrite, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := params.toMap(true)
	resp, err := s.client.Post(ctx, "channel-role/update", requestParams)
	if err != nil {
		return nil, err
	}

	var result ChannelPermissionOverwrite
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道权限更新结果失败: %w", err)
	}

	return &result, nil
}

// DeleteChannelRole 删除频道角色权限
func (s *ChannelService) DeleteChannelRole(ctx context.Context, channelID, overwriteType, value string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}
	if overwriteType != "" {
		params["type"] = overwriteType
	}
	if value != "" {
		params["value"] = value
	}

	_, err := s.client.Post(ctx, "channel-role/delete", params)
	return err
}

// CreateChannelParams 创建频道参数
type CreateChannelParams struct {
	Name         string `json:"name"`                    // 频道名称
	Type         int    `json:"type,omitempty"`          // 频道类型：1文字，2语音
	ParentID     string `json:"parent_id,omitempty"`     // 父分组ID
	LimitAmount  int    `json:"limit_amount,omitempty"`  // 语音频道人数限制
	VoiceQuality int    `json:"voice_quality,omitempty"` // 语音质量
	IsCategory   bool   `json:"is_category,omitempty"`   // 是否为分组
}

// UpdateChannelParams 更新频道参数
type UpdateChannelParams struct {
	Name         string `json:"name,omitempty"`          // 频道名称
	Topic        string `json:"topic,omitempty"`         // 频道主题
	SlowMode     int    `json:"slow_mode,omitempty"`     // 慢速模式（秒）
	LimitAmount  int    `json:"limit_amount,omitempty"`  // 语音频道人数限制
	VoiceQuality int    `json:"voice_quality,omitempty"` // 语音质量
	Password     string `json:"password,omitempty"`      // 频道密码
}

// ListChannelsResponse 频道列表响应
type ListChannelsResponse struct {
	Items []Channel      `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// ChannelRoleResponse 频道角色权限响应
type ChannelRoleResponse struct {
	PermissionOverwrites []PermissionOverwrite `json:"permission_overwrites"`
	PermissionUsers      []PermissionUser      `json:"permission_users"`
	PermissionSync       int                   `json:"permission_sync"`
}

// ChannelRoleParams 创建或更新频道权限覆写参数
type ChannelRoleParams struct {
	ChannelID string `json:"channel_id"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value,omitempty"`
	Allow     int    `json:"allow,omitempty"`
	Deny      int    `json:"deny,omitempty"`
}

func (p ChannelRoleParams) toMap(includePermissions bool) map[string]interface{} {
	params := map[string]interface{}{"channel_id": p.ChannelID}
	if p.Type != "" {
		params["type"] = p.Type
	}
	if p.Value != "" {
		params["value"] = p.Value
	}
	if includePermissions {
		params["allow"] = p.Allow
		params["deny"] = p.Deny
	}
	return params
}

// ChannelPermissionOverwrite 频道权限覆写结果
type ChannelPermissionOverwrite struct {
	UserID string `json:"user_id,omitempty"`
	RoleID int    `json:"role_id,omitempty"`
	Allow  int    `json:"allow"`
	Deny   int    `json:"deny"`
}
