package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// MessageService 频道消息相关 API。
type MessageService struct {
	client *Client
}

type MessageListParams struct {
	TargetID string
	MsgID    string
	Pin      *int
	Flag     string
	PageSize *int
}

func (p MessageListParams) query() map[string]string {
	query := map[string]string{"target_id": p.TargetID}
	if p.MsgID != "" {
		query["msg_id"] = p.MsgID
	}
	if p.Pin != nil {
		query["pin"] = strconv.Itoa(*p.Pin)
	}
	if p.Flag != "" {
		query["flag"] = p.Flag
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	return query
}

func (s *MessageService) GetMessageList(ctx context.Context, params MessageListParams) (*ListMessagesResponse, error) {
	if params.TargetID == "" {
		return nil, fmt.Errorf("target_id不能为空")
	}
	resp, err := s.client.Get(ctx, "message/list", params.query())
	if err != nil {
		return nil, err
	}
	var result ListMessagesResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析消息列表失败: %w", err)
	}
	return &result, nil
}

type MessageViewParams struct {
	MsgID string
}

func (s *MessageService) GetMessage(ctx context.Context, params MessageViewParams) (*Message, error) {
	if params.MsgID == "" {
		return nil, fmt.Errorf("msg_id不能为空")
	}
	return s.viewMessage(ctx, "message/view", map[string]string{"msg_id": params.MsgID})
}

type MessageCreateParams struct {
	TargetID     string
	Content      string
	Type         *MessageType
	Quote        string
	Nonce        string
	TempTargetID string
	TemplateID   string
	ReplyMsgID   string
}

func (p MessageCreateParams) body() (map[string]interface{}, error) {
	if p.TargetID == "" || p.Content == "" {
		return nil, fmt.Errorf("target_id和content不能为空")
	}
	if p.Type != nil && *p.Type == MessageTypeCard {
		if err := validateCardContent(p.Content); err != nil {
			return nil, err
		}
	}
	body := map[string]interface{}{
		"target_id": p.TargetID,
		"content":   p.Content,
	}
	if p.Type != nil {
		body["type"] = *p.Type
	}
	if p.Quote != "" {
		body["quote"] = p.Quote
	}
	if p.Nonce != "" {
		body["nonce"] = p.Nonce
	}
	if p.TempTargetID != "" {
		body["temp_target_id"] = p.TempTargetID
	}
	if p.TemplateID != "" {
		body["template_id"] = p.TemplateID
	}
	if p.ReplyMsgID != "" {
		body["reply_msg_id"] = p.ReplyMsgID
	}
	return body, nil
}

func (s *MessageService) Create(ctx context.Context, params MessageCreateParams) (*MessageCreateResult, error) {
	body, err := params.body()
	if err != nil {
		return nil, err
	}
	return s.createMessage(ctx, "message/create", body)
}

type UpdateMessageParams struct {
	MsgID        string
	Content      string
	Quote        *string
	TempTargetID *string
	TemplateID   *string
	ReplyMsgID   *string
}

func (s *MessageService) Update(ctx context.Context, params UpdateMessageParams) error {
	return s.updateMessage(ctx, "message/update", params.MsgID, params.Content, params.Quote, params.TempTargetID, params.TemplateID, params.ReplyMsgID)
}

type MessageDeleteParams struct {
	MsgID string
}

func (s *MessageService) DeleteMessage(ctx context.Context, params MessageDeleteParams) error {
	if params.MsgID == "" {
		return fmt.Errorf("msg_id不能为空")
	}
	_, err := s.client.Post(ctx, "message/delete", map[string]interface{}{"msg_id": params.MsgID})
	return err
}

type MessageReactionUsersParams struct {
	MsgID string
	Emoji string
}

func (s *MessageService) ReactionUsers(ctx context.Context, params MessageReactionUsersParams) ([]ReactionUser, error) {
	if params.Emoji == "" {
		return nil, fmt.Errorf("emoji不能为空")
	}
	return s.reactionUsers(ctx, "message/reaction-list", params.MsgID, params.Emoji)
}

type MessageReactionParams struct {
	MsgID string
	Emoji string
}

func (s *MessageService) AddReaction(ctx context.Context, params MessageReactionParams) error {
	return s.mutateReaction(ctx, "message/add-reaction", params.MsgID, params.Emoji, "")
}

type MessageDeleteReactionParams struct {
	MsgID  string
	Emoji  string
	UserID string
}

func (s *MessageService) DeleteReaction(ctx context.Context, params MessageDeleteReactionParams) error {
	return s.mutateReaction(ctx, "message/delete-reaction", params.MsgID, params.Emoji, params.UserID)
}

type SendPipeMessageParams struct {
	AccessToken   string
	TargetID      string
	Type          *MessageType
	Content       string
	TemplateInput map[string]interface{}
}

