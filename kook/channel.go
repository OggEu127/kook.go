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
func (s *ChannelService) GetChannelList(ctx context.Context, args ...any) (*ListChannelsResponse, error) {
	params, err := compatParams("GetChannelList", args, func(args []any) (ChannelListParams, bool) {
		if len(args) != 4 {
			return ChannelListParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		page, okPage := compatInt(args[1])
		pageSize, okPageSize := compatInt(args[2])
		_, okSort := compatString(args[3])
		return ChannelListParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, okGuild && okPage && okPageSize && okSort
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	resp, err := s.client.Get(ctx, "channel/list", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result ListChannelsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析频道列表失败: %w", err)
	}

	return &result, nil
}

// View 获取频道详情，并可请求子频道列表。
func (s *ChannelService) View(ctx context.Context, params ChannelViewParams) (*Channel, error) {
	channelID := params.TargetID
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"target_id": channelID,
	}
	if params.NeedChildren != nil {
		query["need_children"] = strconv.FormatBool(*params.NeedChildren)
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
func (s *ChannelService) CreateChannel(ctx context.Context, args ...any) (*Channel, error) {
	params, err := compatParams("CreateChannel", args, func(args []any) (CreateChannelParams, bool) {
		if len(args) != 2 {
			return CreateChannelParams{}, false
		}
		guildID, okGuild := compatString(args[0])
		params, okParams := args[1].(CreateChannelParams)
		params.GuildID = guildID
		return params, okGuild && okParams
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("频道名称不能为空")
	}

	requestParams := map[string]interface{}{
		"guild_id": params.GuildID,
		"name":     params.Name,
	}

	if params.IsCategory != nil && *params.IsCategory {
		requestParams["is_category"] = 1
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
	if params.IsCategory != nil {
		requestParams["is_category"] = 0
	}

	if params.Type != nil {
		requestParams["type"] = *params.Type
	}

	if params.ParentID != "" {
		requestParams["parent_id"] = params.ParentID
	}
	if params.LimitAmount != nil {
		requestParams["limit_amount"] = *params.LimitAmount
	}
	if params.VoiceQuality != nil {
		requestParams["voice_quality"] = *params.VoiceQuality
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
func (s *ChannelService) UpdateChannel(ctx context.Context, args ...any) (*Channel, error) {
	params, err := compatParams("UpdateChannel", args, func(args []any) (UpdateChannelParams, bool) {
		if len(args) != 2 {
			return UpdateChannelParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		params, okParams := args[1].(UpdateChannelParams)
		params.ChannelID = channelID
		return params, okChannel && okParams
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := map[string]interface{}{
		"channel_id": params.ChannelID,
	}

	if params.Name != nil {
		requestParams["name"] = *params.Name
	}
	if params.Level != nil {
		requestParams["level"] = *params.Level
	}
	if params.ParentID != nil {
		requestParams["parent_id"] = *params.ParentID
	}
	if params.Topic != nil {
		requestParams["topic"] = *params.Topic
	}
	if params.SlowMode != nil {
		requestParams["slow_mode"] = *params.SlowMode
	}
	if params.LimitAmount != nil {
		requestParams["limit_amount"] = *params.LimitAmount
	}
	if params.VoiceQuality != nil {
		requestParams["voice_quality"] = *params.VoiceQuality
	}
	if params.Password != nil {
		requestParams["password"] = *params.Password
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
func (s *ChannelService) DeleteChannel(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteChannel", args, func(args []any) (ChannelDeleteParams, bool) {
		if len(args) != 1 {
			return ChannelDeleteParams{}, false
		}
		channelID, ok := compatString(args[0])
		return ChannelDeleteParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return err
	}
	if params.ChannelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	body := map[string]interface{}{
		"channel_id": params.ChannelID,
	}

	_, err = s.client.Post(ctx, "channel/delete", body)
	return err
}

// MoveUsers 移动用户到语音频道
func (s *ChannelService) MoveUsers(ctx context.Context, args ...any) error {
	params, err := compatParams("MoveUsers", args, func(args []any) (ChannelMoveUserParams, bool) {
		if len(args) != 2 {
			return ChannelMoveUserParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		userIDs, okUsers := args[1].([]string)
		return ChannelMoveUserParams{TargetID: channelID, UserIDs: userIDs}, okChannel && okUsers
	})
	if err != nil {
		return err
	}
	if params.TargetID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if len(params.UserIDs) == 0 {
		return fmt.Errorf("用户ID列表不能为空")
	}
	for _, userID := range params.UserIDs {
		if userID == "" {
			return fmt.Errorf("用户ID不能为空")
		}
	}

	body := map[string]interface{}{
		"target_id": params.TargetID,
		"user_ids":  params.UserIDs,
	}

	_, err = s.client.Post(ctx, "channel/move-user", body)
	return err
}

// GetChannelRole 获取频道角色权限详情
func (s *ChannelService) GetChannelRole(ctx context.Context, args ...any) (*ChannelRoleResponse, error) {
	params, err := compatParams("GetChannelRole", args, func(args []any) (ChannelRoleViewParams, bool) {
		if len(args) != 1 {
			return ChannelRoleViewParams{}, false
		}
		channelID, ok := compatString(args[0])
		return ChannelRoleViewParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"channel_id": params.ChannelID,
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
func (s *ChannelService) KickoutUser(ctx context.Context, args ...any) error {
	params, err := compatParams("KickoutUser", args, func(args []any) (ChannelKickoutParams, bool) {
		if len(args) != 2 {
			return ChannelKickoutParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		userID, okUser := compatString(args[1])
		return ChannelKickoutParams{ChannelID: channelID, UserID: userID}, okChannel && okUser
	})
	if err != nil {
		return err
	}
	if params.ChannelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if params.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	body := map[string]interface{}{
		"channel_id": params.ChannelID,
		"user_id":    params.UserID,
	}

	_, err = s.client.Post(ctx, "channel/kickout", body)
	return err
}

// GetChannelUserList 获取频道内用户列表
func (s *ChannelService) GetChannelUserList(ctx context.Context, args ...any) ([]User, error) {
	params, err := compatParams("GetChannelUserList", args, func(args []any) (ChannelUserListParams, bool) {
		if len(args) != 1 {
			return ChannelUserListParams{}, false
		}
		channelID, ok := compatString(args[0])
		return ChannelUserListParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	query := map[string]string{
		"channel_id": params.ChannelID,
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

// SyncChannelRole 同步频道权限
func (s *ChannelService) SyncChannelRole(ctx context.Context, args ...any) (*ChannelRoleResponse, error) {
	params, err := compatParams("SyncChannelRole", args, func(args []any) (ChannelRoleSyncParams, bool) {
		if len(args) != 1 {
			return ChannelRoleSyncParams{}, false
		}
		channelID, ok := compatString(args[0])
		return ChannelRoleSyncParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	body := map[string]interface{}{
		"channel_id": params.ChannelID,
	}

	resp, err := s.client.Post(ctx, "channel-role/sync", body)
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
func (s *ChannelService) CreateChannelRole(ctx context.Context, args ...any) (*ChannelPermissionOverwrite, error) {
	params, err := compatParams("CreateChannelRole", args, func(args []any) (CreateChannelRoleParams, bool) {
		if len(args) != 1 {
			return CreateChannelRoleParams{}, false
		}
		legacy, ok := args[0].(ChannelRoleParams)
		return CreateChannelRoleParams{ChannelID: legacy.ChannelID, Type: legacy.Type, Value: legacy.Value}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := params.toMap()
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
func (s *ChannelService) UpdateChannelRole(ctx context.Context, args ...any) (*ChannelPermissionOverwrite, error) {
	params, err := compatParams("UpdateChannelRole", args, func(args []any) (UpdateChannelRoleParams, bool) {
		if len(args) != 1 {
			return UpdateChannelRoleParams{}, false
		}
		legacy, ok := args[0].(ChannelRoleParams)
		return UpdateChannelRoleParams{ChannelID: legacy.ChannelID, Type: legacy.Type, Value: legacy.Value, Allow: &legacy.Allow, Deny: &legacy.Deny}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	requestParams := params.toMap()
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
func (s *ChannelService) DeleteChannelRole(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteChannelRole", args, func(args []any) (DeleteChannelRoleParams, bool) {
		if len(args) != 3 {
			return DeleteChannelRoleParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		overwriteType, okType := compatString(args[1])
		value, okValue := compatString(args[2])
		return DeleteChannelRoleParams{ChannelID: channelID, Type: overwriteType, Value: value}, okChannel && okType && okValue
	})
	if err != nil {
		return err
	}
	if params.ChannelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	body := map[string]interface{}{
		"channel_id": params.ChannelID,
	}
	if params.Type != "" {
		body["type"] = params.Type
	}
	if params.Value != "" {
		body["value"] = params.Value
	}

	_, err = s.client.Post(ctx, "channel-role/delete", body)
	return err
}

// GetChannelInfo 是 View 的 v1.1.1 兼容别名。
func (s *ChannelService) GetChannelInfo(ctx context.Context, channelID string) (*Channel, error) {
	return s.View(ctx, ChannelViewParams{TargetID: channelID})
}

func (s *ChannelService) MoveChannel(context.Context, string, []string) error {
	return unsupportedEndpoint("channel/move")
}

func (s *ChannelService) KickoutFromVoiceChannel(ctx context.Context, channelID, userID string) error {
	return s.KickoutUser(ctx, channelID, userID)
}

func (s *ChannelService) MoveUser(ctx context.Context, channelID, userID string) error {
	return s.MoveUsers(ctx, channelID, []string{userID})
}

func (s *ChannelService) GetJoinedChannels(ctx context.Context, guildID, userID string) ([]Channel, error) {
	result, err := (&ChannelUserService{client: s.client}).GetJoinedChannels(ctx, JoinedChannelParams{GuildID: guildID, UserID: userID})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// CreateChannelParams 创建频道参数
type CreateChannelParams struct {
	GuildID      string
	Name         string
	Type         *int
	ParentID     string
	LimitAmount  *int
	VoiceQuality *string
	IsCategory   *bool
}

// ChannelListParams 频道列表参数
type ChannelListParams struct {
	GuildID  string
	Page     *int
	PageSize *int
	Type     *int
	ParentID string
}

// ChannelViewParams 获取频道详情参数。
type ChannelViewParams struct {
	TargetID     string
	NeedChildren *bool
}

func (p ChannelListParams) toQuery() map[string]string {
	query := map[string]string{
		"guild_id": p.GuildID,
	}
	if p.Page != nil {
		query["page"] = strconv.Itoa(*p.Page)
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	if p.Type != nil {
		query["type"] = strconv.Itoa(*p.Type)
	}
	if p.ParentID != "" {
		query["parent_id"] = p.ParentID
	}
	return query
}

// UpdateChannelParams 更新频道参数
type UpdateChannelParams struct {
	ChannelID    string
	Name         *string
	Level        *int
	ParentID     *string
	Topic        *string
	SlowMode     *int
	LimitAmount  *int
	VoiceQuality *string
	Password     *string
}

type ChannelDeleteParams struct {
	ChannelID string
}

type ChannelUserListParams struct {
	ChannelID string
}

type ChannelMoveUserParams struct {
	TargetID string
	UserIDs  []string
}

type ChannelKickoutParams struct {
	ChannelID string
	UserID    string
}

type ChannelRoleViewParams struct {
	ChannelID string
}

type ChannelRoleSyncParams struct {
	ChannelID string
}

// ListChannelsResponse 频道列表响应
type ListChannelsResponse struct {
	Items []Channel      `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// ChannelRoleResponse 频道角色权限响应
type ChannelRoleResponse struct {
	PermissionOverwrites []PermissionOverwrite `json:"permission_overwrites"`
	PermissionUsers      []PermissionUser      `json:"permission_users"`
	PermissionSync       int                   `json:"permission_sync"`
}

type CreateChannelRoleParams struct {
	ChannelID string
	Type      string
	Value     string
}

func (p CreateChannelRoleParams) toMap() map[string]interface{} {
	params := map[string]interface{}{"channel_id": p.ChannelID}
	if p.Type != "" {
		params["type"] = p.Type
	}
	if p.Value != "" {
		params["value"] = p.Value
	}
	return params
}

type UpdateChannelRoleParams struct {
	ChannelID string
	Type      string
	Value     string
	Allow     *int
	Deny      *int
}

func (p UpdateChannelRoleParams) toMap() map[string]interface{} {
	params := map[string]interface{}{"channel_id": p.ChannelID}
	if p.Type != "" {
		params["type"] = p.Type
	}
	if p.Value != "" {
		params["value"] = p.Value
	}
	if p.Allow != nil {
		params["allow"] = *p.Allow
	}
	if p.Deny != nil {
		params["deny"] = *p.Deny
	}
	return params
}

type DeleteChannelRoleParams struct {
	ChannelID string
	Type      string
	Value     string
}

// ChannelRoleParams 保留 v1.1.1 的合并创建/更新权限参数。
type ChannelRoleParams struct {
	ChannelID string `json:"channel_id"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value,omitempty"`
	Allow     int    `json:"allow,omitempty"`
	Deny      int    `json:"deny,omitempty"`
}

// ChannelPermissionOverwrite 频道权限覆写结果
type ChannelPermissionOverwrite struct {
	UserID string `json:"user_id,omitempty"`
	RoleID int    `json:"role_id,omitempty"`
	Allow  int    `json:"allow"`
	Deny   int    `json:"deny"`
}

func (p *ChannelPermissionOverwrite) UnmarshalJSON(data []byte) error {
	type overwriteAlias ChannelPermissionOverwrite
	value := struct {
		*overwriteAlias
		UserID json.RawMessage `json:"user_id"`
		RoleID json.RawMessage `json:"role_id"`
	}{overwriteAlias: (*overwriteAlias)(p)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.UserID) > 0 {
		userID, err := decodeStringOrNumber(value.UserID)
		if err != nil {
			return fmt.Errorf("解析channel permission user_id失败: %w", err)
		}
		p.UserID = userID
	}
	if len(value.RoleID) > 0 {
		roleID, err := decodeIntOrString(value.RoleID)
		if err != nil {
			return fmt.Errorf("解析channel permission role_id失败: %w", err)
		}
		p.RoleID = roleID
	}
	return nil
}
