package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AssetService 媒体资源相关API服务
type AssetService struct {
	client *Client
}

// UploadFile 上传文件
func (s *AssetService) UploadFile(ctx context.Context, filePath string) (*Asset, error) {
	if filePath == "" {
		return nil, fmt.Errorf("文件路径不能为空")
	}

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 读取文件内容
	fileContent, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}

	fileName := filepath.Base(filePath)
	return s.UploadFileContent(ctx, fileName, fileContent)
}

// UploadFileContent 上传文件内容
func (s *AssetService) UploadFileContent(ctx context.Context, fileName string, content []byte) (*Asset, error) {
	if fileName == "" {
		return nil, fmt.Errorf("文件名不能为空")
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("文件内容不能为空")
	}

	s.client.logger.Debugf("上传文件: %s", fileName)
	response, err := s.client.doMultipartRequest(ctx, "asset/create", nil, map[string]multipartFile{
		"file": {
			FileName: fileName,
			Content:  content,
		},
	}, nil)
	if err != nil {
		return nil, err
	}

	var asset Asset
	if err := json.Unmarshal(response.Data, &asset); err != nil {
		return nil, fmt.Errorf("解析资源信息失败: %w", err)
	}

	s.client.logger.Infof("文件上传成功: %s -> %s", fileName, asset.URL)
	return &asset, nil
}

// 数据结构定义

// Asset 媒体资源信息
type Asset struct {
	URL  string `json:"url"`  // 资源URL
	Type string `json:"type"` // 资源类型
	Name string `json:"name"` // 文件名
	Size int64  `json:"size"` // 文件大小
}
