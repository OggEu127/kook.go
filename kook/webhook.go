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
	defaultWebhookMaxBody  = 1 << 20
	defaultWebhookMaxData  = 8 << 20
	defaultWebhookQueue    = 256
	defaultWebhookWorkers  = 1
)

var (
	ErrWebhookVerifyToken  = errors.New("webhook verify_token不匹配")
	ErrWebhookBodyTooLarge = errors.New("Webhook请求体超过限制")
	ErrWebhookQueueFull    = errors.New("Webhook事件队列已满")
	ErrWebhookClosed       = errors.New("Webhook处理器已关闭")
)

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
		handler.deduplicator = deduplicator
	}
}

// WithWebhookDedupTTL 设置 SN 在去重存储中的有效期。
func WithWebhookDedupTTL(ttl time.Duration) WebhookOption {
	return func(handler *WebhookHandler) {
		handler.dedupTTL = ttl
	}
}

// WithWebhookBodyLimits 设置压缩前和解压后的最大请求体字节数。
func WithWebhookBodyLimits(maxBodyBytes, maxDecodedBytes int64) WebhookOption {
	return func(handler *WebhookHandler) {
		handler.maxBodyBytes = maxBodyBytes
		handler.maxDecodedBytes = maxDecodedBytes
	}
}

// WithWebhookDispatch 设置有界事件队列容量和worker数量。
func WithWebhookDispatch(queueSize, workers int) WebhookOption {
	return func(handler *WebhookHandler) {
		handler.queueSize = queueSize
		handler.workerCount = workers
	}
}

// WebhookHandler 处理 KOOK Webhook 验证、解密、解压、去重和事件分发。
type WebhookHandler struct {
	client          *Client
	encryptKey      string
	verifyToken     string
	dispatcher      *eventDispatcher
	deduplicator    WebhookDeduplicator
	dedupTTL        time.Duration
	maxBodyBytes    int64
	maxDecodedBytes int64
	queueSize       int
	workerCount     int
	configErr       error
	dispatchQueue   chan *Event
	dispatchSlots   chan struct{}
	queueMu         sync.RWMutex
	closed          bool
	workerWG        sync.WaitGroup
	workersDone     chan struct{}
	serverMu        sync.Mutex
	server          *http.Server
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
	handler, err := newWebhookHandler(client, encryptKey, verifyToken, options...)
	if err != nil {
		handler.configErr = err
	}
	return handler
}

// NewWebhookHandlerWithError 创建经过完整配置校验的Webhook处理器。
func NewWebhookHandlerWithError(client *Client, encryptKey, verifyToken string, options ...WebhookOption) (*WebhookHandler, error) {
	handler, err := newWebhookHandler(client, encryptKey, verifyToken, options...)
	if err != nil {
		return nil, err
	}
	return handler, nil
}

