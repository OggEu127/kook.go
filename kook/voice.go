package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// VoiceService 语音相关API服务
type VoiceService struct {
	client *Client
}

// JoinVoiceChannel 加入语音频道。
func (s *VoiceService) JoinVoiceChannel(ctx context.Context, args ...any) (*VoiceConnectionInfo, error) {
	params, err := compatParams("JoinVoiceChannel", args, func(args []any) (VoiceJoinParams, bool) {
		if len(args) != 1 {
			return VoiceJoinParams{}, false
		}
		channelID, ok := compatString(args[0])
		return VoiceJoinParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	resp, err := s.client.Post(ctx, "voice/join", params.toMap())
	if err != nil {
		return nil, err
	}

	var connInfo VoiceConnectionInfo
	if err := json.Unmarshal(resp.Data, &connInfo); err != nil {
		return nil, fmt.Errorf("解析语音连接信息失败: %w", err)
	}

	return &connInfo, nil
}

type VoiceLeaveParams struct {
	ChannelID string
}

// LeaveVoiceChannel 离开语音频道。
func (s *VoiceService) LeaveVoiceChannel(ctx context.Context, args ...any) error {
	params, err := compatParams("LeaveVoiceChannel", args, func(args []any) (VoiceLeaveParams, bool) {
		if len(args) != 1 {
			return VoiceLeaveParams{}, false
		}
		channelID, ok := compatString(args[0])
		return VoiceLeaveParams{ChannelID: channelID}, ok
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

	_, err = s.client.Post(ctx, "voice/leave", body)
	return err
}

// GetJoinedVoiceChannels 获取机器人已加入的语音频道列表
func (s *VoiceService) GetJoinedVoiceChannels(ctx context.Context) (*ListVoiceChannelsResponse, error) {
	resp, err := s.client.Get(ctx, "voice/list", nil)
	if err != nil {
		return nil, err
	}

	var result ListVoiceChannelsResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析语音频道列表失败: %w", err)
	}
	return &result, nil
}

type VoiceKeepAliveParams struct {
	ChannelID string
}

// KeepAliveVoiceChannel 续期语音频道占用。
func (s *VoiceService) KeepAliveVoiceChannel(ctx context.Context, args ...any) error {
	params, err := compatParams("KeepAliveVoiceChannel", args, func(args []any) (VoiceKeepAliveParams, bool) {
		if len(args) != 1 {
			return VoiceKeepAliveParams{}, false
		}
		channelID, ok := compatString(args[0])
		return VoiceKeepAliveParams{ChannelID: channelID}, ok
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

	_, err = s.client.Post(ctx, "voice/keep-alive", body)
	return err
}

// GetVoiceChannelUsers 是 v1.1.1 兼容入口。官方文档未确认该端点。
func (s *VoiceService) GetVoiceChannelUsers(context.Context, string) ([]VoiceUser, error) {
	return nil, unsupportedEndpoint("voice/users")
}

func (s *VoiceService) MuteUser(context.Context, string, string) error {
	return unsupportedEndpoint("voice/mute")
}

func (s *VoiceService) UnmuteUser(context.Context, string, string) error {
	return unsupportedEndpoint("voice/unmute")
}

func (s *VoiceService) DeafenUser(context.Context, string, string) error {
	return unsupportedEndpoint("voice/deafen")
}

func (s *VoiceService) UndeafenUser(context.Context, string, string) error {
	return unsupportedEndpoint("voice/undeafen")
}

// VoiceUser 保留 v1.1.1 的语音成员数据结构。
type VoiceUser struct {
	User         User `json:"user"`
	Muted        bool `json:"muted"`
	Deafened     bool `json:"deafened"`
	SelfMuted    bool `json:"self_muted"`
	SelfDeafened bool `json:"self_deafened"`
	Speaking     bool `json:"speaking"`
}

// 数据结构定义

// VoiceJoinParams 加入语音频道参数
type VoiceJoinParams struct {
	ChannelID string `json:"channel_id"`
	AudioSSRC string `json:"audio_ssrc,omitempty"`
	AudioPT   string `json:"audio_pt,omitempty"`
	RTCPMux   *bool  `json:"rtcp_mux,omitempty"`
	Password  string `json:"password,omitempty"`
}

func (p VoiceJoinParams) toMap() map[string]interface{} {
	params := map[string]interface{}{
		"channel_id": p.ChannelID,
	}
	if p.AudioSSRC != "" {
		params["audio_ssrc"] = p.AudioSSRC
	}
	if p.AudioPT != "" {
		params["audio_pt"] = p.AudioPT
	}
	if p.RTCPMux != nil {
		params["rtcp_mux"] = *p.RTCPMux
	}
	if p.Password != "" {
		params["password"] = p.Password
	}
	return params
}

// VoiceConnectionInfo 语音连接信息
type VoiceConnectionInfo struct {
	IP        string `json:"ip"`         // 媒体服务器推流IP
	Port      string `json:"port"`       // 媒体服务器推流端口
	RTCPMux   bool   `json:"rtcp_mux"`   // 是否将RTCP与RTP使用同一个端口
	RTCPPort  string `json:"rtcp_port"`  // RTCP推流端口
	Bitrate   int    `json:"bitrate"`    // 当前语音房间要求的比特率
	AudioSSRC string `json:"audio_ssrc"` // 最终的SSRC
	AudioPT   string `json:"audio_pt"`   // 最终的Payload Type
}

// UnmarshalJSON 兼容 port/rtcp_port 在文档表格和示例中的 int/string 差异。
func (v *VoiceConnectionInfo) UnmarshalJSON(data []byte) error {
	type voiceConnectionAlias VoiceConnectionInfo
	value := struct {
		*voiceConnectionAlias
		Port     json.RawMessage `json:"port"`
		RTCPPort json.RawMessage `json:"rtcp_port"`
	}{voiceConnectionAlias: (*voiceConnectionAlias)(v)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Port) > 0 {
		port, err := decodeStringOrNumber(value.Port)
		if err != nil {
			return fmt.Errorf("解析voice.port失败: %w", err)
		}
		v.Port = port
	}
	if len(value.RTCPPort) > 0 {
		port, err := decodeStringOrNumber(value.RTCPPort)
		if err != nil {
			return fmt.Errorf("解析voice.rtcp_port失败: %w", err)
		}
		v.RTCPPort = port
	}
	return nil
}

// VoiceChannel 机器人已加入的语音频道
type VoiceChannel struct {
	ID       string `json:"id"`        // 频道ID
	GuildID  string `json:"guild_id"`  // 服务器ID
	ParentID string `json:"parent_id"` // 父频道ID
	Name     string `json:"name"`      // 频道名称
}

// ListVoiceChannelsResponse 语音频道列表响应
type ListVoiceChannelsResponse struct {
	Items []VoiceChannel `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}
