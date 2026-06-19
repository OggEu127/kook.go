package kook

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// EventHandler 事件处理器函数类型
type EventHandler func(*Event)

// WebSocketOptions WebSocket客户端配置。
type WebSocketOptions struct {
	ReadLimit      int64
	ReadTimeout    time.Duration
	HelloTimeout   time.Duration
	PongTimeout    time.Duration
	MaxEventBuffer int
	EventBufferTTL time.Duration
}

// DefaultWebSocketOptions 返回默认WebSocket配置。
func DefaultWebSocketOptions() WebSocketOptions {
	return WebSocketOptions{
		ReadLimit:      8 << 20,
		ReadTimeout:    90 * time.Second,
		HelloTimeout:   6 * time.Second,
		PongTimeout:    6 * time.Second,
		MaxEventBuffer: 1024,
		EventBufferTTL: 30 * time.Second,
	}
}

// ErrEventBufferFull 表示WebSocket乱序事件缓冲区已满。
var ErrEventBufferFull = errors.New("WebSocket事件缓冲区已满")

type bufferedEvent struct {
	msg     *WebSocketMessage
	created time.Time
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	client          *Client
	conn            *websocket.Conn
	eventHandlers   map[int][]EventHandler
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	compress        bool
	sn              int
	sessionID       string
	stateMu         sync.RWMutex
	heartbeatTicker *time.Ticker
	gatewayURL      string
	reconnectCount  int
	maxReconnects   int
	reconnectDelay  time.Duration
	isConnected     bool
	connMu          sync.RWMutex
	writeMu         sync.Mutex
	options         WebSocketOptions
	pendingPingSN   int
	pendingPingMu   sync.Mutex
	eventBuffer     map[int]bufferedEvent
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	S  int             `json:"s"`           // 信令类型
	D  json.RawMessage `json:"d,omitempty"` // 数据
	SN int             `json:"sn"`          // 序号
}

// HelloMessage Hello消息
type HelloMessage struct {
	Code      int    `json:"code"`
	SessionID string `json:"session_id"`
}

// PingMessage Ping消息
type PingMessage struct {
	SN int `json:"sn"`
}

// PongMessage Pong消息
type PongMessage struct {
	SN int `json:"sn"`
}

// ResumeMessage Resume消息
type ResumeMessage struct {
	SessionID string `json:"session_id"`
	SN        int    `json:"sn"`
}

// 信令类型常量
const (
	SignalEvent     = 0 // 事件
	SignalHello     = 1 // 服务端发送，客户端接收，代表连接成功
	SignalPing      = 2 // 双向：服务端ping客户端，客户端也可以ping服务端
	SignalPong      = 3 // 双向：ping的响应
	SignalResume    = 4 // 客户端发送，服务端接收，代表重连
	SignalReconnect = 5 // 服务端发送，客户端接收，代表需要重连
	SignalResumeAck = 6 // 服务端发送，客户端接收，代表重连成功
)

// NewWebSocketClient 创建新的WebSocket客户端
func NewWebSocketClient(client *Client, compress bool, options ...WebSocketOptions) *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())
	wsOptions := DefaultWebSocketOptions()
	if len(options) > 0 {
		wsOptions = mergeWebSocketOptions(wsOptions, options[0])
	}

	return &WebSocketClient{
		client:         client,
		eventHandlers:  make(map[int][]EventHandler),
		ctx:            ctx,
		cancel:         cancel,
		compress:       compress,
		maxReconnects:  10,
		reconnectDelay: 5 * time.Second,
		options:        wsOptions,
		eventBuffer:    make(map[int]bufferedEvent),
	}
}

func mergeWebSocketOptions(defaults, override WebSocketOptions) WebSocketOptions {
	if override.ReadLimit != 0 {
		defaults.ReadLimit = override.ReadLimit
	}
	if override.ReadTimeout != 0 {
		defaults.ReadTimeout = override.ReadTimeout
	}
	if override.HelloTimeout != 0 {
		defaults.HelloTimeout = override.HelloTimeout
	}
	if override.PongTimeout != 0 {
		defaults.PongTimeout = override.PongTimeout
	}
	if override.MaxEventBuffer != 0 {
		defaults.MaxEventBuffer = override.MaxEventBuffer
	}
	if override.EventBufferTTL != 0 {
		defaults.EventBufferTTL = override.EventBufferTTL
	}
	return defaults
}

