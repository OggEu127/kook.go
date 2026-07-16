package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// GameService 游戏/动态相关API服务
type GameService struct {
	client *Client
}

type GameListParams struct {
	Type string
}

// GetGameList 获取游戏列表。
func (s *GameService) GetGameList(ctx context.Context, params GameListParams) (*ListGamesResponse, error) {
	query := make(map[string]string)
	if params.Type != "" {
		query["type"] = params.Type
	}

	resp, err := s.client.Get(ctx, "game", query)
	if err != nil {
		return nil, err
	}

	var result ListGamesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析游戏列表失败: %w", err)
	}

	return &result, nil
}

type GameCreateParams struct {
	Name string
	Icon string
}

// CreateGame 添加游戏。
func (s *GameService) CreateGame(ctx context.Context, params GameCreateParams) (*Game, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("游戏名称不能为空")
	}

	body := map[string]interface{}{
		"name": params.Name,
	}

	if params.Icon != "" {
		body["icon"] = params.Icon
	}

	resp, err := s.client.Post(ctx, "game/create", body)
	if err != nil {
		return nil, err
	}

	var game Game
	if err := json.Unmarshal(resp.Data, &game); err != nil {
		return nil, fmt.Errorf("解析游戏信息失败: %w", err)
	}

	return &game, nil
}

type GameUpdateParams struct {
	ID   int
	Name *string
	Icon *string
}

// UpdateGame 更新游戏。
func (s *GameService) UpdateGame(ctx context.Context, params GameUpdateParams) (*Game, error) {
	if params.ID <= 0 {
		return nil, fmt.Errorf("游戏ID不能为空")
	}

	body := map[string]interface{}{
		"id": params.ID,
	}

	if params.Name != nil {
		body["name"] = *params.Name
	}
	if params.Icon != nil {
		body["icon"] = *params.Icon
	}

	resp, err := s.client.Post(ctx, "game/update", body)
	if err != nil {
		return nil, err
	}

	var game Game
	if err := json.Unmarshal(resp.Data, &game); err != nil {
		return nil, fmt.Errorf("解析游戏信息失败: %w", err)
	}

	return &game, nil
}

type GameDeleteParams struct {
	ID int
}

// DeleteGame 删除游戏。
func (s *GameService) DeleteGame(ctx context.Context, params GameDeleteParams) error {
	if params.ID <= 0 {
		return fmt.Errorf("游戏ID不能为空")
	}

	body := map[string]interface{}{
		"id": params.ID,
	}

	_, err := s.client.Post(ctx, "game/delete", body)
	return err
}

type GameActivityParams struct {
	ID        *int
	DataType  int
	Software  string
	Singer    string
	MusicName string
}

// AddActivity 添加游戏或音乐动态。
func (s *GameService) AddActivity(ctx context.Context, params GameActivityParams) error {
	if params.DataType != GameActivityTypeGame && params.DataType != GameActivityTypeMusic {
		return fmt.Errorf("数据类型必须为1（游戏）或2（音乐）")
	}
	if params.DataType == GameActivityTypeGame && (params.ID == nil || *params.ID <= 0) {
		return fmt.Errorf("游戏动态的游戏ID不能为空")
	}
	if params.DataType == GameActivityTypeMusic && (params.Singer == "" || params.MusicName == "") {
		return fmt.Errorf("音乐动态的歌手名和歌曲名不能为空")
	}

	body := map[string]interface{}{
		"data_type": params.DataType,
	}
	if params.ID != nil {
		body["id"] = *params.ID
	}
	if params.Software != "" {
		body["software"] = params.Software
	}
	if params.Singer != "" {
		body["singer"] = params.Singer
	}
	if params.MusicName != "" {
		body["music_name"] = params.MusicName
	}

	_, err := s.client.Post(ctx, "game/activity", body)
	return err
}

type GameDeleteActivityParams struct {
	DataType int
}

// DeleteActivity 删除游戏或音乐动态。
func (s *GameService) DeleteActivity(ctx context.Context, params GameDeleteActivityParams) error {
	if params.DataType != GameActivityTypeGame && params.DataType != GameActivityTypeMusic {
		return fmt.Errorf("数据类型必须为1（游戏）或2（音乐）")
	}

	body := map[string]interface{}{
		"data_type": params.DataType,
	}

	_, err := s.client.Post(ctx, "game/delete-activity", body)
	return err
}

// 数据结构定义

// Game 游戏信息
type Game struct {
	ID          int      `json:"id"`           // 游戏ID
	Name        string   `json:"name"`         // 游戏名称
	Type        int      `json:"type"`         // 游戏类型：0游戏，1VUP，2进程
	Options     string   `json:"options"`      // 进程额外信息
	KmhookAdmin bool     `json:"kmhook_admin"` // 是否以管理员权限启动KOOK
	ProcessName []string `json:"process_name"` // 进程名称列表
	ProductName []string `json:"product_name"` // 产品名称列表
	Icon        string   `json:"icon"`         // 游戏图标URL
	StartTime   int64    `json:"start_time"`   // 动态开始时间
}

// ListGamesResponse 游戏列表响应
type ListGamesResponse struct {
	Items []Game         `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// 游戏类型常量
const (
	GameTypeAll    = "0" // 全部
	GameTypeUser   = "1" // 用户创建
	GameTypeSystem = "2" // 系统创建
)

const (
	GameActivityTypeGame  = 1
	GameActivityTypeMusic = 2
)

// 音乐软件常量
const (
	SoftwareCloudMusic = "cloudmusic" // 网易云音乐
	SoftwareQQMusic    = "qqmusic"    // QQ音乐
	SoftwareKugou      = "kugou"      // 酷狗音乐
)
