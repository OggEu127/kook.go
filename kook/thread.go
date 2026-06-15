package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

// ThreadService 帖子相关API服务
type ThreadService struct {
	client *Client
}

// GetThreadCategories 获取帖子分区列表
func (s *ThreadService) GetThreadCategories(ctx context.Context, channelID string) (*ThreadCategoryListResponse, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}

	resp, err := s.client.Get(ctx, "category/list", map[string]string{
		"channel_id": channelID,
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
func (s *ThreadService) ReplyThread(ctx context.Context, params ReplyThreadParams) (*ThreadReply, error) {
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

	var result ThreadReply
	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("解析帖子回复结果失败: %w", err)
	}

	return &result, nil
}

// GetThread 获取帖子详情
func (s *ThreadService) GetThread(ctx context.Context, channelID, threadID string) (*Thread, error) {
	if channelID == "" {
		return nil, fmt.Errorf("频道ID不能为空")
	}
	if threadID == "" {
		return nil, fmt.Errorf("帖子ID不能为空")
	}

	resp, err := s.client.Get(ctx, "thread/view", map[string]string{
		"channel_id": channelID,
		"thread_id":  threadID,
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

// DeleteThread 删除帖子或帖子回复
func (s *ThreadService) DeleteThread(ctx context.Context, channelID, threadID, postID string) error {
	if channelID == "" {
		return fmt.Errorf("频道ID不能为空")
	}
	if threadID == "" && postID == "" {
		return fmt.Errorf("帖子ID或回复ID不能为空")
	}

	params := map[string]interface{}{
		"channel_id": channelID,
	}
	if threadID != "" {
		params["thread_id"] = threadID
	}
	if postID != "" {
		params["post_id"] = postID
	}

	_, err := s.client.Post(ctx, "thread/delete", params)
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
	if params.Order == "" {
		params.Order = "asc"
	}
	if params.Page <= 0 {
		params.Page = 1
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

// GetThreadPost 获取回复列表。保留旧方法名以兼容历史调用。
func (s *ThreadService) GetThreadPost(ctx context.Context, channelID, threadID string) (*ThreadPostListResponse, error) {
	return s.GetThreadPosts(ctx, GetThreadPostsParams{
		ChannelID: channelID,
		ThreadID:  threadID,
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
	ChannelID  string `json:"channel_id"`
	CategoryID string `json:"category_id,omitempty"`
	Sort       int    `json:"sort,omitempty"`
	Time       int64  `json:"time,omitempty"`
	PageSize   int    `json:"page_size,omitempty"`
}

func (p GetThreadListParams) toQuery() map[string]string {
	query := map[string]string{"channel_id": p.ChannelID}
	if p.CategoryID != "" {
		query["category_id"] = p.CategoryID
	}
	if p.Sort > 0 {
		query["sort"] = strconv.Itoa(p.Sort)
	}
	if p.Time > 0 {
		query["time"] = strconv.FormatInt(p.Time, 10)
	}
	if p.PageSize > 0 {
		query["page_size"] = strconv.Itoa(p.PageSize)
	}
	return query
}

// GetThreadPostsParams 获取帖子回复列表参数
type GetThreadPostsParams struct {
	ChannelID string `json:"channel_id"`
	ThreadID  string `json:"thread_id"`
	PostID    string `json:"post_id,omitempty"`
	Time      string `json:"time,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	Order     string `json:"order,omitempty"`
	Page      int    `json:"page,omitempty"`
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
	if p.PageSize > 0 {
		query["page_size"] = strconv.Itoa(p.PageSize)
	}
	return query
}

// ThreadCategory 帖子分区
type ThreadCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ChannelID string `json:"channel_id"`
}

// Thread 帖子
type Thread struct {
	ID         string `json:"id"`
	PostID     string `json:"post_id"`
	ChannelID  string `json:"channel_id"`
	CategoryID string `json:"category_id"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Cover      string `json:"cover"`
	User       User   `json:"user"`
	CreateAt   int64  `json:"create_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// ThreadReply 帖子回复
type ThreadReply struct {
	ID        string `json:"id"`
	PostID    string `json:"post_id"`
	Content   string `json:"content"`
	User      User   `json:"user"`
	CreateAt  int64  `json:"create_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ThreadCategoryListResponse 帖子分区列表响应
type ThreadCategoryListResponse struct {
	List []ThreadCategory `json:"list"`
}

// ThreadListResponse 帖子列表响应
type ThreadListResponse struct {
	Items []Thread       `json:"items"`
	Meta  PaginationMeta `json:"meta"`
	Sort  map[string]int `json:"sort"`
}

// ThreadPostListResponse 帖子回复列表响应
type ThreadPostListResponse struct {
	Items []ThreadReply  `json:"items"`
	Meta  PaginationMeta `json:"meta"`
}
