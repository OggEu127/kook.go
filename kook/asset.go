package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AssetService 媒体资源相关 API 服务。
type AssetService struct {
	client *Client
}

// UploadFile 是 CreateFromPath 的 v1.1.1 兼容别名。
func (s *AssetService) UploadFile(ctx context.Context, filePath string) (*Asset, error) {
	return s.CreateFromPath(ctx, AssetPathParams{Path: filePath})
}

// UploadFileContent 是 Create 的 v1.1.1 兼容别名。
func (s *AssetService) UploadFileContent(ctx context.Context, fileName string, content []byte) (*Asset, error) {
	return s.Create(ctx, AssetCreateParams{FileName: fileName, Content: content})
}

type AssetPathParams struct {
	Path string
}

// CreateFromPath 从本地路径上传媒体文件。
func (s *AssetService) CreateFromPath(ctx context.Context, params AssetPathParams) (*Asset, error) {
	if params.Path == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	file, err := os.Open(params.Path)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer func() { _ = file.Close() }()

	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	return s.Create(ctx, AssetCreateParams{
		FileName: filepath.Base(params.Path),
		Content:  fileContent,
	})
}

type AssetCreateParams struct {
	FileName string
	Content  []byte
}

// Create 上传内存中的媒体文件。
func (s *AssetService) Create(ctx context.Context, params AssetCreateParams) (*Asset, error) {
	if params.FileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if len(params.Content) == 0 {
		return nil, fmt.Errorf("文件内容不能为空")
	}

	response, err := s.client.doMultipartRequest(ctx, "asset/create", nil, map[string]multipartFile{
		"file": {
			FileName: params.FileName,
			Content:  params.Content,
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	var asset Asset
	if err := json.Unmarshal(response.Data, &asset); err != nil {
		return nil, fmt.Errorf("解析资源信息失败: %w", err)
	}

	return &asset, nil
}

// Asset 媒体资源信息。
type Asset struct {
	URL  string `json:"url"`
	Type string `json:"type"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}