// OnEvent 注册事件处理器
func (ws *WebSocketClient) OnEvent(eventType int, handler EventHandler) {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	ws.eventHandlers[eventType] = append(ws.eventHandlers[eventType], handler)
}

// Connect 连接到WebSocket网关
func (ws *WebSocketClient) Connect() error {
	return ws.connectWithRetry()
}

// connectWithRetry 带重试的连接
func (ws *WebSocketClient) connectWithRetry() error {
	for attempts := 0; attempts <= ws.maxReconnects; attempts++ {
		err := ws.doConnect()
		if err == nil {
			ws.setReconnectCount(0)
			return nil
		}

		ws.client.logger.WithError(err).Errorf("WebSocket连接失败，尝试 %d/%d", attempts+1, ws.maxReconnects+1)

		if attempts < ws.maxReconnects {
			select {
			case <-ws.ctx.Done():
				return ws.ctx.Err()
			case <-time.After(ws.reconnectDelay * time.Duration(attempts+1)):
				// 指数退避
			}
		}
	}

	return fmt.Errorf("WebSocket连接失败，已达到最大重试次数")
}

// doConnect 执行实际连接
func (ws *WebSocketClient) doConnect() error {
	compress := 0
	if ws.compress {
		compress = 1
	}

	connectURL := ws.resumeGatewayURL()
	if connectURL == "" {
		gateway, err := ws.client.Gateway.GetGateway(ws.ctx, compress)
		if err != nil {
			return fmt.Errorf("获取网关信息失败: %w", err)
		}
		ws.gatewayURL = gateway.URL
		connectURL = gateway.URL
	}

	// 创建WebSocket连接
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("%s %s", ws.client.tokenType, ws.client.token))

	ws.client.logger.Infof("连接到WebSocket网关: %s", connectURL)

	dialCtx := ws.ctx
	var cancel context.CancelFunc
	if ws.options.HelloTimeout > 0 {
		dialCtx, cancel = context.WithTimeout(ws.ctx, ws.options.HelloTimeout)
		defer cancel()
	}

	conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, connectURL, header)
	if err != nil {
		return fmt.Errorf("WebSocket连接失败: %w", err)
	}
	ws.configureRead(conn)

	ws.connMu.Lock()
	ws.conn = conn
	ws.isConnected = true
	ws.connMu.Unlock()

	ws.client.logger.Info("WebSocket连接成功")

	// 启动消息处理协程
	go ws.handleMessages()

	return nil
}

func (ws *WebSocketClient) configureRead(conn *websocket.Conn) {
	if conn == nil {
		return
	}
	if ws.options.ReadLimit > 0 {
		conn.SetReadLimit(ws.options.ReadLimit)
	}
	ws.refreshReadDeadline(conn)
}

func (ws *WebSocketClient) refreshReadDeadline(conn *websocket.Conn) {
	if conn == nil || ws.options.ReadTimeout <= 0 {
		return
	}
	_ = conn.SetReadDeadline(time.Now().Add(ws.options.ReadTimeout))
}

func (ws *WebSocketClient) resumeGatewayURL() string {
	sessionID := ws.getSessionID()
	sn := ws.getSN()
	if ws.gatewayURL == "" || sessionID == "" || sn <= 0 {
		return ""
	}

	u, err := url.Parse(ws.gatewayURL)
	if err != nil {
		ws.client.logger.WithError(err).Warn("解析WebSocket resume URL失败，将重新获取gateway")
		return ""
	}

	q := u.Query()
	q.Set("resume", "1")
	q.Set("sn", strconv.Itoa(sn))
	q.Set("session_id", sessionID)
	u.RawQuery = q.Encode()
	return u.String()
}

// Close 关闭WebSocket连接
func (ws *WebSocketClient) Close() error {
	ws.cancel()

	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
	}
	ws.clearPendingPing()

	ws.connMu.RLock()
	conn := ws.conn
	ws.connMu.RUnlock()

	if conn != nil {
		return conn.Close()
	}

	return nil
}

