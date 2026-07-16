package kook

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultWebhookDedupTTL = 10 * time.Minute
	defaultWebhookDedupMax = 10000
)

var ErrWebhookVerifyToken = errors.New("Webhook verify_token不匹配")

// WebhookDeduplicator 可由数据库或缓存实现，以便多实例共享事件去重状态。
// duplicate=true 表示该 SN 已处理过；实现必须原子地检查并记录 SN。
type WebhookDeduplicator interface {
	CheckAndStore(ctx context.Context, sn int, ttl time.Duration) (duplicate bool, err error)
}

// MemoryWebhookDeduplicator 是有界、TTL 型的进程内去重器。
type MemoryWebhookDeduplicator struct {
	mu         sync.Mutex
	entries    map[int]time.Time
	maxEntries int
	now        func() time.Time
}

func NewMemoryWebhookDeduplicator(maxEntries int) *MemoryWebhookDeduplicator {
	if maxEntries <= 0 {
		maxEntries = defaultWebhookDedupMax
	}
	return &MemoryWebhookDeduplicator{
		entries:    make(map[int]time.Time),
		maxEntries: maxEntries,
		now:        time.Now,
	}
}

func (d *MemoryWebhookDeduplicator) CheckAndStore(_ context.Context, sn int, ttl time.Duration) (bool, error) {
	if sn <= 0 {
		return false, nil
	}
	if ttl <= 0 {
		ttl = defaultWebhookDedupTTL
	}
	now := d.now()

	d.mu.Lock()
	defer d.mu.Unlock()
	for storedSN, expiresAt := range d.entries {
		if !expiresAt.After(now) {
			delete(d.entries, storedSN)
		}
	}
	if expiresAt, exists := d.entries[sn]; exists && expiresAt.After(now) {
		return true, nil
	}
	if len(d.entries) >= d.maxEntries {
		oldestSN := 0
		var oldestExpiry time.Time
		for storedSN, expiresAt := range d.entries {
			if oldestExpiry.IsZero() || expiresAt.Before(oldestExpiry) {
				oldestSN = storedSN
				oldestExpiry = expiresAt
			}
		}
		delete(d.entries, oldestSN)
	}
	d.entries[sn] = now.Add(ttl)
	return false, nil
}

type WebhookOption func(*WebhookHandler)

// WithWebhookDeduplicator 注入持久化或分布式去重器。
func WithWebhookDeduplicator(deduplicator WebhookDeduplicator) WebhookOption {
	return func(handler *WebhookHandler) {
		if deduplicator != nil {
			handler.deduplicator = deduplicator
		}
	}
}

// WithWebhookDedupTTL 设置 SN 在去重存储中的有效期。
func WithWebhookDedupTTL(ttl time.Duration) WebhookOption {
	return func(handler *WebhookHandler) {
		if ttl > 0 {
			handler.dedupTTL = ttl
		}
	}
}

// WebhookHandler 处理 KOOK Webhook 验证、解密、解压、去重和事件分发。
type WebhookHandler struct {
	client       *Client
	encryptKey   string
	verifyToken  string
	dispatcher   *eventDispatcher
	deduplicator WebhookDeduplicator
	dedupTTL     time.Duration
}

type WebhookMessage struct {
	S  int             `json:"s"`
	D  json.RawMessage `json:"d"`
	SN int             `json:"sn"`
}

type encryptedWebhookMessage struct {
	Encrypt string `json:"encrypt"`
}

type webhookPayloadMeta struct {
	ChannelType EventChannelType `json:"channel_type"`
	VerifyToken string           `json:"verify_token"`
	Challenge   string           `json:"challenge"`
}

