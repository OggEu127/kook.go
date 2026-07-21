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

func (s *MessageService) GetMessageList(ctx context.Context, args ...any) (*ListMessagesResponse, error) {
	params, err := compatParams("GetMessageList", args, func(args []any) (MessageListParams, bool) {
		if len(args) != 2 {
			return MessageListParams{}, false
		}
		targetID, okTarget := compatString(args[0])
		legacy, okParams := args[1].(GetMessageListParams)
		if legacy.Type != "" && legacy.Type != "channel" && legacy.Type != "guild" {
			return MessageListParams{}, false
		}
		return MessageListParams{
			TargetID: targetID,
			MsgID:    legacy.MsgID,
			Pin:      optionalPositiveInt(legacy.Pin),
			Flag:     legacy.Flag,
			PageSize: optionalPositiveInt(legacy.PageSize),
		}, okTarget && okParams
	})
	if err != nil {
		return nil, err
	}
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

func (s *MessageService) GetMessage(ctx context.Context, args ...any) (*Message, error) {
	params, err := compatParams("GetMessage", args, func(args []any) (MessageViewParams, bool) {
		if len(args) != 1 {
			return MessageViewParams{}, false
		}
		msgID, ok := compatString(args[0])
		return MessageViewParams{MsgID: msgID}, ok
	})
	if err != nil {
		return nil, err
	}
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

func (s *MessageService) DeleteMessage(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteMessage", args, func(args []any) (MessageDeleteParams, bool) {
		if len(args) != 1 {
			return MessageDeleteParams{}, false
		}
		msgID, ok := compatString(args[0])
		return MessageDeleteParams{MsgID: msgID}, ok
	})
	if err != nil {
		return err
	}
	if params.MsgID == "" {
		return fmt.Errorf("msg_id不能为空")
	}
	_, err = s.client.Post(ctx, "message/delete", map[string]interface{}{"msg_id": params.MsgID})
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

func (s *MessageService) AddReaction(ctx context.Context, args ...any) error {
	params, err := compatParams("AddReaction", args, func(args []any) (MessageReactionParams, bool) {
		if len(args) != 2 {
			return MessageReactionParams{}, false
		}
		msgID, okMsg := compatString(args[0])
		emoji, okEmoji := compatString(args[1])
		return MessageReactionParams{MsgID: msgID, Emoji: emoji}, okMsg && okEmoji
	})
	if err != nil {
		return err
	}
	return s.mutateReaction(ctx, "message/add-reaction", params.MsgID, params.Emoji, "")
}

type MessageDeleteReactionParams struct {
	MsgID  string
	Emoji  string
	UserID string
}

func (s *MessageService) DeleteReaction(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteReaction", args, func(args []any) (MessageDeleteReactionParams, bool) {
		if len(args) != 3 {
			return MessageDeleteReactionParams{}, false
		}
		msgID, okMsg := compatString(args[0])
		emoji, okEmoji := compatString(args[1])
		userID, okUser := compatString(args[2])
		return MessageDeleteReactionParams{MsgID: msgID, Emoji: emoji, UserID: userID}, okMsg && okEmoji && okUser
	})
	if err != nil {
		return err
	}
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

func (s *MessageService) PinMessage(ctx context.Context, args ...any) error {
	params, err := compatParams("PinMessage", args, func(args []any) (MessagePinParams, bool) {
		if len(args) != 2 {
			return MessagePinParams{}, false
		}
		msgID, okMsg := compatString(args[0])
		targetID, okTarget := compatString(args[1])
		return MessagePinParams{MsgID: msgID, TargetID: targetID}, okMsg && okTarget
	})
	if err != nil {
		return err
	}
	return s.pinMessage(ctx, "message/pin", params.MsgID, params.TargetID)
}

type MessageUnpinParams struct {
	MsgID    string
	TargetID string
}

func (s *MessageService) UnpinMessage(ctx context.Context, args ...any) error {
	params, err := compatParams("UnpinMessage", args, func(args []any) (MessageUnpinParams, bool) {
		if len(args) != 2 {
			return MessageUnpinParams{}, false
		}
		msgID, okMsg := compatString(args[0])
		targetID, okTarget := compatString(args[1])
		return MessageUnpinParams{MsgID: msgID, TargetID: targetID}, okMsg && okTarget
	})
	if err != nil {
		return err
	}
	return s.pinMessage(ctx, "message/unpin", params.MsgID, params.TargetID)
}

