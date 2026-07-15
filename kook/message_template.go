package kook

import (
	"context"
	"encoding/json"
	"fmt"
)

// TemplateService 消息模板相关API服务
type TemplateService struct {
	client *Client
}

// GetTemplateList 获取模板列表
func (s *TemplateService) GetTemplateList(ctx context.Context) (*ListTemplatesResponse, error) {
	resp, err := s.client.Get(ctx, "template/list", nil)
	if err != nil {
		return nil, err
	}

	var result ListTemplatesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析模板列表失败: %w", err)
	}

	return &result, nil
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(ctx context.Context, params TemplateParams) (*Template, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("模板标题不能为空")
	}
	if params.Content == "" {
		return nil, fmt.Errorf("模板内容不能为空")
	}

	resp, err := s.client.Post(ctx, "template/create", params.toMap(false))
	if err != nil {
		return nil, err
	}

	return parseTemplateModel(resp.Data)
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(ctx context.Context, params TemplateParams) (*Template, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("模板ID不能为空")
	}

	resp, err := s.client.Post(ctx, "template/update", params.toMap(true))
	if err != nil {
		return nil, err
	}

	return parseTemplateModel(resp.Data)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("模板ID不能为空")
	}

	_, err := s.client.Post(ctx, "template/delete", map[string]interface{}{
		"id": id,
	})
	return err
}

func parseTemplateModel(data json.RawMessage) (*Template, error) {
	var wrapper struct {
		Model Template `json:"model"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("解析模板信息失败: %w", err)
	}
	return &wrapper.Model, nil
}

// Template 消息模板
type Template struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Type        int    `json:"type"`
	MsgType     int    `json:"msgtype"`
	Status      int    `json:"status"`
	TestData    string `json:"test_data"`
	TestChannel string `json:"test_channel"`
	Content     string `json:"content"`
}

// TemplateParams 创建或更新模板参数
type TemplateParams struct {
	ID          string `json:"id,omitempty"`
	Title       string `json:"title,omitempty"`
	Type        int    `json:"type,omitempty"`
	MsgType     int    `json:"msgtype,omitempty"`
	TestData    string `json:"test_data,omitempty"`
	TestChannel string `json:"test_channel,omitempty"`
	Content     string `json:"content,omitempty"`
}

func (p TemplateParams) toMap(includeID bool) map[string]interface{} {
	params := make(map[string]interface{})
	if includeID {
		params["id"] = p.ID
	}
	if p.Title != "" {
		params["title"] = p.Title
	}
	if p.Content != "" {
		params["content"] = p.Content
	}
	if p.TestData != "" {
		params["test_data"] = p.TestData
	}
	if p.TestChannel != "" {
		params["test_channel"] = p.TestChannel
	}
	if p.Type != 0 {
		params["type"] = p.Type
	}
	if p.MsgType != 0 {
		params["msgtype"] = p.MsgType
	}
	return params
}

// ListTemplatesResponse 模板列表响应
type ListTemplatesResponse struct {
	Items []Template     `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}
