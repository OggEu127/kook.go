package kook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
)

// MessageType 是 KOOK 普通消息类型。系统事件固定使用 MessageTypeSystem，
// 具体事件名称从 extra.type 读取。
type MessageType int

const (
	MessageTypeText   MessageType = 1
	MessageTypeImage  MessageType = 2
	MessageTypeVideo  MessageType = 3
	MessageTypeFile   MessageType = 4
	MessageTypeAudio  MessageType = 8
	MessageTypeKMD    MessageType = 9
	MessageTypeCard   MessageType = 10
	MessageTypeItem   MessageType = 12
	MessageTypeSystem MessageType = 255
)

// EventChannelType 是事件来源类型。
type EventChannelType string

const (
	EventChannelTypeGroup            EventChannelType = "GROUP"
	EventChannelTypePerson           EventChannelType = "PERSON"
	EventChannelTypeBroadcast        EventChannelType = "BROADCAST"
	EventChannelTypeWebhookChallenge EventChannelType = "WEBHOOK_CHALLENGE"
)

// SystemEventType 是 extra.type 中的系统事件名称。
type SystemEventType string

const (
	SystemEventAddedReaction          SystemEventType = "added_reaction"
	SystemEventDeletedReaction        SystemEventType = "deleted_reaction"
	SystemEventUpdatedMessage         SystemEventType = "updated_message"
	SystemEventDeletedMessage         SystemEventType = "deleted_message"
	SystemEventAddedChannel           SystemEventType = "added_channel"
	SystemEventUpdatedChannel         SystemEventType = "updated_channel"
	SystemEventDeletedChannel         SystemEventType = "deleted_channel"
	SystemEventPinnedMessage          SystemEventType = "pinned_message"
	SystemEventUnpinnedMessage        SystemEventType = "unpinned_message"
	SystemEventUpdatedPrivateMessage  SystemEventType = "updated_private_message"
	SystemEventDeletedPrivateMessage  SystemEventType = "deleted_private_message"
	SystemEventPrivateAddedReaction   SystemEventType = "private_added_reaction"
	SystemEventPrivateDeletedReaction SystemEventType = "private_deleted_reaction"
	SystemEventJoinedGuild            SystemEventType = "joined_guild"
	SystemEventExitedGuild            SystemEventType = "exited_guild"
	SystemEventUpdatedGuildMember     SystemEventType = "updated_guild_member"
	SystemEventGuildMemberOnline      SystemEventType = "guild_member_online"
	SystemEventGuildMemberOffline     SystemEventType = "guild_member_offline"
	SystemEventAddedRole              SystemEventType = "added_role"
	SystemEventDeletedRole            SystemEventType = "deleted_role"
	SystemEventUpdatedRole            SystemEventType = "updated_role"
	SystemEventUpdatedGuild           SystemEventType = "updated_guild"
	SystemEventDeletedGuild           SystemEventType = "deleted_guild"
	SystemEventAddedBlockList         SystemEventType = "added_block_list"
	SystemEventDeletedBlockList       SystemEventType = "deleted_block_list"
	SystemEventAddedEmoji             SystemEventType = "added_emoji"
	SystemEventRemovedEmoji           SystemEventType = "removed_emoji"
	SystemEventUpdatedEmoji           SystemEventType = "updated_emoji"
	SystemEventJoinedChannel          SystemEventType = "joined_channel"
	SystemEventExitedChannel          SystemEventType = "exited_channel"
	SystemEventUserUpdated            SystemEventType = "user_updated"
	SystemEventSelfJoinedGuild        SystemEventType = "self_joined_guild"
	SystemEventSelfExitedGuild        SystemEventType = "self_exited_guild"
	SystemEventMessageButtonClick     SystemEventType = "message_btn_click"
)

// Event 是 WebSocket 与 Webhook 共用的原始事件信封。
// S 和 SN 来自外层信令，解码时由 SDK 写入。
type Event struct {
	S            int              `json:"s,omitempty"`
	SN           int              `json:"sn,omitempty"`
	ChannelType  EventChannelType `json:"channel_type"`
	Type         MessageType      `json:"type"`
	TargetID     string           `json:"target_id"`
	AuthorID     string           `json:"author_id"`
	Content      json.RawMessage  `json:"content"`
	Extra        json.RawMessage  `json:"extra"`
	MsgID        string           `json:"msg_id"`
	MsgTimestamp int64            `json:"msg_timestamp"`
	Nonce        string           `json:"nonce"`
	VerifyToken  string           `json:"verify_token"`
}

// DecodeContent 将 content 解码到调用者提供的目标中。
func (e *Event) DecodeContent(dst any) error {
	return decodeEventRaw("content", e.Content, dst)
}

// DecodeExtra 将 extra 解码到调用者提供的目标中。
func (e *Event) DecodeExtra(dst any) error {
	return decodeEventRaw("extra", e.Extra, dst)
}

// TextContent 返回字符串消息内容。对象型内容（例如道具消息）会返回错误。
func (e *Event) TextContent() (string, error) {
	var content string
	if err := e.DecodeContent(&content); err != nil {
		return "", err
	}
	return content, nil
}

func decodeEventRaw(name string, raw json.RawMessage, dst any) error {
	if dst == nil {
		return fmt.Errorf("%s解码目标不能为空", name)
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("%s为空", name)
	}
	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("解析事件%s失败: %w", name, err)
	}
	return nil
}

// MessageEvent 是普通消息事件。
type MessageEvent struct {
	*Event
}

// SystemEvent 是系统事件。Body 保持原始 JSON，避免官方扩展字段导致 SDK 丢失数据。
type SystemEvent struct {
	*Event
	Type SystemEventType
	Body json.RawMessage
}