func (s *MessageService) SendPipe(ctx context.Context, params SendPipeMessageParams) (*MessageCreateResult, error) {
	if params.AccessToken == "" {
		return nil, fmt.Errorf("access_token不能为空")
	}
	if params.TemplateInput == nil && params.Content == "" {
		return nil, fmt.Errorf("content不能为空")
	}
	if params.Type != nil && *params.Type == MessageTypeCard && params.TemplateInput == nil {
		if err := validateCardContent(params.Content); err != nil {
			return nil, err
		}
	}
	query := map[string]string{
		"access_token": params.AccessToken,
	}
	if params.Type != nil {
		query["type"] = strconv.Itoa(int(*params.Type))
	}
	if params.TargetID != "" {
		query["target_id"] = params.TargetID
	}
	body := params.TemplateInput
	if body == nil {
		body = map[string]interface{}{"content": params.Content}
	}
	resp, err := s.client.doRequest(ctx, "POST", "message/send-pipemsg", body, query)
	if err != nil {
		return nil, err
	}
	return decodeMessageCreateResult(resp.Data)
}

type MessagePinParams struct {
	MsgID    string
	TargetID string
}

func (s *MessageService) PinMessage(ctx context.Context, params MessagePinParams) error {
	return s.pinMessage(ctx, "message/pin", params.MsgID, params.TargetID)
}

type MessageUnpinParams struct {
	MsgID    string
	TargetID string
}

func (s *MessageService) UnpinMessage(ctx context.Context, params MessageUnpinParams) error {
	return s.pinMessage(ctx, "message/unpin", params.MsgID, params.TargetID)
}

func (s *MessageService) pinMessage(ctx context.Context, endpoint, msgID, targetID string) error {
	if msgID == "" || targetID == "" {
		return fmt.Errorf("msg_id和target_id不能为空")
	}
	_, err := s.client.Post(ctx, endpoint, map[string]interface{}{
		"msg_id": msgID, "target_id": targetID,
	})
	return err
}

func (s *MessageService) createMessage(ctx context.Context, endpoint string, body map[string]interface{}) (*MessageCreateResult, error) {
	resp, err := s.client.Post(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	return decodeMessageCreateResult(resp.Data)
}

func decodeMessageCreateResult(data json.RawMessage) (*MessageCreateResult, error) {
	var result MessageCreateResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析消息创建结果失败: %w", err)
	}
	return &result, nil
}

func (s *MessageService) viewMessage(ctx context.Context, endpoint string, query map[string]string) (*Message, error) {
	resp, err := s.client.Get(ctx, endpoint, query)
	if err != nil {
		return nil, err
	}
	var message Message
	if err := json.Unmarshal(resp.Data, &message); err != nil {
		return nil, fmt.Errorf("解析消息失败: %w", err)
	}
	return &message, nil
}

func (s *MessageService) updateMessage(ctx context.Context, endpoint, msgID, content string, quote, tempTargetID, templateID, replyMsgID *string) error {
	if msgID == "" || content == "" {
		return fmt.Errorf("msg_id和content不能为空")
	}
	body := map[string]interface{}{"msg_id": msgID, "content": content}
	if quote != nil {
		body["quote"] = *quote
	}
	if tempTargetID != nil && endpoint == "message/update" {
		body["temp_target_id"] = *tempTargetID
	}
	if templateID != nil {
		body["template_id"] = *templateID
	}
	if replyMsgID != nil {
		body["reply_msg_id"] = *replyMsgID
	}
	_, err := s.client.Post(ctx, endpoint, body)
	return err
}

func (s *MessageService) reactionUsers(ctx context.Context, endpoint, msgID, emoji string) ([]ReactionUser, error) {
	if msgID == "" {
		return nil, fmt.Errorf("msg_id不能为空")
	}
	query := map[string]string{"msg_id": msgID}
	if emoji != "" {
		query["emoji"] = emoji
	}
	resp, err := s.client.Get(ctx, endpoint, query)
	if err != nil {
		return nil, err
	}
	var users []ReactionUser
	if err := json.Unmarshal(resp.Data, &users); err != nil {
		return nil, fmt.Errorf("解析回应用户失败: %w", err)
	}
	return users, nil
}

func (s *MessageService) mutateReaction(ctx context.Context, endpoint, msgID, emoji, userID string) error {
	if msgID == "" || emoji == "" {
		return fmt.Errorf("msg_id和emoji不能为空")
	}
	body := map[string]interface{}{"msg_id": msgID, "emoji": emoji}
	if userID != "" {
		body["user_id"] = userID
	}
	_, err := s.client.Post(ctx, endpoint, body)
	return err
}

func validateCardContent(content string) error {
	var cards []json.RawMessage
	if err := json.Unmarshal([]byte(content), &cards); err != nil {
		return fmt.Errorf("卡片消息content必须是JSON数组字符串: %w", err)
	}
	if len(cards) == 0 {
		return fmt.Errorf("卡片消息至少包含一个card")
	}
	return nil
}

type MessageCreateResult struct {
	MsgID                 string   `json:"msg_id"`
	MsgTimestamp          int64    `json:"msg_timestamp"`
	Nonce                 string   `json:"nonce"`
	NotPermissionsMention []string `json:"not_permissions_mention"`
}

type ReactionUser struct {
	User
	ReactionTime int64 `json:"reaction_time"`
}

type ListMessagesResponse struct {
	Items []Message `json:"items"`
}
