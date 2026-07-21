package kook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ThreadService 帖子相关API服务
type ThreadService struct {
	client *Client
}

type ThreadCategoryListParams struct {
	ChannelID string
}

// GetThreadCategories 获取帖子分区列表。
func (s *ThreadService) GetThreadCategories(ctx context.Context, args ...any) (*ThreadCategoryListResponse, error) {
	params, err := compatParams("GetThreadCategories", args, func(args []any) (ThreadCategoryListParams, bool) {
		if len(args) != 1 {
			return ThreadCategoryListParams{}, false
		}
		channelID, ok := compatString(args[0])
		return ThreadCategoryListParams{ChannelID: channelID}, ok
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	resp, err := s.client.Get(ctx, "category/list", map[string]string{
		"channel_id": params.ChannelID,
	})
	if err != nil {
		return nil, err
	}

	var result ThreadCategoryListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子分区列表失败: %w", err)
	}

	return &result, nil
}

// CreateThread 发布帖子
func (s *ThreadService) CreateThread(ctx context.Context, params CreateThreadParams) (*Thread, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}
	if params.GuildID == "" {
		return nil, fmt.Errorf("服务器ID不能为空")
	}
	if params.Title == "" {
		return nil, fmt.Errorf("帖子标题不能为空")
	}
	if params.Content == "" {
		return nil, fmt.Errorf("帖子内容不能为空")
	}

	resp, err := s.client.Post(ctx, "thread/create", params.toMap())
	if err != nil {
		return nil, err
	}

	var result Thread
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子创建结果失败: %w", err)
	}

	return &result, nil
}

// ReplyThread 回复帖子
func (s *ThreadService) ReplyThread(ctx context.Context, params ReplyThreadParams) (*Post, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}
	if params.ThreadID == "" {
		return nil, fmt.Errorf("帖子ID不能为空")
	}
	if params.Content == "" {
		return nil, fmt.Errorf("回复内容不能为空")
	}

	resp, err := s.client.Post(ctx, "thread/reply", params.toMap())
	if err != nil {
		return nil, err
	}

	var result Post
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子回复结果失败: %w", err)
	}

	return &result, nil
}

