package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ChannelUserService 频道用户相关接口。
type ChannelUserService struct {
	client *Client
}

// JoinedChannelParams 查询用户所在语音频道的参数。
type JoinedChannelParams struct {
	GuildID  string
	UserID   string
	Page     *int
	PageSize *int
}

// GetJoinedChannels 根据用户和服务器获取所在语音频道。
func (s *ChannelUserService) GetJoinedChannels(ctx context.Context, params JoinedChannelParams) (*ListChannelsResponse, error) {
	if params.GuildID == "" || params.UserID == "" {
		return nil, fmt.Errorf("服务器ID和用户ID不能为空")
	}
	query := map[string]string{"guild_id": params.GuildID, "user_id": params.UserID}
	if params.Page != nil {
		query["page"] = strconv.Itoa(*params.Page)
	}
	if params.PageSize != nil {
		query["page_size"] = strconv.Itoa(*params.PageSize)
	}
	resp, err := s.client.Get(ctx, "channel-user/get-joined-channel", query)
	if err != nil {
		return nil, err
	}
	var result ListChannelsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析用户语音频道失败: %w", err)
	}
	return &result, nil
}
