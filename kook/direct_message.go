package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type DirectMessageService struct {
	client *Client
}

type DirectMessageListParams struct {
	ChatCode string
	TargetID string
	MsgID    string
	Flag     string
	Page     *int
	PageSize *int
}

func (p DirectMessageListParams) query() map[string]string {
	query := make(map[string]string)
	if p.ChatCode != "" {
		query["chat_code"] = p.ChatCode
	}
	if p.TargetID != "" {
		query["target_id"] = p.TargetID
	}
	if p.MsgID != "" {
		query["msg_id"] = p.MsgID
	}
	if p.Flag != "" {
		query["flag"] = p.Flag
	}
	if p.Page != nil {
		query["page"] = strconv.Itoa(*p.Page)
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	return query
}

func (s *DirectMessageService) List(ctx context.Context, params DirectMessageListParams) (*ListMessagesResponse, error) {
	if params.ChatCode == "" && params.TargetID == "" {
		return nil, fmt.Errorf("chat_code和target_id至少提供一个")
	}
	resp, err := s.client.Get(ctx, "direct-message/list", params.query())
	if err != nil {
		return nil, err
	}
	var result ListMessagesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析私信消息列表失败: %w", err)
	}
	return &result, nil
}

type DirectMessageViewParams struct {
	ChatCode string
	MsgID    string
}

func (s *DirectMessageService) View(ctx context.Context, params DirectMessageViewParams) (*Message, error) {
	if params.ChatCode == "" || params.MsgID == "" {
		return nil, fmt.Errorf("chat_code和msg_id不能为空")
	}
	return s.client.Message.viewMessage(ctx, "direct-message/view", map[string]string{
		"chat_code": params.ChatCode, "msg_id": params.MsgID,
	})
}

type DirectMessageCreateParams struct {
	TargetID   string
	ChatCode   string
	Content    string
	Type       *MessageType
	Quote      string
	Nonce      string
	TemplateID string
	ReplyMsgID string
}

func (s *DirectMessageService) Create(ctx context.Context, params DirectMessageCreateParams) (*MessageCreateResult, error) {
	if params.TargetID == "" && params.ChatCode == "" {
		return nil, fmt.Errorf("target_id和chat_code至少提供一个")
	}
	if params.Content == "" {
		return nil, fmt.Errorf("content不能为空")
	}
	if params.Type != nil && *params.Type == MessageTypeCard {
		if err := validateCardContent(params.Content); err != nil {
			return nil, err
		}
	}
	body := map[string]interface{}{"content": params.Content}
	if params.Type != nil {
		body["type"] = *params.Type
	}
	if params.TargetID != "" {
		body["target_id"] = params.TargetID
	}
	if params.ChatCode != "" {
		body["chat_code"] = params.ChatCode
	}
	if params.Quote != "" {
		body["quote"] = params.Quote
	}
	if params.Nonce != "" {
		body["nonce"] = params.Nonce
	}
	if params.TemplateID != "" {
		body["template_id"] = params.TemplateID
	}
	if params.ReplyMsgID != "" {
		body["reply_msg_id"] = params.ReplyMsgID
	}
	return s.client.Message.createMessage(ctx, "direct-message/create", body)
}

type DirectMessageUpdateParams struct {
	MsgID      string
	Content    string
	Quote      *string
	TemplateID *string
	ReplyMsgID *string
}

func (s *DirectMessageService) Update(ctx context.Context, params DirectMessageUpdateParams) error {
	return s.client.Message.updateMessage(ctx, "direct-message/update", params.MsgID, params.Content, params.Quote, nil, params.TemplateID, params.ReplyMsgID)
}

type DirectMessageDeleteParams struct {
	MsgID string
}

func (s *DirectMessageService) Delete(ctx context.Context, params DirectMessageDeleteParams) error {
	if params.MsgID == "" {
		return fmt.Errorf("msg_id不能为空")
	}
	_, err := s.client.Post(ctx, "direct-message/delete", map[string]interface{}{"msg_id": params.MsgID})
	return err
}

type DirectMessageReactionUsersParams struct {
	MsgID string
	Emoji string
}

func (s *DirectMessageService) ReactionUsers(ctx context.Context, params DirectMessageReactionUsersParams) ([]ReactionUser, error) {
	return s.client.Message.reactionUsers(ctx, "direct-message/reaction-list", params.MsgID, params.Emoji)
}

type DirectMessageReactionParams struct {
	MsgID string
	Emoji string
}

func (s *DirectMessageService) AddReaction(ctx context.Context, params DirectMessageReactionParams) error {
	return s.client.Message.mutateReaction(ctx, "direct-message/add-reaction", params.MsgID, params.Emoji, "")
}

type DirectMessageDeleteReactionParams struct {
	MsgID  string
	Emoji  string
	UserID string
}

func (s *DirectMessageService) DeleteReaction(ctx context.Context, params DirectMessageDeleteReactionParams) error {
	return s.client.Message.mutateReaction(ctx, "direct-message/delete-reaction", params.MsgID, params.Emoji, params.UserID)
}