// ThreadReply 保留 v1.1.1 的精简回复类型。
type ThreadReply struct {
	ID        string `json:"id"`
	PostID    string `json:"post_id"`
	Content   string `json:"content"`
	User      User   `json:"user"`
	CreateAt  int64  `json:"create_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ReplyThreadLegacy 提供 v1.1.1 的返回值形状。
func (s *ThreadService) ReplyThreadLegacy(ctx context.Context, params ReplyThreadParams) (*ThreadReply, error) {
	post, err := s.ReplyThread(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ThreadReply{
		ID:        post.ID,
		PostID:    post.ThreadID,
		Content:   post.Content,
		User:      post.User,
		CreateAt:  post.CreateTime,
		UpdatedAt: 0,
	}, nil
}

type ThreadViewParams struct {
	ChannelID string
	ThreadID  string
}

// GetThread 获取帖子详情。
func (s *ThreadService) GetThread(ctx context.Context, args ...any) (*Thread, error) {
	params, err := compatParams("GetThread", args, func(args []any) (ThreadViewParams, bool) {
		if len(args) != 2 {
			return ThreadViewParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		threadID, okThread := compatString(args[1])
		return ThreadViewParams{ChannelID: channelID, ThreadID: threadID}, okChannel && okThread
	})
	if err != nil {
		return nil, err
	}
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}
	if params.ThreadID == "" {
		return nil, fmt.Errorf("帖子ID不能为空")
	}

	resp, err := s.client.Get(ctx, "thread/view", map[string]string{
		"channel_id": params.ChannelID,
		"thread_id":  params.ThreadID,
	})
	if err != nil {
		return nil, err
	}

	var result Thread
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子详情失败: %w", err)
	}

	return &result, nil
}

// GetThreadList 获取帖子列表
func (s *ThreadService) GetThreadList(ctx context.Context, params GetThreadListParams) (*ThreadListResponse, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	resp, err := s.client.Get(ctx, "thread/list", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result ThreadListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子列表失败: %w", err)
	}

	return &result, nil
}

type ThreadDeleteParams struct {
	ChannelID string
	ThreadID  string
	PostID    string
}

// DeleteThread 删除帖子或帖子回复。
func (s *ThreadService) DeleteThread(ctx context.Context, args ...any) error {
	params, err := compatParams("DeleteThread", args, func(args []any) (ThreadDeleteParams, bool) {
		if len(args) != 3 {
			return ThreadDeleteParams{}, false
		}
		channelID, okChannel := compatString(args[0])
		threadID, okThread := compatString(args[1])
		postID, okPost := compatString(args[2])
		return ThreadDeleteParams{ChannelID: channelID, ThreadID: threadID, PostID: postID}, okChannel && okThread && okPost
	})
	if err != nil {
		return err
	}
	if params.ChannelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if params.ThreadID == "" && params.PostID == "" {
		return fmt.Errorf("帖子ID或回复ID不能为空")
	}

	body := map[string]interface{}{
		"channel_id": params.ChannelID,
	}
	if params.ThreadID != "" {
		body["thread_id"] = params.ThreadID
	}
	if params.PostID != "" {
		body["post_id"] = params.PostID
	}

	_, err = s.client.Post(ctx, "thread/delete", body)
	return err
}

// GetThreadPosts 获取回复列表
func (s *ThreadService) GetThreadPosts(ctx context.Context, params GetThreadPostsParams) (*ThreadPostListResponse, error) {
	if params.ChannelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}
	if params.ThreadID == "" {
		return nil, fmt.Errorf("帖子ID不能为空")
	}
	if params.Order != "asc" && params.Order != "desc" {
		return nil, fmt.Errorf("order必须为asc或desc")
	}
	if params.Page <= 0 {
		return nil, fmt.Errorf("page必须大于0")
	}

	resp, err := s.client.Get(ctx, "thread/post", params.toQuery())
	if err != nil {
		return nil, err
	}

	var result ThreadPostListResponse
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子回复列表失败: %w", err)
	}

	return &result, nil
}

// GetThreadPost 是 v1.1.1 的回复列表别名。
func (s *ThreadService) GetThreadPost(ctx context.Context, channelID, threadID string) (*ThreadPostListResponse, error) {
	return s.GetThreadPosts(ctx, GetThreadPostsParams{
		ChannelID: channelID,
		ThreadID:  threadID,
		Order:     "asc",
		Page:      1,
	})
}

// CreateThreadParams 发布帖子参数
type CreateThreadParams struct {
	ChannelID  string `json:"channel_id"`
	GuildID    string `json:"guild_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	CategoryID string `json:"category_id,omitempty"`
	Cover      string `json:"cover,omitempty"`
}

func (p CreateThreadParams) toMap() map[string]interface{} {
	params := map[string]interface{}{
		"channel_id": p.ChannelID,
		"guild_id":   p.GuildID,
		"title":      p.Title,
		"content":    p.Content,
	}
	if p.CategoryID != "" {
		params["category_id"] = p.CategoryID
	}
	if p.Cover != "" {
		params["cover"] = p.Cover
	}
	return params
}

// ReplyThreadParams 回复帖子参数
type ReplyThreadParams struct {
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
	ReplyID   string `json:"reply_id,omitempty"`
	Content   string `json:"content"`
}

func (p ReplyThreadParams) toMap() map[string]interface{} {
	params := map[string]interface{}{
		"channel_id": p.ChannelID,
		"thread_id":  p.ThreadID,
		"content":    p.Content,
	}
	if p.ReplyID != "" {
		params["reply_id"] = p.ReplyID
	}
	return params
}

// GetThreadListParams 获取帖子列表参数
type GetThreadListParams struct {
	ChannelID  string
	CategoryID string
	Sort       *int
	Time       *int64
	PageSize   *int
}