// handleMessages 处理WebSocket消息
func (ws *WebSocketClient) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			ws.client.logger.Errorf("WebSocket消息处理发生panic: %v", r)
		}

		// 标记连接已断开
		ws.connMu.Lock()
		ws.isConnected = false
		ws.connMu.Unlock()

		// 主动关闭时不重连
		if ws.ctx.Err() == nil {
			ws.attemptReconnect()
		}
	}()

	for {
		select {
		case <-ws.ctx.Done():
			return
		default:
			ws.connMu.RLock()
			conn := ws.conn
			ws.connMu.RUnlock()

			if conn == nil {
				ws.client.logger.Error("WebSocket连接为空")
				return
			}

			_, data, err := conn.ReadMessage()
			if err != nil {
				ws.client.logger.WithError(err).Error("读取WebSocket消息失败")
				return
			}
			ws.refreshReadDeadline(conn)

			// 如果启用了压缩，需要解压
			if ws.compress {
				data, err = ws.decompress(data)
				if err != nil {
					ws.client.logger.WithError(err).Error("解压消息失败")
					continue
				}
			}

			ws.client.logger.Debugf("收到WebSocket消息: %s", string(data))

			var msg WebSocketMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				ws.client.logger.WithError(err).Error("解析WebSocket消息失败")
				continue
			}

			if err := ws.handleMessage(&msg); err != nil {
				ws.client.logger.WithError(err).Error("处理WebSocket消息失败")
			}
		}
	}
}

// attemptReconnect 尝试重连
func (ws *WebSocketClient) attemptReconnect() {
	if ws.ctx.Err() != nil {
		return
	}
	reconnectCount := ws.getReconnectCount()
	if reconnectCount >= ws.maxReconnects {
		ws.client.logger.Error("已达到最大重连次数，停止重连")
		return
	}

	reconnectCount = ws.incrementReconnectCount()
	ws.client.logger.Infof("开始第 %d 次重连尝试", reconnectCount)

	// 等待一段时间后重连
	delay := ws.reconnectDelay * time.Duration(reconnectCount)
	select {
	case <-time.After(delay):
	case <-ws.ctx.Done():
		return
	}

	err := ws.doConnect()
	if err != nil {
		ws.client.logger.WithError(err).Errorf("重连失败")
		// 递归尝试重连
		go ws.attemptReconnect()
	} else {
		ws.client.logger.Info("重连成功")
		ws.setReconnectCount(0)
	}
}

// IsConnected 检查连接状态
func (ws *WebSocketClient) IsConnected() bool {
	ws.connMu.RLock()
	defer ws.connMu.RUnlock()
	return ws.isConnected
}

// handleMessage 处理单个WebSocket消息
func (ws *WebSocketClient) handleMessage(msg *WebSocketMessage) error {
	switch msg.S {
	case SignalEvent:
		// 处理事件
		return ws.handleEvent(msg)
	case SignalHello:
		// 处理Hello消息
		return ws.handleHello(msg)
	case SignalPing:
		// 处理Ping消息
		return ws.handlePing(msg)
	case SignalReconnect:
		// 处理重连消息
		return ws.handleReconnect(msg)
	case SignalResumeAck:
		// 处理重连确认消息
		return ws.handleResumeAck(msg)
	case SignalPong:
		// 处理Pong消息
		pongSN := msg.SN
		if pongSN == 0 && msg.D != nil {
			var pong PongMessage
			if err := json.Unmarshal(msg.D, &pong); err != nil {
				ws.client.logger.WithError(err).Debug("解析Pong消息失败，可能是空的Pong")
			} else {
				pongSN = pong.SN
			}
		}
		ws.client.logger.Debugf("收到Pong响应，SN: %d", pongSN)
		ws.ackPendingPing(pongSN)
		return nil
	default:
		ws.client.logger.Warnf("收到未知信令类型: %d", msg.S)
	}

	return nil
}

// handleEvent 处理事件消息
func (ws *WebSocketClient) handleEvent(msg *WebSocketMessage) error {
	if msg.SN > 0 {
		return ws.handleOrderedEvent(msg)
	}

	event, err := ws.decodeEvent(msg)
	if err != nil {
		return err
	}
	ws.dispatchDecodedEvent(event)
	return nil
}

