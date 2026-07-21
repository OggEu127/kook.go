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
func (s *GatewayService) GetGateway(ctx context.Context, args ...any) (*Gateway, error) {
	params, err := compatParams("GetGateway", args, func(args []any) (GatewayParams, bool) {
		if len(args) != 1 {
			return GatewayParams{}, false
		}
		compress, ok := compatInt(args[0])
		return GatewayParams{Compress: &compress}, ok
	})
	if err != nil {
		return nil, err
	}
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

// GetVoiceGateway 是 v1.1.1 兼容入口。官方文档未确认该端点。
func (s *GatewayService) GetVoiceGateway(context.Context, string) (*VoiceGateway, error) {
	return nil, unsupportedEndpoint("gateway/voice")
}

// VoiceGateway 保留 v1.1.1 的语音网关响应类型。
type VoiceGateway struct {
	GatewayURL  string `json:"gateway_url"`
	IosVoiceSDK int    `json:"ios_voice_sdk"`
	PCVoiceSDK  int    `json:"pc_voice_sdk"`
}
