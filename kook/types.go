package kook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// BoolInt 兼容官方接口中 boolean、0/1 两种表示。
type BoolInt bool

func (b *BoolInt) UnmarshalJSON(data []byte) error {
	switch string(bytes.TrimSpace(data)) {
	case "true", "1":
		*b = true
		return nil
	case "false", "0", "null":
		*b = false
		return nil
	default:
		return fmt.Errorf("无法将%s解析为布尔值", data)
	}
}

// SortFields 兼容分页响应中 sort 返回对象或空数组的两种形式。
type SortFields map[string]int

func (s *SortFields) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*s = nil
		return nil
	}
	if trimmed[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(trimmed, &values); err != nil {
			return err
		}
		*s = SortFields{}
		return nil
	}
	var fields map[string]int
	if err := json.Unmarshal(trimmed, &fields); err != nil {
		return err
	}
	*s = fields
	return nil
}

func decodeStringOrNumber(data json.RawMessage) (string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return "", nil
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return "", err
		}
		return value, nil
	}
	var number json.Number
	if err := json.Unmarshal(trimmed, &number); err != nil {
		return "", err
	}
	return number.String(), nil
}

func decodeBoolOrInt(data json.RawMessage) (bool, error) {
	var value BoolInt
	if err := value.UnmarshalJSON(data); err != nil {
		return false, err
	}
	return bool(value), nil
}

func decodeIntOrString(data json.RawMessage) (int, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return 0, nil
	}
	if trimmed[0] == '"' {
		var value string
		if err := json.Unmarshal(trimmed, &value); err != nil {
			return 0, err
		}
		return strconv.Atoi(value)
	}
	var value int
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return 0, err
	}
	return value, nil
}

func decodeStringSlice(data json.RawMessage) ([]string, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return nil, err
	}
	result := make([]string, 0, len(values))
	for _, value := range values {
		decoded, err := decodeStringOrNumber(value)
		if err != nil {
			return nil, err
		}
		result = append(result, decoded)
	}
	return result, nil
}

func decodeIntSlice(data json.RawMessage) ([]int, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}
	var values []json.RawMessage
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return nil, err
	}
	result := make([]int, 0, len(values))
	for _, value := range values {
		decoded, err := decodeIntOrString(value)
		if err != nil {
			return nil, err
		}
		result = append(result, decoded)
	}
	return result, nil
}

// User 用户信息
type User struct {
	ID               string         `json:"id"`
	Username         string         `json:"username"`
	IdentifyNum      string         `json:"identify_num"`
	Online           bool           `json:"online"`
	OS               string         `json:"os"`
	Bot              bool           `json:"bot"`
	BotStatus        BoolInt        `json:"bot_status"`
	Status           int            `json:"status"`
	Avatar           string         `json:"avatar"`
	VipAvatar        string         `json:"vip_avatar"`
	Banner           string         `json:"banner"`
	Nickname         string         `json:"nickname"`
	Roles            []int          `json:"roles"`
	IsVip            bool           `json:"is_vip"`
	VipAmp           bool           `json:"vip_amp"`
	InvitedCount     int            `json:"invited_count"`
	TagInfo          TagInfo        `json:"tag_info"`
	MobileVerified   bool           `json:"mobile_verified"`
	IsSys            bool           `json:"is_sys"`
	ClientID         string         `json:"client_id"`
	Verified         bool           `json:"verified"`
	MobilePrefix     string         `json:"mobile_prefix"`
	Mobile           string         `json:"mobile"`
	JoinedAt         int64          `json:"joined_at"`
	ActiveTime       int64          `json:"active_time"`
	BoostStartAt     *int64         `json:"boost_start_at"`
	IsAIReduceNoise  bool           `json:"is_ai_reduce_noise"`
	IsPersonalCardBG bool           `json:"is_personal_card_bg"`
	LiveInfo         UserLiveInfo   `json:"live_info"`
	DecorationsIDMap map[string]int `json:"decorations_id_map"`
	Game             *Game          `json:"game"`
}