func newWebhookHandler(client *Client, encryptKey, verifyToken string, options ...WebhookOption) (*WebhookHandler, error) {
	handler := &WebhookHandler{
		client:          client,
		encryptKey:      encryptKey,
		verifyToken:     verifyToken,
		dispatcher:      newEventDispatcher(),
		deduplicator:    NewMemoryWebhookDeduplicator(defaultWebhookDedupMax),
		dedupTTL:        defaultWebhookDedupTTL,
		maxBodyBytes:    defaultWebhookMaxBody,
		maxDecodedBytes: defaultWebhookMaxData,
		queueSize:       defaultWebhookQueue,
		workerCount:     defaultWebhookWorkers,
		workersDone:     make(chan struct{}),
	}
	for _, option := range options {
		if option == nil {
			return handler, fmt.Errorf("Webhook配置选项不能为空")
		}
		option(handler)
	}
	if client == nil || client.logger == nil || client.closed.Load() {
		return handler, fmt.Errorf("Webhook客户端不能为空")
	}
	if strings.TrimSpace(verifyToken) == "" {
		return handler, fmt.Errorf("webhook verifyToken不能为空")
	}
	if encryptKey != "" && len(encryptKey) != 16 && len(encryptKey) != 24 && len(encryptKey) != 32 {
		return handler, fmt.Errorf("webhook encryptKey长度必须为16、24或32字节")
	}
	if handler.deduplicator == nil {
		return handler, fmt.Errorf("Webhook去重器不能为空")
	}
	if handler.dedupTTL <= 0 {
		return handler, fmt.Errorf("Webhook去重TTL必须大于0")
	}
	if handler.maxBodyBytes <= 0 || handler.maxDecodedBytes <= 0 {
		return handler, fmt.Errorf("Webhook请求体限制无效")
	}
	if handler.queueSize <= 0 || handler.workerCount != 1 {
		return handler, fmt.Errorf("Webhook派发配置无效")
	}
	handler.dispatchQueue = make(chan *Event, handler.queueSize)
	handler.dispatchSlots = make(chan struct{}, handler.queueSize)
	for index := 0; index < handler.workerCount; index++ {
		handler.workerWG.Add(1)
		go handler.runWorker()
	}
	go func() {
		handler.workerWG.Wait()
		close(handler.workersDone)
	}()
	return handler, nil
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

// OnEvent 保留 v1.1.1 的数字事件注册方式。
func (wh *WebhookHandler) OnEvent(eventType int, handler EventHandler) {
	if handler == nil {
		return
	}
	wh.OnAnyEvent(func(event *Event) {
		if int(event.Type) == eventType {
			handler(event)
		}
	})
}

func (wh *WebhookHandler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if wh == nil || wh.configErr != nil {
		http.Error(w, "Webhook Unavailable", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, wh.maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		wh.client.logger.Error("读取Webhook请求体失败")
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	body, err = decodeRequestBody(body, r.Header.Get("Content-Encoding"), wh.maxDecodedBytes)
	if err != nil {
		wh.client.logger.Error("解码Webhook请求体失败")
		if errors.Is(err, ErrWebhookBodyTooLarge) {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	body, err = wh.tryDecryptBody(body)
	if err != nil {
		wh.client.logger.Error("解密Webhook请求体失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if int64(len(body)) > wh.maxDecodedBytes {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return
	}

	var msg WebhookMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		wh.client.logger.Error("解析Webhook消息失败")
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	challenge, err := wh.handleMessage(r.Context(), &msg)
	if err != nil {
		wh.client.logger.Error("处理Webhook消息失败")
		if errors.Is(err, ErrWebhookVerifyToken) {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if errors.Is(err, ErrWebhookQueueFull) || errors.Is(err, ErrWebhookClosed) {
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
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

func decodeRequestBody(body []byte, encoding string, limit int64) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "", "identity":
		if json.Valid(body) {
			return body, nil
		}
		if decoded, err := readZlib(body, limit); err == nil {
			return decoded, nil
		} else if errors.Is(err, ErrWebhookBodyTooLarge) {
			return nil, err
		}
		if len(body) >= 2 && body[0] == 0x1f && body[1] == 0x8b {
			return readGzip(body, limit)
		}
		return body, nil
	case "gzip":
		return readGzip(body, limit)
	case "deflate":
		return readZlib(body, limit)
	default:
		return nil, fmt.Errorf("不支持的Content-Encoding: %s", encoding)
	}
}

func readGzip(body []byte, limit int64) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return readLimited(reader, limit)
}

func readZlib(body []byte, limit int64) ([]byte, error) {
	reader, err := zlib.NewReader(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return readLimited(reader, limit)
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, ErrWebhookBodyTooLarge
	}
	return data, nil
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
	if meta.VerifyToken != wh.verifyToken {
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

	wh.queueMu.RLock()
	if wh.closed {
		wh.queueMu.RUnlock()
		return ErrWebhookClosed
	}
	select {
	case wh.dispatchSlots <- struct{}{}:
	default:
		wh.queueMu.RUnlock()
		return ErrWebhookQueueFull
	}

	duplicate, err := wh.deduplicator.CheckAndStore(ctx, msg.SN, wh.dedupTTL)
	if err != nil {
		<-wh.dispatchSlots
		wh.queueMu.RUnlock()
		return fmt.Errorf("检查Webhook事件SN失败: %w", err)
	}
	if duplicate {
		<-wh.dispatchSlots
		wh.queueMu.RUnlock()
		wh.client.logger.Debugf("忽略重复Webhook事件SN: %d", msg.SN)
		return nil
	}
	wh.dispatchQueue <- &event
	wh.queueMu.RUnlock()
	return nil
}

func (wh *WebhookHandler) runWorker() {
	defer wh.workerWG.Done()
	for event := range wh.dispatchQueue {
		err := wh.dispatcher.dispatch(event, func(recovered any) {
			wh.client.logger.Error("事件处理器发生panic")
		})
		if err != nil {
			wh.client.logger.Error("分发Webhook事件失败")
		}
		<-wh.dispatchSlots
	}
}

func (wh *WebhookHandler) StartWebhookServer(addr, path string) error {
	if wh == nil {
		return fmt.Errorf("Webhook处理器不能为空")
	}
	if wh.configErr != nil {
		return fmt.Errorf("webhook配置无效: %w", wh.configErr)
	}
	if !strings.HasPrefix(path, "/") {
		return fmt.Errorf("Webhook路径必须以/开头")
	}
	wh.queueMu.RLock()
	closed := wh.closed
	wh.queueMu.RUnlock()
	if closed {
		return ErrWebhookClosed
	}
	mux := http.NewServeMux()
	mux.HandleFunc(path, wh.HandleRequest)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	wh.serverMu.Lock()
	if wh.server != nil {
		wh.serverMu.Unlock()
		return fmt.Errorf("Webhook服务器已经启动")
	}
	wh.server = server
	wh.serverMu.Unlock()
	defer func() {
		wh.serverMu.Lock()
		if wh.server == server {
			wh.server = nil
		}
		wh.serverMu.Unlock()
	}()
	wh.client.logger.Infof("启动Webhook服务器: %s%s", addr, path)
	err := server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Shutdown 停止接收新事件、关闭内建服务器并等待已接收事件完成。
func (wh *WebhookHandler) Shutdown(ctx context.Context) error {
	if wh == nil {
		return nil
	}
	if ctx == nil {
		return fmt.Errorf("shutdown context不能为空")
	}
	wh.serverMu.Lock()
	server := wh.server
	wh.serverMu.Unlock()
	if server != nil {
		if err := server.Shutdown(ctx); err != nil {
			return err
		}
	}

	wh.queueMu.Lock()
	if !wh.closed {
		wh.closed = true
		if wh.dispatchQueue != nil {
			close(wh.dispatchQueue)
		}
	}
	wh.queueMu.Unlock()
	if wh.workersDone == nil || wh.dispatchQueue == nil {
		return wh.configErr
	}
	select {
	case <-wh.workersDone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
