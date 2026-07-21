package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// RoleService 角色相关API服务
type RoleService struct {
	client *Client
}

// GetRoleList 获取服务器角色列表
func (s *RoleService) GetRoleList(ctx context.Context, args ...any) (*ListRolesResponse, error) {
	params, err := compatParams("GetRoleList", args, func(args []any) (RoleListParams, bool) {
		if len(args) != 3 {
			return RoleListParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		page, ok2 := compatInt(args[1])
		pageSize, ok3 := compatInt(args[2])
		return RoleListParams{GuildID: guildID, Page: optionalPositiveInt(page), PageSize: optionalPositiveInt(pageSize)}, ok1 && ok2 && ok3
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

	resp, err := s.client.Get(ctx, "guild-role/list", query)
	if err != nil {
		return nil, err
	}

	var result ListRolesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析角色列表失败: %w", err)
	}

	return &result, nil
}

// CreateRole 创建服务器角色
func (s *RoleService) CreateRole(ctx context.Context, args ...any) (*GuildRole, error) {
	params, err := compatParams("CreateRole", args, func(args []any) (CreateRoleParams, bool) {
		if len(args) != 2 {
			return CreateRoleParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		name, ok2 := compatString(args[1])
		return CreateRoleParams{GuildID: guildID, Name: stringPointer(name)}, ok1 && ok2
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id": params.GuildID,
	}

	if params.Name != nil {
		body["name"] = *params.Name
	}

	resp, err := s.client.Post(ctx, "guild-role/create", body)
	if err != nil {
		return nil, err
	}

	return decodeRoleResult(resp.Data)
}

// UpdateRole 更新服务器角色
func (s *RoleService) UpdateRole(ctx context.Context, args ...any) (*GuildRole, error) {
	params, err := compatParams("UpdateRole", args, func(args []any) (UpdateRoleParams, bool) {
		if len(args) != 3 {
			return UpdateRoleParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		roleID, ok2 := compatInt(args[1])
		value, ok3 := args[2].(UpdateRoleParams)
		value.GuildID = guildID
		value.RoleID = roleID
		return value, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.RoleID <= 0 {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	requestParams := map[string]interface{}{
		"guild_id": params.GuildID,
		"role_id":  params.RoleID,
	}

	if params.Name != nil {
		requestParams["name"] = *params.Name
	}
	if params.Color != nil {
		requestParams["color"] = *params.Color
	}
	if params.Hoist != nil {
		requestParams["hoist"] = *params.Hoist
	}
	if params.Mentionable != nil {
		requestParams["mentionable"] = *params.Mentionable
	}
	if params.Permissions != nil {
		requestParams["permissions"] = *params.Permissions
	}

	resp, err := s.client.Post(ctx, "guild-role/update", requestParams)
	if err != nil {
		return nil, err
	}

	return decodeRoleResult(resp.Data)
}

// DeleteRole 删除服务器角色
func (s *RoleService) DeleteRole(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteRole", args, func(args []any) (DeleteRoleParams, bool) {
		if len(args) != 2 {
			return DeleteRoleParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		roleID, ok2 := compatInt(args[1])
		return DeleteRoleParams{GuildID: guildID, RoleID: roleID}, ok1 && ok2
	})
	if err != nil {
		return err
	}
	if params.GuildID == "" {
		return fmt.Errorf("服务器ID不能为空")
	}
	if params.RoleID <= 0 {
		return fmt.Errorf("角色ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id": params.GuildID,
		"role_id":  params.RoleID,
	}

	_, err = s.client.Post(ctx, "guild-role/delete", body)
	return err
}

// GrantRole 赋予用户角色
func (s *RoleService) GrantRole(ctx context.Context, args ...any) (*UserRoleResponse, error) {
	params, err := compatParams("GrantRole", args, func(args []any) (GrantRoleParams, bool) {
		if len(args) != 3 {
			return GrantRoleParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		roleID, ok3 := compatInt(args[2])
		return GrantRoleParams{GuildID: guildID, UserID: userID, RoleID: roleID}, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.UserID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}
	if params.RoleID <= 0 {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id": params.GuildID,
		"user_id":  params.UserID,
		"role_id":  params.RoleID,
	}

	resp, err := s.client.Post(ctx, "guild-role/grant", body)
	if err != nil {
		return nil, err
	}

	var result UserRoleResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析用户角色信息失败: %w", err)
	}

	return &result, nil
}

// RevokeRole 删除用户角色
func (s *RoleService) RevokeRole(ctx context.Context, args ...any) (*UserRoleResponse, error) {
	params, err := compatParams("RevokeRole", args, func(args []any) (RevokeRoleParams, bool) {
		if len(args) != 3 {
			return RevokeRoleParams{}, false
		}
		guildID, ok1 := compatString(args[0])
		userID, ok2 := compatString(args[1])
		roleID, ok3 := compatInt(args[2])
		return RevokeRoleParams{GuildID: guildID, UserID: userID, RoleID: roleID}, ok1 && ok2 && ok3
	})
	if err != nil {
		return nil, err
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.UserID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}
	if params.RoleID <= 0 {
		return nil, fmt.Errorf("角色ID不能为空")
	}

	body := map[string]interface{}{
		"guild_id": params.GuildID,
		"user_id":  params.UserID,
		"role_id":  params.RoleID,
	}

	resp, err := s.client.Post(ctx, "guild-role/revoke", body)
	if err != nil {
		return nil, err
	}

	var result UserRoleResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析用户角色信息失败: %w", err)
	}

	return &result, nil
}

// 数据结构定义

// GuildRole 服务器角色信息
type GuildRole struct {
	RoleID      int    `json:"role_id"`     // 角色ID
	Name        string `json:"name"`        // 角色名称
	Color       int    `json:"color"`       // 角色色值
	Position    int    `json:"position"`    // 角色位置
	Hoist       int    `json:"hoist"`       // 是否在用户列表排到前面
	Mentionable int    `json:"mentionable"` // 是否可以被提及
	Permissions int    `json:"permissions"` // 权限值
}

// UpdateRoleParams 更新角色参数
type UpdateRoleParams struct {
	GuildID     string
	RoleID      int
	Name        *string
	Color       *int
	Hoist       *int
	Mentionable *int
	Permissions *int
}

type RoleListParams struct {
	GuildID  string
	Page     *int
	PageSize *int
}

type CreateRoleParams struct {
	GuildID string
	Name    *string
}

type DeleteRoleParams struct {
	GuildID string
	RoleID  int
}

type GrantRoleParams struct {
	GuildID string
	UserID  string
	RoleID  int
}

type RevokeRoleParams struct {
	GuildID string
	UserID  string
	RoleID  int
}

func decodeRoleResult(data json.RawMessage) (*GuildRole, error) {
	var list []GuildRole
	if err := json.Unmarshal(data, &list); err == nil {
		if len(list) == 0 {
			return nil, fmt.Errorf("未返回角色信息")
		}
		return &list[0], nil
	}
	var role GuildRole
	if err := json.Unmarshal(data, &role); err != nil {
		return nil, fmt.Errorf("解析角色信息失败: %w", err)
	}
	return &role, nil
}

// ListRolesResponse 角色列表响应
type ListRolesResponse struct {
	Items []GuildRole    `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// UserRoleResponse 用户角色响应
type UserRoleResponse struct {
	UserID  string `json:"user_id"`  // 用户ID
	GuildID string `json:"guild_id"` // 服务器ID
	Roles   []int  `json:"roles"`    // 角色ID列表
}