// DecodeBody 将系统事件 extra.body 解码到调用者自己的结构中。
func (e *SystemEvent) DecodeBody(dst any) error {
	return decodeEventRaw("extra.body", e.Body, dst)
}

// AsSystemEvent 从 type=255 的原始事件中提取 extra.type 和 extra.body。
func (e *Event) AsSystemEvent() (*SystemEvent, error) {
	if e.Type != MessageTypeSystem {
		return nil, fmt.Errorf("消息类型%d不是系统事件", e.Type)
	}
	var extra struct {
		Type SystemEventType `json:"type"`
		Body json.RawMessage `json:"body"`
	}
	if err := e.DecodeExtra(&extra); err != nil {
		return nil, err
	}
	if extra.Type == "" {
		return nil, fmt.Errorf("系统事件extra.type为空")
	}
	return &SystemEvent{Event: e, Type: extra.Type, Body: extra.Body}, nil
}

type MessageEventHandler func(*MessageEvent)
type SystemEventHandler func(*SystemEvent)
type RawEventHandler func(*Event)

type eventDispatcher struct {
	mu              sync.RWMutex
	messageHandlers map[MessageType][]MessageEventHandler
	systemHandlers  map[SystemEventType][]SystemEventHandler
	anyHandlers     []RawEventHandler
}

func newEventDispatcher() *eventDispatcher {
	return &eventDispatcher{
		messageHandlers: make(map[MessageType][]MessageEventHandler),
		systemHandlers:  make(map[SystemEventType][]SystemEventHandler),
	}
}

func (d *eventDispatcher) onMessage(eventType MessageType, handler MessageEventHandler) {
	if handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.messageHandlers[eventType] = append(d.messageHandlers[eventType], handler)
}

func (d *eventDispatcher) onSystemEvent(eventType SystemEventType, handler SystemEventHandler) {
	if handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.systemHandlers[eventType] = append(d.systemHandlers[eventType], handler)
}

func (d *eventDispatcher) onAnyEvent(handler RawEventHandler) {
	if handler == nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.anyHandlers = append(d.anyHandlers, handler)
}

func (d *eventDispatcher) dispatch(event *Event, onPanic func(any)) error {
	d.mu.RLock()
	anyHandlers := append([]RawEventHandler(nil), d.anyHandlers...)
	d.mu.RUnlock()
	for _, handler := range anyHandlers {
		invokeEventHandler(func() { handler(event) }, onPanic)
	}

	if event.Type == MessageTypeSystem {
		systemEvent, err := event.AsSystemEvent()
		if err != nil {
			return err
		}
		d.mu.RLock()
		handlers := append([]SystemEventHandler(nil), d.systemHandlers[systemEvent.Type]...)
		d.mu.RUnlock()
		for _, handler := range handlers {
			invokeEventHandler(func() { handler(systemEvent) }, onPanic)
		}
		return nil
	}

	d.mu.RLock()
	handlers := append([]MessageEventHandler(nil), d.messageHandlers[event.Type]...)
	d.mu.RUnlock()
	messageEvent := &MessageEvent{Event: event}
	for _, handler := range handlers {
		invokeEventHandler(func() { handler(messageEvent) }, onPanic)
	}
	return nil
}

func invokeEventHandler(handler func(), onPanic func(any)) {
	defer func() {
		if recovered := recover(); recovered != nil && onPanic != nil {
			onPanic(recovered)
		}
	}()
	handler()
}

// 角色权限常量。
const (
	PermissionAdministrator       = 1 << 0
	PermissionManageGuild         = 1 << 1
	PermissionViewAuditLog        = 1 << 2
	PermissionCreateInvite        = 1 << 3
	PermissionManageInvites       = 1 << 4
	PermissionManageChannels      = 1 << 5
	PermissionKickMembers         = 1 << 6
	PermissionBanMembers          = 1 << 7
	PermissionManageEmojis        = 1 << 8
	PermissionChangeNickname      = 1 << 9
	PermissionManageRoles         = 1 << 10
	PermissionViewChannel         = 1 << 11
	PermissionSendMessages        = 1 << 12
	PermissionManageMessages      = 1 << 13
	PermissionUploadFiles         = 1 << 14
	PermissionConnectVoice        = 1 << 15
	PermissionManageVoice         = 1 << 16
	PermissionMentionEveryone     = 1 << 17
	PermissionAddReactions        = 1 << 18
	PermissionFollowReactions     = 1 << 19
	PermissionPassiveConnectVoice = 1 << 20
	PermissionUseVoiceActivity    = 1 << 21
	PermissionUseFreeMic          = 1 << 22
	PermissionSpeakVoice          = 1 << 23
	PermissionMuteMembers         = 1 << 24
	PermissionDeafenMembers       = 1 << 25
	PermissionChangeOtherNickname = 1 << 26
	PermissionPlayMusic           = 1 << 27
	PermissionScreenShare         = 1 << 28
	PermissionReplyPost           = 1 << 29
	PermissionRecordAudio         = 1 << 30
)

// GetEventTypeName 返回普通消息类型名称。
func GetEventTypeName(eventType MessageType) string {
	switch eventType {
	case MessageTypeText:
		return "文字消息"
	case MessageTypeImage:
		return "图片消息"
	case MessageTypeVideo:
		return "视频消息"
	case MessageTypeFile:
		return "文件消息"
	case MessageTypeAudio:
		return "音频消息"
	case MessageTypeKMD:
		return "KMarkdown消息"
	case MessageTypeCard:
		return "卡片消息"
	case MessageTypeItem:
		return "道具消息"
	case MessageTypeSystem:
		return "系统事件"
	default:
		return "未知事件"
	}
}