func (ws *WebSocketClient) handleOrderedEvent(msg *WebSocketMessage) error {
	ws.stateMu.Lock()
	ws.expireEventBufferLocked(time.Now())
	currentSN := ws.sn
	if msg.SN <= currentSN {
		ws.stateMu.Unlock()
		ws.client.logger.Debugf("丢弃旧事件SN: %d，当前SN: %d", msg.SN, currentSN)
		return nil
	}
	if msg.SN > currentSN+1 {
		if _, exists := ws.eventBuffer[msg.SN]; exists {
			ws.stateMu.Unlock()
			ws.client.logger.Debugf("忽略重复缓冲事件SN: %d", msg.SN)
			return nil
		}
		if ws.options.MaxEventBuffer > 0 && len(ws.eventBuffer) >= ws.options.MaxEventBuffer {
			ws.stateMu.Unlock()
			return ErrEventBufferFull
		}
		ws.eventBuffer[msg.SN] = bufferedEvent{msg: msg, created: time.Now()}
		ws.stateMu.Unlock()
		ws.client.logger.Warnf("收到乱序事件SN: %d，当前SN: %d，已缓冲", msg.SN, currentSN)
		return nil
	}
	ws.stateMu.Unlock()

	if err := ws.dispatchOrderedEvent(msg); err != nil {
		return err
	}

	for {
		ws.stateMu.Lock()
		ws.expireEventBufferLocked(time.Now())
		nextSN := ws.sn + 1
		next, ok := ws.eventBuffer[nextSN]
		if !ok {
			ws.stateMu.Unlock()
			return nil
		}
		delete(ws.eventBuffer, nextSN)
		ws.stateMu.Unlock()

		if err := ws.dispatchOrderedEvent(next.msg); err != nil {
			return err
		}
	}
}

func (ws *WebSocketClient) expireEventBufferLocked(now time.Time) {
	if ws.options.EventBufferTTL <= 0 || len(ws.eventBuffer) == 0 {
		return
	}
	for sn, event := range ws.eventBuffer {
		if now.Sub(event.created) > ws.options.EventBufferTTL {
			delete(ws.eventBuffer, sn)
			if sn > ws.sn {
				ws.sn = sn
			}
			ws.client.logger.Warnf("丢弃过期缓冲事件SN: %d", sn)
		}
	}
}

func (ws *WebSocketClient) dispatchOrderedEvent(msg *WebSocketMessage) error {
	event, err := ws.decodeEvent(msg)
	if err != nil {
		return err
	}

	if msg.SN > 0 {
		ws.setSN(msg.SN)
	}
	ws.dispatchDecodedEvent(event)
	return nil
}

func (ws *WebSocketClient) decodeEvent(msg *WebSocketMessage) (*Event, error) {
	var event Event
	if err := json.Unmarshal(msg.D, &event); err != nil {
		return nil, fmt.Errorf("解析事件失败: %w", err)
	}
	return &event, nil
}

func (ws *WebSocketClient) dispatchDecodedEvent(event *Event) {
	ws.client.logger.Debugf("收到事件: 类型=%d, 内容=%s", event.Type, event.Content)

	// 调用事件处理器
	ws.mu.RLock()
	handlers := append([]EventHandler(nil), ws.eventHandlers[event.Type]...)
	ws.mu.RUnlock()

	for _, handler := range handlers {
		func(h EventHandler) {
			defer func() {
				if r := recover(); r != nil {
					ws.client.logger.Errorf("事件处理器发生panic: %v", r)
				}
			}()
			h(event)
		}(handler)
	}
}

// handleHello 处理Hello消息
func (ws *WebSocketClient) handleHello(msg *WebSocketMessage) error {
	var hello HelloMessage
	if err := json.Unmarshal(msg.D, &hello); err != nil {
		return fmt.Errorf("解析Hello消息失败: %w", err)
	}

	ws.setSessionID(hello.SessionID)
	ws.client.logger.Infof("WebSocket会话建立成功: %s", hello.SessionID)

	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
	}

	// 启动心跳
	ws.startHeartbeat()

	return nil
}

// handlePing 处理Ping消息
func (ws *WebSocketClient) handlePing(msg *WebSocketMessage) error {
	pingSN := msg.SN
	if pingSN == 0 && msg.D != nil {
		var ping PingMessage
		if err := json.Unmarshal(msg.D, &ping); err != nil {
			return fmt.Errorf("解析Ping消息失败: %w", err)
		}
		pingSN = ping.SN
	}

	// 发送Pong响应
	pong := WebSocketMessage{
		S:  SignalPong,
		SN: pingSN,
	}

	return ws.sendMessage(&pong)
}

// handleReconnect 处理重连消息
func (ws *WebSocketClient) handleReconnect(msg *WebSocketMessage) error {
	ws.client.logger.Warn("服务器要求重连")
	ws.resetSessionStateForFreshReconnect()

	ws.connMu.Lock()
	if ws.conn != nil {
		_ = ws.conn.Close()
	}
	ws.isConnected = false
	ws.connMu.Unlock()

	return nil
}

// handleResumeAck 处理重连确认消息
func (ws *WebSocketClient) handleResumeAck(msg *WebSocketMessage) error {
	ws.client.logger.Info("重连成功")
	return nil
}