// SendMessageParams 保留 v1.1.1 的统一频道/私信消息参数。
type SendMessageParams struct {
	Type         string `json:"type,omitempty"`
	TargetID     string `json:"target_id,omitempty"`
	ChatCode     string `json:"chat_code,omitempty"`
	Content      string `json:"content"`
	MsgType      int    `json:"msg_type,omitempty"`
	Quote        string `json:"quote,omitempty"`
	Nonce        string `json:"nonce,omitempty"`
	TempTargetID string `json:"temp_target_id,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	ReplyMsgID   string `json:"reply_msg_id,omitempty"`
}

// GetMessageListParams 保留 v1.1.1 的消息列表参数。
type GetMessageListParams struct {
	Type     string `json:"type,omitempty"`
	ChatCode string `json:"chat_code,omitempty"`
	MsgID    string `json:"msg_id,omitempty"`
	Pin      int    `json:"pin,omitempty"`
	Flag     string `json:"flag,omitempty"`
	PageSize int    `json:"page_size,omitempty"`
}

// SendMessage 保留 v1.1.1 的统一发送入口，并只映射当前官方频道/私信端点。
func (s *MessageService) SendMessage(ctx context.Context, params SendMessageParams) (*Message, error) {
	msgType := MessageType(params.MsgType)
	if msgType <= 0 {
		msgType = MessageTypeKMD
	}
	var result *MessageCreateResult
	var err error
	switch params.Type {
	case "private", "direct", "dm":
		result, err = s.client.DirectMessage.Create(ctx, DirectMessageCreateParams{
			TargetID: params.TargetID, ChatCode: params.ChatCode, Content: params.Content,
			Type: &msgType, Quote: params.Quote, Nonce: params.Nonce,
			TemplateID: params.TemplateID, ReplyMsgID: params.ReplyMsgID,
		})
	case "", "channel", "guild":
		result, err = s.Create(ctx, MessageCreateParams{
			TargetID: params.TargetID, Content: params.Content, Type: &msgType,
			Quote: params.Quote, Nonce: params.Nonce, TempTargetID: params.TempTargetID,
			TemplateID: params.TemplateID, ReplyMsgID: params.ReplyMsgID,
		})
	default:
		return nil, NewValidationErrorWithValue("type", "消息作用域必须为channel或private", params.Type)
	}
	if err != nil {
		return nil, err
	}
	return &Message{ID: result.MsgID, Type: msgType, Content: params.Content, CreateAt: result.MsgTimestamp}, nil
}

func (s *MessageService) SendCardMessage(ctx context.Context, params SendMessageParams) (*Message, error) {
	params.MsgType = int(MessageTypeCard)
	return s.SendMessage(ctx, params)
}

func (s *MessageService) GetDirectMessage(ctx context.Context, chatCode, msgID string) (*Message, error) {
	return s.client.DirectMessage.View(ctx, DirectMessageViewParams{ChatCode: chatCode, MsgID: msgID})
}

func (s *MessageService) UpdateMessage(ctx context.Context, msgID, content, quote, tempTargetID string) (*Message, error) {
	if err := s.Update(ctx, UpdateMessageParams{
		MsgID: msgID, Content: content, Quote: stringPointer(quote), TempTargetID: stringPointer(tempTargetID),
	}); err != nil {
		return nil, err
	}
	return s.GetMessage(ctx, msgID)
}

func (s *MessageService) UpdateDirectMessage(ctx context.Context, msgID, content, quote string) error {
	return s.client.DirectMessage.Update(ctx, DirectMessageUpdateParams{MsgID: msgID, Content: content, Quote: stringPointer(quote)})
}

func (s *MessageService) DeleteDirectMessage(ctx context.Context, msgID string) error {
	return s.client.DirectMessage.Delete(ctx, DirectMessageDeleteParams{MsgID: msgID})
}

func (s *MessageService) AddDirectReaction(ctx context.Context, msgID, emoji string) error {
	return s.client.DirectMessage.AddReaction(ctx, DirectMessageReactionParams{MsgID: msgID, Emoji: emoji})
}

func (s *MessageService) DeleteDirectReaction(ctx context.Context, msgID, emoji, userID string) error {
	return s.client.DirectMessage.DeleteReaction(ctx, DirectMessageDeleteReactionParams{MsgID: msgID, Emoji: emoji, UserID: userID})
}

func (s *MessageService) GetReactionUserList(ctx context.Context, msgID, emoji string) ([]User, error) {
	users, err := s.ReactionUsers(ctx, MessageReactionUsersParams{MsgID: msgID, Emoji: emoji})
	if err != nil {
		return nil, err
	}
	result := make([]User, len(users))
	for i := range users {
		result[i] = users[i].User
	}
	return result, nil
}

func (s *MessageService) GetDirectReactionUserList(ctx context.Context, msgID, emoji string) ([]User, error) {
	users, err := s.client.DirectMessage.ReactionUsers(ctx, DirectMessageReactionUsersParams{MsgID: msgID, Emoji: emoji})
	if err != nil {
		return nil, err
	}
	result := make([]User, len(users))
	for i := range users {
		result[i] = users[i].User
	}
	return result, nil
}

// CheckCardResponse 保留 v1.1.1 返回类型。
type CheckCardResponse struct {
	Mention struct {
		Mentions     []string      `json:"mentions"`
		MentionRoles []string      `json:"mentionRoles"`
		MentionAll   bool          `json:"mentionAll"`
		MentionHere  bool          `json:"mentionHere"`
		MentionPart  []interface{} `json:"mentionPart"`
		NavChannels  []interface{} `json:"navChannels"`
		ChannelPart  []interface{} `json:"channelPart"`
		GuildEmojis  []interface{} `json:"guildEmojis"`
	} `json:"mention"`
	Content string `json:"content"`
}

func (s *MessageService) CheckCard(context.Context, string) (*CheckCardResponse, error) {
	return nil, unsupportedEndpoint("message/check-card")
}

func (s *MessageService) SendPipeMessage(context.Context, SendMessageParams) (*Message, error) {
	return nil, unsupportedEndpoint("message/send-pipemsg without access_token")
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