func NewWebhookHandler(client *Client, encryptKey, verifyToken string, options ...WebhookOption) *WebhookHandler {
	handler := &WebhookHandler{
		client:       client,
		encryptKey:   encryptKey,
		verifyToken:  verifyToken,
		dispatcher:   newEventDispatcher(),
		deduplicator: NewMemoryWebhookDeduplicator(defaultWebhookDedupMax),
		dedupTTL:     defaultWebhookDedupTTL,
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (wh *WebhookHandler) OnMessage(eventType MessageType, handler MessageEventHandler) {
	wh.dispatcher.onMessage(eventType, handler)
}

func (wh *WebhookHandler) OnSystemEvent(eventType SystemEventType, handler SystemEventHandler) {
	wh.dispatcher.onSystemEvent(eventType, handler)
}

func (wh *WebhookHandler) OnAnyEvent(handler RawEventHandler) {
	wh.dispatcher.onAnyEvent(handler)
}

func (wh *WebhookHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.client.logger.WithError(err).Error("读取请求体失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	body, err = decodeRequestBody(body, r.Header.Get("Content-Encoding"))
	if err != nil {
		wh.client.logger.WithError(err).Error("解码Webhook请求体失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	body, err = wh.tryDecryptBody(body)
	if err != nil {
		wh.client.logger.WithError(err).Error("解密Webhook请求体失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	var msg WebhookMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		wh.client.logger.WithError(err).Error("解析Webhook消息失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	challenge, err := wh.handleMessage(r.Context(), &msg)
	if err != nil {
		wh.client.logger.WithError(err).Error("处理Webhook消息失败")
		if errors.Is(err, ErrWebhookVerifyToken) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if challenge != "" {
		_ = json.NewEncoder(w).Encode(map[string]string{"challenge": challenge})
		return
	}
	_, _ = w.Write([]byte(`{"code":0}`))
}

func decodeRequestBody(body []byte, encoding string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "identity":
		if json.Valid(body) {
			return body, nil
		}
		if decoded, err := readZlib(body); err == nil {
			return decoded, nil
		}
		if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
			return readGzip(body)
		}
		return body, nil
	case "gzip":
		return readGzip(body)
	case "deflate":
		return readZlib(body)
	default:
		return nil, fmt.Errorf("不支持的Content-Encoding: %s", encoding)
	}
}

func readGzip(body []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func readZlib(body []byte) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func (wh *WebhookHandler) tryDecryptBody(body []byte) ([]byte, error) {
	var encrypted encryptedWebhookMessage
	if err := json.Unmarshal(body, &encrypted); err != nil || encrypted.Encrypt == "" {
		return body, nil
	}
	return decryptWebhookPayload(encrypted.Encrypt, wh.encryptKey)
}

func decryptWebhookPayload(encrypted, encryptKey string) ([]byte, error) {
	if encryptKey == "" {
		return nil, fmt.Errorf("Webhook消息已加密但未配置encryptKey")
	}
	keyBytes := []byte(encryptKey)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	payload, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, fmt.Errorf("解码encrypt失败: %w", err)
	}
	if len(payload) <= aes.BlockSize {
		return nil, fmt.Errorf("encrypt内容长度异常")
	}
	iv := payload[:aes.BlockSize]
	cipherText, err := base64.StdEncoding.DecodeString(string(payload[aes.BlockSize:]))
	if err != nil {
		return nil, fmt.Errorf("解码cipher失败: %w", err)
	}
	if len(cipherText)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("cipher长度不是BlockSize整数倍")
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}
	plainText := make([]byte, len(cipherText))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plainText, cipherText)
	plainText, err = pkcs7Unpad(plainText, aes.BlockSize)
	if err != nil {
		return nil, err
	}
	return plainText, nil
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, fmt.Errorf("无效的PKCS7数据")
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, fmt.Errorf("无效的PKCS7填充")
	}
	for i := len(data) - pad; i < len(data); i++ {
		if int(data[i]) != pad {
			return nil, fmt.Errorf("无效的PKCS7填充字节")
		}
	}
	return data[:len(data)-pad], nil
}

func (wh *WebhookHandler) handleMessage(ctx context.Context, msg *WebhookMessage) (string, error) {
	if msg.S != SignalEvent {
		return "", nil
	}
	var meta webhookPayloadMeta
	if err := json.Unmarshal(msg.D, &meta); err != nil {
		return "", fmt.Errorf("解析Webhook元数据失败: %w", err)
	}
	if wh.verifyToken != "" && meta.VerifyToken != wh.verifyToken {
		return "", ErrWebhookVerifyToken
	}
	if meta.ChannelType == EventChannelTypeWebhookChallenge && meta.Challenge != "" {
		return meta.Challenge, nil
	}
	return "", wh.handleEvent(ctx, msg)
}

func (wh *WebhookHandler) handleEvent(ctx context.Context, msg *WebhookMessage) error {
	var event Event
	if err := json.Unmarshal(msg.D, &event); err != nil {
		return fmt.Errorf("解析事件失败: %w", err)
	}
	event.S = msg.S
	event.SN = msg.SN

	duplicate, err := wh.deduplicator.CheckAndStore(ctx, msg.SN, wh.dedupTTL)
	if err != nil {
		return fmt.Errorf("检查Webhook事件SN失败: %w", err)
	}
	if duplicate {
		wh.client.logger.Debugf("忽略重复Webhook事件SN: %d", msg.SN)
		return nil
	}

	go func() {
		err := wh.dispatcher.dispatch(&event, func(recovered any) {
			wh.client.logger.Errorf("事件处理器发生panic: %v", recovered)
		})
		if err != nil {
			wh.client.logger.WithError(err).Error("分发Webhook事件失败")
		}
	}()
	return nil
}

func (wh *WebhookHandler) StartWebhookServer(addr, path string) error {
	http.HandleFunc(path, wh.HandleRequest)
	wh.client.logger.Infof("启动Webhook服务器: %s%s", addr, path)
	return http.ListenAndServe(addr, nil)
}
