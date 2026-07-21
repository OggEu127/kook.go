package kook

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
)

// BadgeService Badge 图片接口。
type BadgeService struct {
	client *Client
}

// Badge 保留 v1.1.1 中未被当前官方徽章图片接口确认的数据类型。
type Badge struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
	Type        int    `json:"type"`
	Level       int    `json:"level"`
	Unlocked    bool   `json:"unlocked"`
}

// GetGuildBadges 是 v1.1.1 兼容入口，不会访问未确认端点。
func (s *BadgeService) GetGuildBadges(context.Context, string) ([]Badge, error) {
	return nil, unsupportedEndpoint("badge/guild")
}

// BadgeStyle Badge 展示样式。
type BadgeStyle int

const (
	BadgeStyleGuildName BadgeStyle = iota
	BadgeStyleOnlineCount
	BadgeStyleOnlineAndTotal
)

// BadgeParams 获取服务器 Badge 的参数。
type BadgeParams struct {
	GuildID string
	Style   BadgeStyle
}

// GetGuildBadge 获取服务器 Badge 图片。
func (s *BadgeService) GetGuildBadge(ctx context.Context, params BadgeParams) (*BinaryResponse, error) {
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.Style < BadgeStyleGuildName || params.Style > BadgeStyleOnlineAndTotal {
		return nil, fmt.Errorf("Badge样式必须为0、1或2")
	}
	return s.client.doBinaryRequest(ctx, http.MethodGet, "badge/guild", map[string]string{
		"guild_id": params.GuildID,
		"style":    strconv.Itoa(int(params.Style)),
	})
}
