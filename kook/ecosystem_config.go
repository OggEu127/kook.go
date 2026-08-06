package kook

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

const maxEcosystemConfigBytes = 64 << 10

var communityNoticeFallback sync.Map

// LoadEcosystemOptions 从YAML文件加载生态配置。
//
// contribute_to_community省略时默认为true。SDK不会自动搜索或读取配置文件，
// 调用方必须显式传入路径并将结果交给WithEcosystem。
func LoadEcosystemOptions(path string) (EcosystemOptions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return EcosystemOptions{}, fmt.Errorf("读取生态配置失败: %w", err)
	}
	if len(data) > maxEcosystemConfigBytes {
		return EcosystemOptions{}, fmt.Errorf("生态配置不能超过64 KiB")
	}

	var options EcosystemOptions
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&options); err != nil {
		return EcosystemOptions{}, fmt.Errorf("解析生态配置失败: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return EcosystemOptions{}, fmt.Errorf("生态配置只能包含一个YAML文档")
	}
	if options.ContributeToCommunity == nil {
		options.ContributeToCommunity = CommunityContribution(true)
	}
	if options.NoticeStatePath == "" {
		absolutePath, err := filepath.Abs(path)
		if err != nil {
			return EcosystemOptions{}, fmt.Errorf("解析生态配置路径失败: %w", err)
		}
		options.NoticeStatePath = absolutePath + ".community-notice-v1"
	}
	if _, err := validateEcosystemBaseURL(options.BaseURL); err != nil {
		return EcosystemOptions{}, err
	}
	if options.Channel == "" {
		options.Channel = ReleaseChannelStable
	}
	if !validReleaseChannel(options.Channel) {
		return EcosystemOptions{}, fmt.Errorf("生态发布通道必须是stable或beta")
	}
	return options, nil
}

func (s *EcosystemService) showCommunityContributionNotice(path string) {
	if path == "" {
		path = defaultCommunityNoticeStatePath()
	}
	show, err := claimCommunityNotice(path)
	if err != nil {
		key := path
		if key == "" {
			key = "process"
		}
		if _, loaded := communityNoticeFallback.LoadOrStore(key, struct{}{}); loaded {
			return
		}
		s.client.logger.WithError(err).Debug("无法持久化SDK社区贡献提示状态")
		show = true
	}
	if !show {
		return
	}
	s.client.logger.Warn(
		"KOOK Go SDK匿名在线实例贡献默认开启；如需关闭，请在生态配置中设置 contribute_to_community: false。本提示仅显示一次",
	)
}

func defaultCommunityNoticeStatePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		return ""
	}
	return filepath.Join(configDir, "kook-go-sdk", "community-notice-v1")
}

func claimCommunityNotice(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("无法确定社区贡献提示状态路径")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if _, err := file.WriteString("KOOK Go SDK community contribution notice shown\n"); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return false, err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, err
	}
	return true, nil
}