func (p GetThreadListParams) toQuery() map[string]string {
	query := map[string]string{"channel_id": p.ChannelID}
	if p.CategoryID != "" {
		query["category_id"] = p.CategoryID
	}
	if p.Sort != nil {
		query["sort"] = strconv.Itoa(*p.Sort)
	}
	if p.Time != nil {
		query["time"] = strconv.FormatInt(*p.Time, 10)
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	return query
}

// GetThreadPostsParams 获取帖子回复列表参数
type GetThreadPostsParams struct {
	ChannelID string
	ThreadID  string
	PostID    string
	Time      string
	PageSize  *int
	Order     string
	Page      int
}

func (p GetThreadPostsParams) toQuery() map[string]string {
	query := map[string]string{
		"channel_id": p.ChannelID,
		"thread_id":  p.ThreadID,
		"order":      p.Order,
		"page":       strconv.Itoa(p.Page),
	}
	if p.PostID != "" {
		query["post_id"] = p.PostID
	}
	if p.Time != "" {
		query["time"] = p.Time
	}
	if p.PageSize != nil {
		query["page_size"] = strconv.Itoa(*p.PageSize)
	}
	return query
}

// ThreadCategory 帖子分区
type ThreadCategory struct {
	ID        string                     `json:"id"`
	Name      string                     `json:"name"`
	ChannelID string                     `json:"channel_id"`
	Allow     int                        `json:"allow"`
	Deny      int                        `json:"deny"`
	Roles     []ThreadCategoryPermission `json:"roles"`
}

func (c *ThreadCategory) UnmarshalJSON(data []byte) error {
	type categoryAlias ThreadCategory
	value := struct {
		*categoryAlias
		ID    json.RawMessage `json:"id"`
		Roles json.RawMessage `json:"roles"`
	}{categoryAlias: (*categoryAlias)(c)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.ID) > 0 {
		id, err := decodeStringOrNumber(value.ID)
		if err != nil {
			return fmt.Errorf("解析thread category id失败: %w", err)
		}
		c.ID = id
	}
	roles := bytes.TrimSpace(value.Roles)
	if len(roles) == 0 || bytes.Equal(roles, []byte("null")) {
		c.Roles = nil
		return nil
	}
	if roles[0] == '{' {
		var keyed map[string]ThreadCategoryPermission
		if err := json.Unmarshal(roles, &keyed); err != nil {
			return fmt.Errorf("解析thread category roles失败: %w", err)
		}
		c.Roles = make([]ThreadCategoryPermission, 0, len(keyed))
		for _, permission := range keyed {
			c.Roles = append(c.Roles, permission)
		}
		return nil
	}
	if err := json.Unmarshal(roles, &c.Roles); err != nil {
		return fmt.Errorf("解析thread category roles失败: %w", err)
	}
	return nil
}

// ThreadCategoryPermission 帖子分区的角色或用户权限。
type ThreadCategoryPermission struct {
	Type   string `json:"type"`
	RoleID int    `json:"role_id"`
	UserID string `json:"user_id"`
	Allow  int    `json:"allow"`
	Deny   int    `json:"deny"`
}

func (p *ThreadCategoryPermission) UnmarshalJSON(data []byte) error {
	type permissionAlias ThreadCategoryPermission
	value := struct {
		*permissionAlias
		RoleID json.RawMessage `json:"role_id"`
		UserID json.RawMessage `json:"user_id"`
	}{permissionAlias: (*permissionAlias)(p)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.RoleID) > 0 {
		roleID, err := decodeIntOrString(value.RoleID)
		if err != nil {
			return fmt.Errorf("解析thread category role_id失败: %w", err)
		}
		p.RoleID = roleID
	}
	if len(value.UserID) > 0 {
		userID, err := decodeStringOrNumber(value.UserID)
		if err != nil {
			return fmt.Errorf("解析thread category user_id失败: %w", err)
		}
		p.UserID = userID
	}
	return nil
}

// ThreadMedia 帖子媒体。
type ThreadMedia struct {
	Type  string `json:"type"`
	Src   string `json:"src"`
	Title string `json:"title"`
}

func (m *ThreadMedia) UnmarshalJSON(data []byte) error {
	type mediaAlias ThreadMedia
	value := struct {
		*mediaAlias
		Type json.RawMessage `json:"type"`
	}{mediaAlias: (*mediaAlias)(m)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if len(value.Type) > 0 {
		mediaType, err := decodeStringOrNumber(value.Type)
		if err != nil {
			return fmt.Errorf("解析thread media type失败: %w", err)
		}
		m.Type = mediaType
	}
	return nil
}

// ChannelPart 帖子内容中引用的频道。
type ChannelPart struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Thread 帖子
type Thread struct {
	ID                 string            `json:"id"`
	ChannelID          string            `json:"channel_id"`
	CategoryID         string            `json:"category_id"`
	Status             int               `json:"status"`
	Title              string            `json:"title"`
	Cover              string            `json:"cover"`
	Category           ThreadCategory    `json:"category"`
	PostID             string            `json:"post_id"`
	Medias             []ThreadMedia     `json:"medias"`
	PreviewContent     string            `json:"preview_content"`
	User               User              `json:"user"`
	Tags               []json.RawMessage `json:"tags"`
	LatestActiveTime   int64             `json:"latest_active_time"`
	CreateTime         int64             `json:"create_time"`
	CreateAt           int64             `json:"create_at"`
	UpdatedAt          int64             `json:"updated_at"`
	IsUpdated          bool              `json:"is_updated"`
	ContentDeleted     bool              `json:"content_deleted"`
	ContentDeletedType int               `json:"content_deleted_type"`
	CollectNum         int               `json:"collect_num"`
	PostCount          int               `json:"post_count"`
	Content            string            `json:"content"`
	Mention            []string          `json:"mention"`
	MentionAll         bool              `json:"mention_all"`
	MentionHere        bool              `json:"mention_here"`
	MentionPart        []MentionPart     `json:"mention_part"`
	MentionRolePart    []MentionRolePart `json:"mention_role_part"`
	ChannelPart        []ChannelPart     `json:"channel_part"`
	ItemPart           []json.RawMessage `json:"item_part"`
}

// UnmarshalJSON 兼容 category 返回完整对象或仅返回分区 ID。
func (t *Thread) UnmarshalJSON(data []byte) error {
	type threadAlias Thread
	value := struct {
		*threadAlias
		ID             json.RawMessage `json:"id"`
		PostID         json.RawMessage `json:"post_id"`
		Category       json.RawMessage `json:"category"`
		Mention        json.RawMessage `json:"mention"`
		IsUpdated      json.RawMessage `json:"is_updated"`
		ContentDeleted json.RawMessage `json:"content_deleted"`
		MentionAll     json.RawMessage `json:"mention_all"`
		MentionHere    json.RawMessage `json:"mention_here"`
	}{threadAlias: (*threadAlias)(t)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	stringFields := []struct {
		raw  json.RawMessage
		dst  *string
		name string
	}{
		{value.ID, &t.ID, "id"},
		{value.PostID, &t.PostID, "post_id"},
	}
	for _, field := range stringFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringOrNumber(field.raw)
		if err != nil {
			return fmt.Errorf("解析thread.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	if len(value.Mention) > 0 {
		mention, err := decodeStringSlice(value.Mention)
		if err != nil {
			return fmt.Errorf("解析thread.mention失败: %w", err)
		}
		t.Mention = mention
	}
	boolFields := []struct {
		raw  json.RawMessage
		dst  *bool
		name string
	}{
		{value.IsUpdated, &t.IsUpdated, "is_updated"},
		{value.ContentDeleted, &t.ContentDeleted, "content_deleted"},
		{value.MentionAll, &t.MentionAll, "mention_all"},
		{value.MentionHere, &t.MentionHere, "mention_here"},
	}
	for _, field := range boolFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeBoolOrInt(field.raw)
		if err != nil {
			return fmt.Errorf("解析thread.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	category := bytes.TrimSpace(value.Category)
	if len(category) == 0 || bytes.Equal(category, []byte("null")) {
		return nil
	}
	if category[0] == '"' || (category[0] >= '0' && category[0] <= '9') {
		categoryID, err := decodeStringOrNumber(category)
		if err != nil {
			return fmt.Errorf("解析thread.category失败: %w", err)
		}
		t.Category.ID = categoryID
		return nil
	}
	if err := json.Unmarshal(category, &t.Category); err != nil {
		return fmt.Errorf("解析thread.category失败: %w", err)
	}
	return nil
}

// Post 帖子评论、回复或楼中楼。
type Post struct {
	ID              string            `json:"id"`
	CategoryID      string            `json:"category_id"`
	ThreadID        string            `json:"thread_id"`
	ReplyID         string            `json:"reply_id"`
	BelongToPostID  string            `json:"belong_to_post_id"`
	Content         string            `json:"content"`
	Status          int               `json:"status"`
	Mention         []string          `json:"mention"`
	MentionAll      bool              `json:"mention_all"`
	MentionHere     bool              `json:"mention_here"`
	MentionPart     []MentionPart     `json:"mention_part"`
	MentionRolePart []MentionRolePart `json:"mention_role_part"`
	ChannelPart     []ChannelPart     `json:"channel_part"`
	ItemPart        []json.RawMessage `json:"item_part"`
	CreateTime      int64             `json:"create_time"`
	IsUpdated       bool              `json:"is_updated"`
	User            User              `json:"user"`
	Replies         []Post            `json:"replies"`
}

func (p *Post) UnmarshalJSON(data []byte) error {
	type postAlias Post
	value := struct {
		*postAlias
		ID             json.RawMessage `json:"id"`
		CategoryID     json.RawMessage `json:"category_id"`
		ThreadID       json.RawMessage `json:"thread_id"`
		ReplyID        json.RawMessage `json:"reply_id"`
		BelongToPostID json.RawMessage `json:"belong_to_post_id"`
		Mention        json.RawMessage `json:"mention"`
		MentionAll     json.RawMessage `json:"mention_all"`
		MentionHere    json.RawMessage `json:"mention_here"`
		IsUpdated      json.RawMessage `json:"is_updated"`
	}{postAlias: (*postAlias)(p)}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	stringFields := []struct {
		raw  json.RawMessage
		dst  *string
		name string
	}{
		{value.ID, &p.ID, "id"},
		{value.CategoryID, &p.CategoryID, "category_id"},
		{value.ThreadID, &p.ThreadID, "thread_id"},
		{value.ReplyID, &p.ReplyID, "reply_id"},
		{value.BelongToPostID, &p.BelongToPostID, "belong_to_post_id"},
	}
	for _, field := range stringFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeStringOrNumber(field.raw)
		if err != nil {
			return fmt.Errorf("解析post.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	if len(value.Mention) > 0 {
		mention, err := decodeStringSlice(value.Mention)
		if err != nil {
			return fmt.Errorf("解析post.mention失败: %w", err)
		}
		p.Mention = mention
	}
	boolFields := []struct {
		raw  json.RawMessage
		dst  *bool
		name string
	}{
		{value.MentionAll, &p.MentionAll, "mention_all"},
		{value.MentionHere, &p.MentionHere, "mention_here"},
		{value.IsUpdated, &p.IsUpdated, "is_updated"},
	}
	for _, field := range boolFields {
		if len(field.raw) == 0 {
			continue
		}
		decoded, err := decodeBoolOrInt(field.raw)
		if err != nil {
			return fmt.Errorf("解析post.%s失败: %w", field.name, err)
		}
		*field.dst = decoded
	}
	return nil
}

// ThreadCategoryListResponse 帖子分区列表响应
type ThreadCategoryListResponse struct {
	List []ThreadCategory `json:"list"`
}

// ThreadListResponse 帖子列表响应
type ThreadListResponse struct {
	Items []Thread       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  SortFields     `json:"sort"`
}

// ThreadPostListResponse 帖子回复列表响应
type ThreadPostListResponse struct {
	Items []Post         `json:"items"`
	Meta  PaginationMeta `json:"meta"`
}
