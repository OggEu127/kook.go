package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// FriendService 好友相关API服务
type FriendService struct {
	client *Client
}

// SendFriendRequest 发送好友请求
func (s *FriendService) SendFriendRequest(ctx context.Context, params SendFriendRequestParams) error {
	if params.UserCode == "" {
		return fmt.Errorf("用户识别码不能为空")
	}

	requestParams := map[string]interface{}{
		"user_code": params.UserCode,
		"from":      params.From,
	}

	if params.From == FriendRequestFromGuild {
		if params.GuildID == "" {
			return fmt.Errorf("从服务器发送好友申请时guild_id不能为空")
		}
		requestParams["guild_id"] = params.GuildID
	}

	_, err := s.client.Post(ctx, "friend/request", requestParams)
	return err
}

type FriendListParams struct {
	Type string
}

// GetFriendsList 获取好友、申请与屏蔽列表。
func (s *FriendService) GetFriendsList(ctx context.Context, params FriendListParams) (*FriendsListResponse, error) {
	var query map[string]string
	if params.Type != "" {
		query = map[string]string{"type": params.Type}
	}

	resp, err := s.client.Get(ctx, "friend", query)
	if err != nil {
		return nil, err
	}

	var result FriendsListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析好友列表失败: %w", err)
	}

	return &result, nil
}

type DeleteFriendParams struct {
	UserID string
}

// DeleteFriend 删除好友。
func (s *FriendService) DeleteFriend(ctx context.Context, params DeleteFriendParams) error {
	if params.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	body := map[string]interface{}{
		"user_id": params.UserID,
	}

	_, err := s.client.Post(ctx, "friend/delete", body)
	return err
}

type HandleFriendRequestParams struct {
	ID     int
	Accept bool
}

// HandleFriendRequest 处理好友请求。
func (s *FriendService) HandleFriendRequest(ctx context.Context, params HandleFriendRequestParams) (bool, error) {
	if params.ID <= 0 {
		return false, fmt.Errorf("请求ID不能为空")
	}

	body := map[string]interface{}{
		"id":     params.ID,
		"accept": params.Accept,
	}

	resp, err := s.client.Post(ctx, "friend/handle-request", body)
	if err != nil {
		return false, err
	}
	var accepted bool
	if err := json.Unmarshal(resp.Data, &accepted); err != nil {
		return false, fmt.Errorf("解析好友申请处理结果失败: %w", err)
	}
	return accepted, nil
}

type BlockFriendParams struct {
	UserID string
}

// BlockFriend 屏蔽用户。
func (s *FriendService) BlockFriend(ctx context.Context, params BlockFriendParams) error {
	if params.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	_, err := s.client.Post(ctx, "friend/block", map[string]interface{}{
		"user_id": params.UserID,
	})
	return err
}

type UnblockFriendParams struct {
	UserID string
}

// UnblockFriend 取消屏蔽用户。
func (s *FriendService) UnblockFriend(ctx context.Context, params UnblockFriendParams) error {
	if params.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	_, err := s.client.Post(ctx, "friend/unblock", map[string]interface{}{
		"user_id": params.UserID,
	})
	return err
}

// 数据结构定义

// SendFriendRequestParams 发送好友请求参数
type SendFriendRequestParams struct {
	UserCode string `json:"user_code"`          // 用户识别码，格式: username#identify_num
	From     int    `json:"from"`               // 请求来源：0直接添加，1普通添加，2从服务器添加
	GuildID  string `json:"guild_id,omitempty"` // 服务器ID（当from=2时必填）
}

// FriendRelation 是好友、好友申请或屏蔽列表中的关系项。
type FriendRelation struct {
	ID         int    `json:"id"`
	Type       string `json:"type"`
	FriendInfo User   `json:"friend_info"`
	Own        bool   `json:"own"`
}

// FriendsListResponse 好友列表响应
type FriendsListResponse struct {
	Request []FriendRelation `json:"request"`
	Friend  []FriendRelation `json:"friend"`
	Block   []FriendRelation `json:"block"`
	Blocked []FriendRelation `json:"blocked"`
}

// UnmarshalJSON 兼容官方字段表中的 block 和实际示例中的 blocked。
func (r *FriendsListResponse) UnmarshalJSON(data []byte) error {
	type responseAlias FriendsListResponse
	var value responseAlias
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Block == nil {
		value.Block = value.Blocked
	}
	if value.Blocked == nil {
		value.Blocked = value.Block
	}
	*r = FriendsListResponse(value)
	return nil
}

// 好友请求来源常量
const (
	FriendRequestFromDirect = 0 // 直接添加
	FriendRequestFromNormal = 1 // 普通添加
	FriendRequestFromGuild  = 2 // 从服务器添加
)