// UnmarshalJSON 兼容用户对象中的数字 ID、0/1 布尔值和字符串角色 ID。
func (u *User) UnmarshalJSON(data []byte) error {
	type userAlias User
	value := struct {
		*userAlias
		ID               json.RawMessage `json:"id"`
		Online           json.RawMessage `json:"online"`
		Bot              json.RawMessage `json:"bot"`
		IsVip            json.RawMessage `json:"is_vip"`
		VipAmp           json.RawMessage `json:"vip_amp"`
		MobileVerified   json.RawMessage `json:"mobile_verified"`
		IsSys            json.RawMessage `json:"is_sys"`
		Verified         json.RawMessage `json:"verified"`
		IsAIReduceNoise  json.RawMessage `json:"is_ai_reduce_noise"`
		IsPersonalCardBG json.RawMessage `json:"is_personal_card_bg"`
		Roles            json.RawMessage `json:"roles"`
	}{userAlias: (*userAlias)(u)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.ID) > 0 {
		id, err := decodeStringOrNumber(value.ID)
		if err != nil {
			return fmt.Errorf("解析user.id失败: %w", err)
		}
		u.ID = id
	}
	boolFields := []struct {
		raw  json.RawMessage
		dst  *bool
		name string
	}{
		{value.Online, &u.Online, "online"},
		{value.Bot, &u.Bot, "bot"},
		{value.IsVip, &u.IsVip, "is_vip"},
		{value.VipAmp, &u.VipAmp, "vip_amp"},
		{value.MobileVerified, &u.MobileVerified, "mobile_verified"},
		{value.IsSys, &u.IsSys, "is_sys"},
		{value.Verified, &u.Verified, "verified"},
		{value.IsAIReduceNoise, &u.IsAIReduceNoise, "is_ai_reduce_noise"},
		{value.IsPersonalCardBG, &u.IsPersonalCardBG, "is_personal_card_bg"},
	}
	for _, field := range boolFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeBoolOrInt(field.raw)
		if err != nil {
			return fmt.Errorf("解析user.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	if len(value.Roles) > 0 {
		roles, err := decodeIntSlice(value.Roles)
		if err != nil {
			return fmt.Errorf("解析user.roles失败: %w", err)
		}
		u.Roles = roles
	}
	return nil
}

// UserLiveInfo 是用户直播状态。
type UserLiveInfo struct {
	InLive        bool   `json:"in_live"`
	AudienceCount int    `json:"audience_count"`
	LiveThumb     string `json:"live_thumb"`
	LiveStartTime int64  `json:"live_start_time"`
}

func (i *UserLiveInfo) UnmarshalJSON(data []byte) error {
	type liveInfoAlias UserLiveInfo
	value := struct {
		*liveInfoAlias
		InLive json.RawMessage `json:"in_live"`
	}{liveInfoAlias: (*liveInfoAlias)(i)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.InLive) > 0 {
		inLive, err := decodeBoolOrInt(value.InLive)
		if err != nil {
			return fmt.Errorf("解析user.live_info.in_live失败: %w", err)
		}
		i.InLive = inLive
	}
	return nil
}

// TagInfo 标签信息
type TagInfo struct {
	Color   string `json:"color"`
	BGColor string `json:"bg_color"`
	Text    string `json:"text"`
}

// Guild 服务器信息
type Guild struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Topic            string         `json:"topic"`
	UserID           string         `json:"user_id"`
	Icon             string         `json:"icon"`
	NotifyType       int            `json:"notify_type"`
	Region           string         `json:"region"`
	EnableOpen       bool           `json:"enable_open"`
	OpenID           string         `json:"open_id"`
	DefaultChannelID string         `json:"default_channel_id"`
	WelcomeChannelID string         `json:"welcome_channel_id"`
	Roles            []Role         `json:"roles"`
	Channels         []Channel      `json:"channels"`
	MaxPersons       int            `json:"max_persons"`
	Level            int            `json:"level"`
	BoostNum         int            `json:"boost_num"`
	BufferBoostNum   int            `json:"buffer_boost_num"`
	Banner           string         `json:"banner"`
	Features         []GuildFeature `json:"features"`
	Emojis           []Emoji        `json:"emojis"`
}

