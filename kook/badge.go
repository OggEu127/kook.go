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
