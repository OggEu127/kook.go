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
func (s *TemplateService) CreateTemplate(ctx context.Context, params CreateTemplateParams) (*Template, error) {
	if params.Title == "" {
		return nil, fmt.Errorf("模板标题不能为空")
	}
	if params.Content == "" {
		return nil, fmt.Errorf("模板内容不能为空")
	}

	resp, err := s.client.Post(ctx, "template/create", params.toMap())
	if err != nil {
		return nil, err
	}

	return parseTemplateModel(resp.Data)
}

// UpdateTemplate 更新模板
func (s *TemplateService) UpdateTemplate(ctx context.Context, params UpdateTemplateParams) (*Template, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("模板ID不能为空")
	}

	resp, err := s.client.Post(ctx, "template/update", params.toMap())
	if err != nil {
		return nil, err
	}

	return parseTemplateModel(resp.Data)
}

type DeleteTemplateParams struct {
	ID string
}

// DeleteTemplate 删除模板。
func (s *TemplateService) DeleteTemplate(ctx context.Context, params DeleteTemplateParams) error {
	if params.ID == "" {
		return fmt.Errorf("模板ID不能为空")
	}

	_, err := s.client.Post(ctx, "template/delete", map[string]interface{}{
		"id": params.ID,
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

// CreateTemplateParams 创建模板参数。
type CreateTemplateParams struct {
	Title       string
	Content     string
	Type        *int
	MsgType     *int
	TestData    *string
	TestChannel *string
}

func (p CreateTemplateParams) toMap() map[string]interface{} {
	params := map[string]interface{}{"title": p.Title, "content": p.Content}
	setTemplateOptionalFields(params, p.Type, p.MsgType, p.TestData, p.TestChannel)
	return params
}

// UpdateTemplateParams 更新模板参数。指针字段允许显式发送零值或空字符串。
type UpdateTemplateParams struct {
	ID          string
	Title       *string
	Content     *string
	Type        *int
	MsgType     *int
	TestData    *string
	TestChannel *string
}

func (p UpdateTemplateParams) toMap() map[string]interface{} {
	params := map[string]interface{}{"id": p.ID}
	if p.Title != nil {
		params["title"] = *p.Title
	}
	if p.Content != nil {
		params["content"] = *p.Content
	}
	setTemplateOptionalFields(params, p.Type, p.MsgType, p.TestData, p.TestChannel)
	return params
}

func setTemplateOptionalFields(params map[string]interface{}, templateType, msgType *int, testData, testChannel *string) {
	if templateType != nil {
		params["type"] = *templateType
	}
	if msgType != nil {
		params["msgtype"] = *msgType
	}
	if testData != nil {
		params["test_data"] = *testData
	}
	if testChannel != nil {
		params["test_channel"] = *testChannel
	}
}

// ListTemplatesResponse 模板列表响应
type ListTemplatesResponse struct {
	Items []Template     `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}