// UnmarshalJSON 兼容服务器对象中的 string/int ID 和 bool/int 标志。
func (g *Guild) UnmarshalJSON(data []byte) error {
	type guildAlias Guild
	value := struct {
		*guildAlias
		ID               json.RawMessage `json:"id"`
		UserID           json.RawMessage `json:"user_id"`
		OpenID           json.RawMessage `json:"open_id"`
		DefaultChannelID json.RawMessage `json:"default_channel_id"`
		WelcomeChannelID json.RawMessage `json:"welcome_channel_id"`
		EnableOpen       json.RawMessage `json:"enable_open"`
		NotifyType       json.RawMessage `json:"notify_type"`
	}{guildAlias: (*guildAlias)(g)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	stringFields := []struct {
		raw  json.RawMessage
		dst  *string
		name string
	}{
		{value.ID, &g.ID, "id"},
		{value.UserID, &g.UserID, "user_id"},
		{value.OpenID, &g.OpenID, "open_id"},
		{value.DefaultChannelID, &g.DefaultChannelID, "default_channel_id"},
		{value.WelcomeChannelID, &g.WelcomeChannelID, "welcome_channel_id"},
	}
	for _, field := range stringFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringOrNumber(field.raw)
		if err != nil {
			return fmt.Errorf("解析guild.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	if len(value.EnableOpen) > 0 {
		enableOpen, err := decodeBoolOrInt(value.EnableOpen)
		if err != nil {
			return fmt.Errorf("解析guild.enable_open失败: %w", err)
		}
		g.EnableOpen = enableOpen
	}
	if len(value.NotifyType) > 0 {
		notifyType, err := decodeIntOrString(value.NotifyType)
		if err != nil {
			return fmt.Errorf("解析guild.notify_type失败: %w", err)
		}
		g.NotifyType = notifyType
	}
	return nil
}

// GuildFeature 服务器功能特性
type GuildFeature struct {
	Feature     string `json:"feature"`
	Description string `json:"description"`
}

// Role 角色信息
type Role struct {
	RoleID      int    `json:"role_id"`
	Name        string `json:"name"`
	Color       int    `json:"color"`
	Position    int    `json:"position"`
	Hoist       int    `json:"hoist"`
	Mentionable int    `json:"mentionable"`
	Permissions int    `json:"permissions"`
}

// Channel 频道信息
type Channel struct {
	ID                   string                `json:"id"`
	Name                 string                `json:"name"`
	UserID               string                `json:"user_id"`
	GuildID              string                `json:"guild_id"`
	Topic                string                `json:"topic"`
	IsCategory           bool                  `json:"is_category"`
	ParentID             string                `json:"parent_id"`
	Level                int                   `json:"level"`
	SlowMode             int                   `json:"slow_mode"`
	Type                 int                   `json:"type"`
	PermissionOverwrites []PermissionOverwrite `json:"permission_overwrites"`
	PermissionUsers      []PermissionUser      `json:"permission_users"`
	PermissionSync       int                   `json:"permission_sync"`
	HasPassword          bool                  `json:"has_password"`
	LimitAmount          int                   `json:"limit_amount"`
	VoiceQuality         string                `json:"voice_quality"`
	ServerURL            string                `json:"server_url"`
	Children             []string              `json:"children"`
	IsReadonly           bool                  `json:"is_readonly"`
	IsPrivate            bool                  `json:"is_private"`
	ServerType           int                   `json:"server_type"`
	IsMaster             bool                  `json:"is_master"`
	Mode                 int                   `json:"mode"`
}

// UnmarshalJSON 兼容频道对象在 HTTP 与系统事件中的 string/int、bool/int 差异。
func (c *Channel) UnmarshalJSON(data []byte) error {
	type channelAlias Channel
	value := struct {
		*channelAlias
		ID             json.RawMessage `json:"id"`
		GuildID        json.RawMessage `json:"guild_id"`
		UserID         json.RawMessage `json:"user_id"`
		ParentID       json.RawMessage `json:"parent_id"`
		VoiceQuality   json.RawMessage `json:"voice_quality"`
		IsCategory     json.RawMessage `json:"is_category"`
		HasPassword    json.RawMessage `json:"has_password"`
		IsReadonly     json.RawMessage `json:"is_readonly"`
		IsPrivate      json.RawMessage `json:"is_private"`
		IsMaster       json.RawMessage `json:"is_master"`
		Type           json.RawMessage `json:"type"`
		Level          json.RawMessage `json:"level"`
		SlowMode       json.RawMessage `json:"slow_mode"`
		PermissionSync json.RawMessage `json:"permission_sync"`
		LimitAmount    json.RawMessage `json:"limit_amount"`
		ServerType     json.RawMessage `json:"server_type"`
		Mode           json.RawMessage `json:"mode"`
		Children       json.RawMessage `json:"children"`
	}{channelAlias: (*channelAlias)(c)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	stringFields := []struct {
		raw  json.RawMessage
		dst  *string
		name string
	}{
		{value.ID, &c.ID, "id"},
		{value.GuildID, &c.GuildID, "guild_id"},
		{value.UserID, &c.UserID, "user_id"},
		{value.ParentID, &c.ParentID, "parent_id"},
		{value.VoiceQuality, &c.VoiceQuality, "voice_quality"},
	}
	for _, field := range stringFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringOrNumber(field.raw)
		if err != nil {
			return fmt.Errorf("解析channel.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	boolFields := []struct {
		raw  json.RawMessage
		dst  *bool
		name string
	}{
		{value.IsCategory, &c.IsCategory, "is_category"},
		{value.HasPassword, &c.HasPassword, "has_password"},
		{value.IsReadonly, &c.IsReadonly, "is_readonly"},
		{value.IsPrivate, &c.IsPrivate, "is_private"},
		{value.IsMaster, &c.IsMaster, "is_master"},
	}
	for _, field := range boolFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeBoolOrInt(field.raw)
		if err != nil {
			return fmt.Errorf("解析channel.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	intFields := []struct {
		raw  json.RawMessage
		dst  *int
		name string
	}{
		{value.Type, &c.Type, "type"},
		{value.Level, &c.Level, "level"},
		{value.SlowMode, &c.SlowMode, "slow_mode"},
		{value.PermissionSync, &c.PermissionSync, "permission_sync"},
		{value.LimitAmount, &c.LimitAmount, "limit_amount"},
		{value.ServerType, &c.ServerType, "server_type"},
		{value.Mode, &c.Mode, "mode"},
	}
	for _, field := range intFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeIntOrString(field.raw)
		if err != nil {
			return fmt.Errorf("解析channel.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	if len(value.Children) > 0 {
		children, err := decodeStringSlice(value.Children)
		if err != nil {
			return fmt.Errorf("解析channel.children失败: %w", err)
		}
		c.Children = children
	}
	return nil
}

// PermissionOverwrite 权限覆写
type PermissionOverwrite struct {
	RoleID int `json:"role_id"`
	Allow  int `json:"allow"`
	Deny   int `json:"deny"`
}

func (p *PermissionOverwrite) UnmarshalJSON(data []byte) error {
	type overwriteAlias PermissionOverwrite
	value := struct {
		*overwriteAlias
		RoleID json.RawMessage `json:"role_id"`
	}{overwriteAlias: (*overwriteAlias)(p)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.RoleID) > 0 {
		roleID, err := decodeIntOrString(value.RoleID)
		if err != nil {
			return fmt.Errorf("解析permission_overwrite.role_id失败: %w", err)
		}
		p.RoleID = roleID
	}
	return nil
}

// PermissionUser 用户权限
type PermissionUser struct {
	User  User `json:"user"`
	Allow int  `json:"allow"`
	Deny  int  `json:"deny"`
}

// Message 消息信息
type Message struct {
	ID           string            `json:"id"`
	Type         MessageType       `json:"type"`
	Content      string            `json:"content"`
	Mention      []string          `json:"mention"`
	MentionAll   bool              `json:"mention_all"`
	MentionRoles []string          `json:"mention_roles"`
	MentionHere  bool              `json:"mention_here"`
	Embeds       []interface{}     `json:"embeds"`
	Attachments  AttachmentPayload `json:"attachments"`
	CreateAt     int64             `json:"create_at"`
	UpdatedAt    int64             `json:"updated_at"`
	Reactions    []Reaction        `json:"reactions"`
	Author       User              `json:"author"`
	AuthorID     string            `json:"author_id"`
	ImageName    string            `json:"image_name"`
	ReadStatus   bool              `json:"read_status"`
	Quote        *Quote            `json:"quote"`
	MentionInfo  MentionInfo       `json:"mention_info"`
	FromType     int               `json:"from_type"`
	MsgIcon      string            `json:"msg_icon"`
}

// UnmarshalJSON 兼容消息对象中的数字 ID、0/1 布尔值和混合 ID 数组。
func (m *Message) UnmarshalJSON(data []byte) error {
	type messageAlias Message
	value := struct {
		*messageAlias
		ID           json.RawMessage `json:"id"`
		AuthorID     json.RawMessage `json:"author_id"`
		Mention      json.RawMessage `json:"mention"`
		MentionRoles json.RawMessage `json:"mention_roles"`
		MentionAll   json.RawMessage `json:"mention_all"`
		MentionHere  json.RawMessage `json:"mention_here"`
		ReadStatus   json.RawMessage `json:"read_status"`
	}{messageAlias: (*messageAlias)(m)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	stringFields := []struct {
		raw  json.RawMessage
		dst  *string
		name string
	}{
		{value.ID, &m.ID, "id"},
		{value.AuthorID, &m.AuthorID, "author_id"},
	}
	for _, field := range stringFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringOrNumber(field.raw)
		if err != nil {
			return fmt.Errorf("解析message.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	sliceFields := []struct {
		raw  json.RawMessage
		dst  *[]string
		name string
	}{
		{value.Mention, &m.Mention, "mention"},
		{value.MentionRoles, &m.MentionRoles, "mention_roles"},
	}
	for _, field := range sliceFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringSlice(field.raw)
		if err != nil {
			return fmt.Errorf("解析message.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	boolFields := []struct {
		raw  json.RawMessage
		dst  *bool
		name string
	}{
		{value.MentionAll, &m.MentionAll, "mention_all"},
		{value.MentionHere, &m.MentionHere, "mention_here"},
		{value.ReadStatus, &m.ReadStatus, "read_status"},
	}
	for _, field := range boolFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeBoolOrInt(field.raw)
		if err != nil {
			return fmt.Errorf("解析message.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	return nil
}

// Attachment 附件信息
type Attachment struct {
	Type     string  `json:"type"`
	URL      string  `json:"url"`
	Name     string  `json:"name"`
	FileType string  `json:"file_type"`
	Size     int64   `json:"size"`
	Duration float64 `json:"duration"`
	Width    int     `json:"width"`
	Height   int     `json:"height"`
}

// AttachmentPayload 兼容 KOOK 不同消息接口返回的 attachments 形态。
type AttachmentPayload []Attachment

// UnmarshalJSON 兼容 object、array、null 和 false。
func (a *AttachmentPayload) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	switch string(data) {
	case "null", "false":
		*a = nil
		return nil
	}

	var list []Attachment
	if err := json.Unmarshal(data, &list); err == nil {
		*a = list
		return nil
	}

	var single Attachment
	if err := json.Unmarshal(data, &single); err != nil {
		return err
	}
	*a = []Attachment{single}
	return nil
}

// Reaction 反应信息
type Reaction struct {
	Emoji Emoji `json:"emoji"`
	Count int   `json:"count"`
	Me    bool  `json:"me"`
}

func (r *Reaction) UnmarshalJSON(data []byte) error {
	type reactionAlias Reaction
	value := struct {
		*reactionAlias
		Me json.RawMessage `json:"me"`
	}{reactionAlias: (*reactionAlias)(r)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Me) > 0 {
		me, err := decodeBoolOrInt(value.Me)
		if err != nil {
			return fmt.Errorf("解析reaction.me失败: %w", err)
		}
		r.Me = me
	}
	return nil
}

// Quote 引用消息
type Quote struct {
	ID        string      `json:"id"`
	Type      MessageType `json:"type"`
	Content   string      `json:"content"`
	CreateAt  int64       `json:"create_at"`
	Author    User        `json:"author"`
	RonCreate bool        `json:"ron_create"`
}

// MentionInfo 提及信息
type MentionInfo struct {
	MentionPart     []MentionPart     `json:"mention_part"`
	MentionRolePart []MentionRolePart `json:"mention_role_part"`
	ChannelPart     []ChannelPart     `json:"channel_part"`
	ItemPart        []json.RawMessage `json:"item_part"`
}

// MentionPart 提及用户信息
type MentionPart struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	FullName string `json:"full_name"`
	Avatar   string `json:"avatar"`
}

// MentionRolePart 提及角色信息
type MentionRolePart struct {
	RoleID int    `json:"role_id"`
	Name   string `json:"name"`
	Color  int    `json:"color"`
}

// Gateway 网关信息
type Gateway struct {
	URL string `json:"url"`
}

// PaginationMeta 分页信息
type PaginationMeta struct {
	Page      int `json:"page"`
	PageTotal int `json:"page_total"`
	PageSize  int `json:"page_size"`
	Total     int `json:"total"`
}

// ListResponse 列表响应
type ListResponse struct {
	Items []interface{}  `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// Time 自定义时间类型，用于处理KOOK API的时间戳
type Time struct {
	time.Time
}

// UnmarshalJSON 实现JSON反序列化
func (t *Time) UnmarshalJSON(data []byte) error {
	var timestamp int64
	if err := json.Unmarshal(data, &timestamp); err != nil {
		return err
	}
	t.Time = time.Unix(timestamp/1000, (timestamp%1000)*1000000)
	return nil
}

// MarshalJSON 实现JSON序列化
func (t Time) MarshalJSON() ([]byte, error) {
	timestamp := t.Unix()*1000 + int64(t.Nanosecond()/1000000)
	return json.Marshal(timestamp)
}
