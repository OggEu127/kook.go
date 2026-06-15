package kook

// 事件类型常量
const (
	// 消息事件
	EventTypeTextMessage  = 1  // 文字消息
	EventTypeImageMessage = 2  // 图片消息
	EventTypeVideoMessage = 3  // 视频消息
	EventTypeFileMessage  = 4  // 文件消息
	EventTypeAudioMessage = 8  // 音频消息
	EventTypeKMDMessage   = 9  // KMarkdown消息
	EventTypeCardMessage  = 10 // 卡片消息

	// 系统事件
	EventTypeUserJoinedGuild    = 255 // 用户加入服务器
	EventTypeUserLeftGuild      = 254 // 用户离开服务器
	EventTypeUserUpdatedGuild   = 253 // 用户更新服务器信息
	EventTypeChannelCreated     = 252 // 频道创建
	EventTypeChannelUpdated     = 251 // 频道更新
	EventTypeChannelDeleted     = 250 // 频道删除
	EventTypeMessageDeleted     = 249 // 消息删除
	EventTypeMessageUpdated     = 248 // 消息更新
	EventTypeReactionAdded      = 247 // 添加回应
	EventTypeReactionRemoved    = 246 // 移除回应
	EventTypeGuildUpdated       = 245 // 服务器更新
	EventTypeGuildDeleted       = 244 // 服务器删除
	EventTypeGuildMemberOnline  = 243 // 成员上线
	EventTypeGuildMemberOffline = 242 // 成员下线

	// 私聊事件
	EventTypePrivateMessage         = 1   // 私聊消息
	EventTypePrivateMessageDeleted  = 249 // 私聊消息删除
	EventTypePrivateMessageUpdated  = 248 // 私聊消息更新
	EventTypePrivateReactionAdded   = 247 // 私聊添加回应
	EventTypePrivateReactionRemoved = 246 // 私聊移除回应
)

// 频道类型常量
const (
	ChannelTypeText  = 1 // 文字频道
	ChannelTypeVoice = 2 // 语音频道
)

// 消息类型常量
const (
	MessageTypeText   = 1   // 文本消息
	MessageTypeImage  = 2   // 图片消息
	MessageTypeVideo  = 3   // 视频消息
	MessageTypeFile   = 4   // 文件消息
	MessageTypeAudio  = 8   // 音频消息
	MessageTypeKMD    = 9   // KMarkdown消息
	MessageTypeCard   = 10  // 卡片消息
	MessageTypeSystem = 255 // 系统消息
)

// 角色权限常量
const (
	PermissionAdministrator       = 1 << 0  // 管理员
	PermissionManageGuild         = 1 << 1  // 管理服务器
	PermissionViewAuditLog        = 1 << 2  // 查看管理日志
	PermissionCreateInvite        = 1 << 3  // 创建服务器邀请
	PermissionManageInvites       = 1 << 4  // 管理邀请
	PermissionManageChannels      = 1 << 5  // 频道管理
	PermissionKickMembers         = 1 << 6  // 踢出用户
	PermissionBanMembers          = 1 << 7  // 封禁用户
	PermissionManageEmojis        = 1 << 8  // 管理自定义表情
	PermissionChangeNickname      = 1 << 9  // 修改服务器昵称
	PermissionManageRoles         = 1 << 10 // 管理角色权限
	PermissionViewChannel         = 1 << 11 // 查看文字、语音频道
	PermissionSendMessages        = 1 << 12 // 发布消息
	PermissionManageMessages      = 1 << 13 // 管理消息
	PermissionUploadFiles         = 1 << 14 // 上传文件
	PermissionConnectVoice        = 1 << 15 // 语音连接
	PermissionManageVoice         = 1 << 16 // 语音管理
	PermissionMentionEveryone     = 1 << 17 // 提及@全体成员
	PermissionAddReactions        = 1 << 18 // 添加反应
	PermissionFollowReactions     = 1 << 19 // 跟随添加反应
	PermissionPassiveConnectVoice = 1 << 20 // 被动连接语音频道
	PermissionUseVoiceActivity    = 1 << 21 // 仅使用按键说话
	PermissionUseFreeMic          = 1 << 22 // 使用自由麦
	PermissionSpeakVoice          = 1 << 23 // 说话
	PermissionMuteMembers         = 1 << 24 // 服务器静音
	PermissionDeafenMembers       = 1 << 25 // 服务器闭麦
	PermissionChangeOtherNickname = 1 << 26 // 修改他人昵称
	PermissionPlayMusic           = 1 << 27 // 播放伴奏
	PermissionScreenShare         = 1 << 28 // 屏幕分享
	PermissionReplyPost           = 1 << 29 // 回复帖子
	PermissionRecordAudio         = 1 << 30 // 开启录音
)

// GetEventTypeName 获取事件类型名称
func GetEventTypeName(eventType int) string {
	switch eventType {
	case EventTypeTextMessage:
		return "文字消息"
	case EventTypeImageMessage:
		return "图片消息"
	case EventTypeVideoMessage:
		return "视频消息"
	case EventTypeFileMessage:
		return "文件消息"
	case EventTypeAudioMessage:
		return "音频消息"
	case EventTypeKMDMessage:
		return "KMarkdown消息"
	case EventTypeCardMessage:
		return "卡片消息"
	case EventTypeUserJoinedGuild:
		return "用户加入服务器"
	case EventTypeUserLeftGuild:
		return "用户离开服务器"
	case EventTypeChannelCreated:
		return "频道创建"
	case EventTypeChannelUpdated:
		return "频道更新"
	case EventTypeChannelDeleted:
		return "频道删除"
	case EventTypeMessageDeleted:
		return "消息删除"
	case EventTypeMessageUpdated:
		return "消息更新"
	case EventTypeReactionAdded:
		return "添加回应"
	case EventTypeReactionRemoved:
		return "移除回应"
	default:
		return "未知事件"
	}
}
