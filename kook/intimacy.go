package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// IntimacyService 亲密度相关API服务
type IntimacyService struct {
	client *Client
}

type IntimacyViewParams struct {
	UserID string
}

// GetIntimacy 获取用户亲密度。
func (s *IntimacyService) GetIntimacy(ctx context.Context, args ...any) (*Intimacy, error) {
	params, err := compatParams("GetIntimacy", args, func(args []any) (IntimacyViewParams, bool) {
		if len(args) != 1 {
			return IntimacyViewParams{}, false
		}
		userID, ok := compatString(args[0])
		return IntimacyViewParams{UserID: userID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.UserID == "" {
		return nil, fmt.Errorf("用户ID不能为空")
	}

	query := map[string]string{
		"user_id": params.UserID,
	}

	resp, err := s.client.Get(ctx, "intimacy/index", query)
	if err != nil {
		return nil, err
	}

	var intimacy Intimacy
	if err := json.Unmarshal(resp.Data, &intimacy); err != nil {
		return nil, fmt.Errorf("解析亲密度信息失败: %w", err)
	}

	return &intimacy, nil
}

// UpdateIntimacy 更新用户亲密度。
func (s *IntimacyService) UpdateIntimacy(ctx context.Context, args ...any) error {
	params, err := compatParams("UpdateIntimacy", args, func(args []any) (UpdateIntimacyParams, bool) {
		if len(args) != 4 {
			return UpdateIntimacyParams{}, false
		}
		userID, okUser := compatString(args[0])
		score, okScore := compatInt(args[1])
		socialInfo, okSocial := compatString(args[2])
		imgID, okImg := compatString(args[3])
		return UpdateIntimacyParams{UserID: userID, Score: &score, SocialInfo: stringPointer(socialInfo), ImgID: stringPointer(imgID)}, okUser && okScore && okSocial && okImg
	})
	if err != nil {
		return err
	}
	if params.UserID == "" {
		return fmt.Errorf("用户ID不能为空")
	}

	body := map[string]interface{}{
		"user_id": params.UserID,
	}

	if params.Score != nil {
		body["score"] = *params.Score
	}
	if params.SocialInfo != nil {
		body["social_info"] = *params.SocialInfo
	}
	if params.ImgID != nil {
		body["img_id"] = *params.ImgID
	}

	_, err = s.client.Post(ctx, "intimacy/update", body)
	return err
}

// UpdateIntimacyLegacy 提供 v1.1.1 的返回值形状。
func (s *IntimacyService) UpdateIntimacyLegacy(ctx context.Context, userID string, score int, socialInfo, imgID string) (*Intimacy, error) {
	if err := s.UpdateIntimacy(ctx, userID, score, socialInfo, imgID); err != nil {
		return nil, err
	}
	return &Intimacy{UserID: userID, Score: score, SocialInfo: socialInfo, ImgID: imgID}, nil
}

// 数据结构定义

// Intimacy 亲密度信息
type Intimacy struct {
	UserID     string          `json:"user_id"`     // 用户ID
	Score      int             `json:"score"`       // 亲密度分数
	SocialInfo string          `json:"social_info"` // 社交信息
	LastRead   int64           `json:"last_read"`   // 最后阅读时间
	LastModify int64           `json:"last_modify"` // 最后修改时间
	ImgID      string          `json:"img_id"`      // 图片ID
	ImgURL     string          `json:"img_url"`     // 图片URL
	ImgList    []IntimacyImage `json:"img_list"`
}

type IntimacyImage struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

// UpdateIntimacyParams 使用指针区分未传与 0/空字符串。
type UpdateIntimacyParams struct {
	UserID     string
	Score      *int
	SocialInfo *string
	ImgID      *string
}
