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

// JoinVoiceChannel 加入语音频道
func (s *VoiceService) JoinVoiceChannel(ctx context.Context, channelID string) (*VoiceConnectionInfo, error) {
	return s.JoinVoiceChannelWithParams(ctx, VoiceJoinParams{ChannelID: channelID})
}

// JoinVoiceChannelWithParams 加入语音频道
func (s *VoiceService) JoinVoiceChannelWithParams(ctx context.Context, params VoiceJoinParams) (*VoiceConnectionInfo, error) {
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

// LeaveVoiceChannel 离开语音频道
func (s *VoiceService) LeaveVoiceChannel(ctx context.Context, channelID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}

	_, err := s.client.Post(ctx, "voice/leave", params)
	return err
}

// GetVoiceChannelUsers 获取语音频道用户列表
func (s *VoiceService) GetVoiceChannelUsers(ctx context.Context, channelID string) ([]VoiceUser, error) {
	return nil, fmt.Errorf("KOOK v3 官方接口未提供 voice/users；请改用 GetJoinedVoiceChannels")
}

// MuteUser 静音用户
func (s *VoiceService) MuteUser(ctx context.Context, channelID, userID string) error {
	return fmt.Errorf("KOOK v3 官方接口未提供 voice/mute")
}

// UnmuteUser 取消静音用户
func (s *VoiceService) UnmuteUser(ctx context.Context, channelID, userID string) error {
	return fmt.Errorf("KOOK v3 官方接口未提供 voice/unmute")
}

// DeafenUser 闭麦用户
func (s *VoiceService) DeafenUser(ctx context.Context, channelID, userID string) error {
	return fmt.Errorf("KOOK v3 官方接口未提供 voice/deafen")
}

// UndeafenUser 取消闭麦用户
func (s *VoiceService) UndeafenUser(ctx context.Context, channelID, userID string) error {
	return fmt.Errorf("KOOK v3 官方接口未提供 voice/undeafen")
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

// KeepAliveVoiceChannel 续期语音频道占用
func (s *VoiceService) KeepAliveVoiceChannel(ctx context.Context, channelID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}

	_, err := s.client.Post(ctx, "voice/keep-alive", params)
	return err
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
	Sort  map[string]int `json:"sort"`
}

// VoiceUser 语音频道用户
type VoiceUser struct {
	User         User `json:"user"`          // 用户信息
	Muted        bool `json:"muted"`         // 是否被静音
	Deafened     bool `json:"deafened"`      // 是否被闭麦
	SelfMuted    bool `json:"self_muted"`    // 是否自我静音
	SelfDeafened bool `json:"self_deafened"` // 是否自我闭麦
	Speaking     bool `json:"speaking"`      // 是否正在说话
}
