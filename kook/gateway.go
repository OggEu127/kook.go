package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// GatewayService 网关相关API服务
type GatewayService struct {
	client *Client
}

// GatewayParams 获取网关连接地址参数。
type GatewayParams struct {
	Compress *int
}

// GetGateway 获取网关连接信息
func (s *GatewayService) GetGateway(ctx context.Context, params GatewayParams) (*Gateway, error) {
	query := make(map[string]string)
	if params.Compress != nil {
		if *params.Compress != 0 && *params.Compress != 1 {
			return nil, fmt.Errorf("compress必须为0或1")
		}
		query["compress"] = fmt.Sprintf("%d", *params.Compress)
	}

	resp, err := s.client.Get(ctx, "gateway/index", query)
	if err != nil {
		return nil, err
	}

	var gateway Gateway
	if err := json.Unmarshal(resp.Data, &gateway); err != nil {
		return nil, fmt.Errorf("解析网关信息失败: %w", err)
	}

	return &gateway, nil
}