// startHeartbeat 启动心跳
func (ws *WebSocketClient) startHeartbeat() {
	// 每30秒发送一次心跳
	ws.heartbeatTicker = time.NewTicker(30 * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ws.client.logger.Errorf("心跳处理发生panic: %v", r)
			}
		}()

		consecutiveFailures := 0
		const maxFailures = 3

		for {
			select {
			case <-ws.ctx.Done():
				return
			case <-ws.heartbeatTicker.C:
				ping := WebSocketMessage{
					S:  SignalPing,
					SN: ws.getSN(),
				}

				if err := ws.sendMessage(&ping); err != nil {
					consecutiveFailures++
					ws.client.logger.WithError(err).Errorf("发送心跳失败 (%d/%d)", consecutiveFailures, maxFailures)

					if consecutiveFailures >= maxFailures {
						ws.client.logger.Error("连续心跳失败，触发重连")
						ws.connMu.Lock()
						ws.isConnected = false
						if ws.conn != nil {
							ws.conn.Close()
						}
						ws.connMu.Unlock()
						go ws.attemptReconnect()
						return
					}
				} else {
					ws.trackPendingPing(ping.SN)
					if consecutiveFailures > 0 {
						ws.client.logger.Info("心跳恢复正常")
					}
					consecutiveFailures = 0
				}
			}
		}
	}()
}

func (ws *WebSocketClient) trackPendingPing(sn int) {
	ws.pendingPingMu.Lock()
	ws.pendingPingSN = sn
	ws.pendingPingMu.Unlock()

	timeout := ws.options.PongTimeout
	if timeout <= 0 {
		return
	}

	time.AfterFunc(timeout, func() {
		ws.pendingPingMu.Lock()
		pending := ws.pendingPingSN == sn
		ws.pendingPingMu.Unlock()
		if !pending || ws.ctx.Err() != nil {
			return
		}

		ws.client.logger.Errorf("WebSocket Pong超时，SN: %d", sn)
		ws.connMu.Lock()
		ws.isConnected = false
		if ws.conn != nil {
			_ = ws.conn.Close()
		}
		ws.connMu.Unlock()
	})
}

func (ws *WebSocketClient) clearPendingPing() {
	ws.pendingPingMu.Lock()
	ws.pendingPingSN = 0
	ws.pendingPingMu.Unlock()
}

func (ws *WebSocketClient) ackPendingPing(sn int) {
	ws.pendingPingMu.Lock()
	if ws.pendingPingSN == sn || sn == 0 {
		ws.pendingPingSN = 0
	}
	ws.pendingPingMu.Unlock()
}

// sendMessage 发送WebSocket消息
func (ws *WebSocketClient) sendMessage(msg *WebSocketMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	ws.client.logger.Debugf("发送WebSocket消息: %s", string(data))

	ws.connMu.RLock()
	conn := ws.conn
	ws.connMu.RUnlock()
	if conn == nil {
		return fmt.Errorf("WebSocket连接不可用")
	}

	ws.writeMu.Lock()
	defer ws.writeMu.Unlock()

	return conn.WriteMessage(websocket.TextMessage, data)
}

func (ws *WebSocketClient) setSN(sn int) {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.sn = sn
}

func (ws *WebSocketClient) getSN() int {
	ws.stateMu.RLock()
	defer ws.stateMu.RUnlock()
	return ws.sn
}

func (ws *WebSocketClient) setSessionID(sessionID string) {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.sessionID = sessionID
}

func (ws *WebSocketClient) getSessionID() string {
	ws.stateMu.RLock()
	defer ws.stateMu.RUnlock()
	return ws.sessionID
}

func (ws *WebSocketClient) setReconnectCount(count int) {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.reconnectCount = count
}

func (ws *WebSocketClient) resetSessionStateForFreshReconnect() {
	ws.stateMu.Lock()
	ws.sn = 0
	ws.sessionID = ""
	ws.gatewayURL = ""
	ws.eventBuffer = make(map[int]bufferedEvent)
	ws.stateMu.Unlock()
	ws.clearPendingPing()
}

func (ws *WebSocketClient) getReconnectCount() int {
	ws.stateMu.RLock()
	defer ws.stateMu.RUnlock()
	return ws.reconnectCount
}

func (ws *WebSocketClient) incrementReconnectCount() int {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.reconnectCount++
	return ws.reconnectCount
}

// decompress 解压数据
func (ws *WebSocketClient) decompress(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return io.ReadAll(r)
}
