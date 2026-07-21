package kook

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketOptions WebSocket客户端配置。
type WebSocketOptions struct {
	ReadLimit         int64
	ReadTimeout       time.Duration
	HelloTimeout      time.Duration
	PongTimeout       time.Duration
	MaxEventBuffer    int
	EventBufferTTL    time.Duration
	DispatchQueueSize int
}

// DefaultWebSocketOptions 返回默认WebSocket配置。
func DefaultWebSocketOptions() WebSocketOptions {
	return WebSocketOptions{
		ReadLimit:         8 << 20,
		ReadTimeout:       90 * time.Second,
		HelloTimeout:      6 * time.Second,
		PongTimeout:       6 * time.Second,
		MaxEventBuffer:    1024,
		EventBufferTTL:    30 * time.Second,
		DispatchQueueSize: 256,
	}
}

// ErrEventBufferFull 表示WebSocket乱序事件缓冲区已满。
var ErrEventBufferFull = errors.New("WebSocket事件缓冲区已满")

// ErrEventDispatchQueueFull 表示业务处理器持续过慢，有界派发队列已满。
var ErrEventDispatchQueueFull = errors.New("WebSocket事件派发队列已满")

// ErrEventSequenceGap 表示事件序列缺口在TTL内没有补齐，需要重连重放。
var ErrEventSequenceGap = errors.New("WebSocket事件序列存在缺口")

type queuedWebSocketEvent struct {
	event      *Event
	ordered    bool
	generation uint64
}

type bufferedEvent struct {
	msg     *WebSocketMessage
	created time.Time
}

// WebSocketClient WebSocket客户端
type WebSocketClient struct {
	client          *Client
	conn            *websocket.Conn
	dispatcher      *eventDispatcher
	dispatchQueue   chan queuedWebSocketEvent
	ctx             context.Context
	cancel          context.CancelFunc
	compress        bool
	sn              int
	receivedSN      int
	sessionID       string
	generation      uint64
	stateMu         sync.RWMutex
	reconnecting    bool
	heartbeatTicker *time.Ticker
	heartbeatCancel context.CancelFunc
	heartbeatMu     sync.Mutex
	gatewayURL      string
	reconnectCount  int
	maxReconnects   int
	reconnectDelay  time.Duration
	isConnected     bool
	connMu          sync.RWMutex
	writeMu         sync.Mutex
	options         WebSocketOptions
	pendingPingSN   int
	pendingPing     bool
	pendingPingMu   sync.Mutex
	eventBuffer     map[int]bufferedEvent
	configErr       error
	connectMu       sync.Mutex
	connecting      bool
	closeOnce       sync.Once
	gapTimerMu      sync.Mutex
	gapTimer        *time.Timer
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
	ws, err := newWebSocketClient(client, compress, options...)
	if err != nil {
		ws.configErr = err
		ws.cancel()
	}
	return ws
}

// NewWebSocketClientWithError 创建经过完整配置校验的WebSocket客户端。
func NewWebSocketClientWithError(client *Client, compress bool, options ...WebSocketOptions) (*WebSocketClient, error) {
	ws, err := newWebSocketClient(client, compress, options...)
	if err != nil {
		return nil, err
	}
	return ws, nil
}

func newWebSocketClient(client *Client, compress bool, options ...WebSocketOptions) (*WebSocketClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	wsOptions := DefaultWebSocketOptions()
	ws := &WebSocketClient{
		client:         client,
		dispatcher:     newEventDispatcher(),
		ctx:            ctx,
		cancel:         cancel,
		compress:       compress,
		maxReconnects:  10,
		reconnectDelay: 5 * time.Second,
		options:        wsOptions,
		eventBuffer:    make(map[int]bufferedEvent),
	}
	if len(options) > 1 {
		cancel()
		return ws, fmt.Errorf("只能提供一组WebSocket配置")
	}
	if len(options) > 0 {
		wsOptions = mergeWebSocketOptions(wsOptions, options[0])
		ws.options = wsOptions
	}
	if client == nil || client.logger == nil || client.closed.Load() {
		cancel()
		return ws, fmt.Errorf("WebSocket客户端不能为空")
	}
	if err := validateWebSocketOptions(wsOptions); err != nil {
		cancel()
		return ws, err
	}
	ws.dispatchQueue = make(chan queuedWebSocketEvent, wsOptions.DispatchQueueSize)
	go ws.runEventDispatcher()
	return ws, nil
}

func validateWebSocketOptions(options WebSocketOptions) error {
	if options.ReadLimit <= 0 || options.ReadTimeout <= 0 || options.HelloTimeout <= 0 || options.PongTimeout <= 0 {
		return fmt.Errorf("WebSocket读取和超时配置必须大于0")
	}
	if options.MaxEventBuffer <= 0 || options.EventBufferTTL <= 0 || options.DispatchQueueSize <= 0 {
		return fmt.Errorf("WebSocket缓冲区配置必须大于0")
	}
	return nil
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
	if override.DispatchQueueSize != 0 {
		defaults.DispatchQueueSize = override.DispatchQueueSize
	}
	return defaults
}

// OnMessage 注册普通消息处理器。
func (ws *WebSocketClient) OnMessage(eventType MessageType, handler MessageEventHandler) {
	ws.dispatcher.onMessage(eventType, handler)
}

// OnSystemEvent 按 extra.type 注册系统事件处理器。
func (ws *WebSocketClient) OnSystemEvent(eventType SystemEventType, handler SystemEventHandler) {
	ws.dispatcher.onSystemEvent(eventType, handler)
}

// OnAnyEvent 注册所有原始事件的处理器。
func (ws *WebSocketClient) OnAnyEvent(handler RawEventHandler) {
	ws.dispatcher.onAnyEvent(handler)
}

// OnEvent 保留 v1.1.1 的数字事件注册方式。
func (ws *WebSocketClient) OnEvent(eventType int, handler EventHandler) {
	if handler == nil {
		return
	}
	ws.OnAnyEvent(func(event *Event) {
		if int(event.Type) == eventType {
			handler(event)
		}
	})
}

// Connect 连接到WebSocket网关
func (ws *WebSocketClient) Connect() error {
	if ws == nil {
		return fmt.Errorf("WebSocket客户端不能为空")
	}
	if ws.configErr != nil {
		return ws.configErr
	}
	ws.connectMu.Lock()
	if ws.IsConnected() {
		ws.connectMu.Unlock()
		return nil
	}
	if ws.connecting {
		ws.connectMu.Unlock()
		return fmt.Errorf("WebSocket正在连接")
	}
	ws.stateMu.RLock()
	reconnecting := ws.reconnecting
	ws.stateMu.RUnlock()
	if reconnecting {
		ws.connectMu.Unlock()
		return fmt.Errorf("WebSocket正在重连")
	}
	if ws.ctx.Err() != nil {
		ws.connectMu.Unlock()
		return ws.ctx.Err()
	}
	ws.connecting = true
	ws.connectMu.Unlock()
	defer func() {
		ws.connectMu.Lock()
		ws.connecting = false
		ws.connectMu.Unlock()
	}()
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

		ws.client.logger.Errorf("WebSocket连接失败，尝试 %d/%d", attempts+1, ws.maxReconnects+1)

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
		gateway, err := ws.client.Gateway.GetGateway(ws.ctx, GatewayParams{Compress: &compress})
		if err != nil {
			return fmt.Errorf("获取网关信息失败: %w", err)
		}
		ws.setGatewayURL(gateway.URL)
		connectURL = gateway.URL
	}

	// 创建WebSocket连接
	header := http.Header{}
	header.Set("Authorization", fmt.Sprintf("%s %s", ws.client.tokenType, ws.client.token))

	ws.client.logger.Infof("连接到WebSocket网关: %s", sanitizedURL(connectURL))

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
	if err := ws.waitHello(conn); err != nil {
		_ = conn.Close()
		return err
	}

	ws.connMu.Lock()
	if ws.ctx.Err() != nil {
		ws.connMu.Unlock()
		_ = conn.Close()
		return ws.ctx.Err()
	}
	ws.conn = conn
	ws.isConnected = true
	ws.connMu.Unlock()
	ws.activateConnectionGeneration()

	ws.client.logger.Info("WebSocket连接成功")

	// 启动消息处理协程
	go ws.handleMessages(conn)

	return nil
}

func (ws *WebSocketClient) waitHello(conn *websocket.Conn) error {
	if conn == nil {
		return fmt.Errorf("WebSocket连接不可用")
	}
	if ws.options.ReadLimit > 0 {
		conn.SetReadLimit(ws.options.ReadLimit)
	}
	if ws.options.HelloTimeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(ws.options.HelloTimeout))
	}

	msg, err := ws.readWebSocketMessage(conn)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return fmt.Errorf("等待Hello消息超时: %w", err)
		}
		return fmt.Errorf("读取Hello消息失败: %w", err)
	}
	switch msg.S {
	case SignalHello:
		if err := ws.handleHello(msg); err != nil {
			return err
		}
	case SignalResumeAck:
		if ws.getSessionID() == "" {
			return fmt.Errorf("收到ResumeAck但本地会话为空")
		}
		if err := ws.handleResumeAck(msg); err != nil {
			return err
		}
		ws.startHeartbeat()
	default:
		return fmt.Errorf("首个WebSocket消息信令=%d，期望Hello或ResumeAck信令", msg.S)
	}
	ws.refreshReadDeadline(conn)
	return nil
}

func (ws *WebSocketClient) refreshReadDeadline(conn *websocket.Conn) {
	if conn == nil || ws.options.ReadTimeout <= 0 {
		return
	}
	_ = conn.SetReadDeadline(time.Now().Add(ws.options.ReadTimeout))
}

func (ws *WebSocketClient) resumeGatewayURL() string {
	ws.stateMu.RLock()
	gatewayURL := ws.gatewayURL
	sessionID := ws.sessionID
	sn := ws.sn
	ws.stateMu.RUnlock()
	if gatewayURL == "" || sessionID == "" || sn <= 0 {
		return ""
	}

	u, err := url.Parse(gatewayURL)
	if err != nil {
		ws.client.logger.Warn("解析WebSocket resume URL失败，将重新获取gateway")
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
	if ws == nil {
		return nil
	}
	var closeErr error
	ws.closeOnce.Do(func() {
		ws.cancel()
		ws.stopHeartbeat()
		ws.stopGapTimer()
		ws.clearPendingPing()
		closeErr = ws.closeActiveConnection()
	})
	return closeErr
}

// handleMessages 处理WebSocket消息
func (ws *WebSocketClient) handleMessages(conn *websocket.Conn) {
	defer func() {
		ws.markReconnectPending()
		ws.stopHeartbeat()
		ws.stopGapTimer()
		if r := recover(); r != nil {
			ws.client.logger.Error("WebSocket消息处理发生panic")
		}

		ws.connMu.Lock()
		if ws.conn == conn {
			ws.conn = nil
			ws.isConnected = false
		}
		ws.connMu.Unlock()
		_ = conn.Close()

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
			msg, err := ws.readWebSocketMessage(conn)
			if err != nil {
				ws.client.logger.Error("读取WebSocket消息失败")
				return
			}
			ws.refreshReadDeadline(conn)

			if err := ws.handleMessage(msg); err != nil {
				ws.client.logger.Error("处理WebSocket消息失败")
				return
			}
		}
	}
}

func (ws *WebSocketClient) readWebSocketMessage(conn *websocket.Conn) (*WebSocketMessage, error) {
	_, data, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	if ws.compress {
		data, err = ws.decompress(data)
		if err != nil {
			return nil, fmt.Errorf("解压消息失败: %w", err)
		}
	}

	var msg WebSocketMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("解析WebSocket消息失败: %w", err)
	}
	ws.client.logger.Debugf("收到WebSocket消息: signal=%d sn=%d bytes=%d", msg.S, msg.SN, len(data))
	return &msg, nil
}

// attemptReconnect 尝试重连
func (ws *WebSocketClient) attemptReconnect() {
	ws.connectMu.Lock()
	if ws.connecting {
		ws.connectMu.Unlock()
		return
	}
	ws.connecting = true
	ws.connectMu.Unlock()
	defer func() {
		ws.connectMu.Lock()
		ws.connecting = false
		ws.connectMu.Unlock()
	}()

	ws.stateMu.Lock()
	ws.reconnecting = true
	ws.stateMu.Unlock()
	defer func() {
		ws.stateMu.Lock()
		ws.reconnecting = false
		ws.stateMu.Unlock()
	}()

	for ws.ctx.Err() == nil {
		reconnectCount := ws.incrementReconnectCount()
		if reconnectCount > ws.maxReconnects {
			ws.client.logger.Error("已达到最大重连次数，停止重连")
			return
		}
		ws.client.logger.Infof("开始第 %d 次重连尝试", reconnectCount)

		select {
		case <-time.After(ws.reconnectDelay * time.Duration(reconnectCount)):
		case <-ws.ctx.Done():
			return
		}

		if err := ws.doConnect(); err != nil {
			ws.client.logger.Error("重连失败")
			continue
		}
		ws.client.logger.Info("重连成功")
		ws.setReconnectCount(0)
		return
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
				ws.client.logger.Debug("解析Pong消息失败，可能是空的Pong")
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
	return ws.dispatchDecodedEvent(event)
}

func (ws *WebSocketClient) handleOrderedEvent(msg *WebSocketMessage) error {
	ws.stateMu.Lock()
	if ws.expireEventBufferLocked(time.Now()) {
		ws.stateMu.Unlock()
		return ErrEventSequenceGap
	}
	currentSN := ws.receivedSN
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
		ws.scheduleGapTimer()
		ws.client.logger.Warnf("收到乱序事件SN: %d，当前SN: %d，已缓冲", msg.SN, currentSN)
		return nil
	}
	ws.stateMu.Unlock()

	if err := ws.dispatchOrderedEvent(msg); err != nil {
		return err
	}

	for {
		ws.stateMu.Lock()
		if ws.expireEventBufferLocked(time.Now()) {
			ws.stateMu.Unlock()
			return ErrEventSequenceGap
		}
		nextSN := ws.receivedSN + 1
		next, ok := ws.eventBuffer[nextSN]
		if !ok {
			ws.stateMu.Unlock()
			return nil
		}
		delete(ws.eventBuffer, nextSN)
		bufferEmpty := len(ws.eventBuffer) == 0
		ws.stateMu.Unlock()
		if bufferEmpty {
			ws.stopGapTimer()
		}

		if err := ws.dispatchOrderedEvent(next.msg); err != nil {
			return err
		}
	}
}

func (ws *WebSocketClient) expireEventBufferLocked(now time.Time) bool {
	if ws.options.EventBufferTTL <= 0 || len(ws.eventBuffer) == 0 {
		return false
	}
	expired := false
	for sn, event := range ws.eventBuffer {
		if now.Sub(event.created) > ws.options.EventBufferTTL {
			delete(ws.eventBuffer, sn)
			expired = true
			ws.client.logger.Warnf("丢弃过期缓冲事件SN: %d", sn)
		}
	}
	return expired
}

func (ws *WebSocketClient) dispatchOrderedEvent(msg *WebSocketMessage) error {
	event, err := ws.decodeEvent(msg)
	if err != nil {
		return err
	}

	if err := ws.enqueueDecodedEvent(event, true); err != nil {
		return err
	}
	ws.setReceivedSN(msg.SN)
	return nil
}

func (ws *WebSocketClient) decodeEvent(msg *WebSocketMessage) (*Event, error) {
	var event Event
	if err := json.Unmarshal(msg.D, &event); err != nil {
		return nil, fmt.Errorf("解析事件失败: %w", err)
	}
	event.S = msg.S
	event.SN = msg.SN
	return &event, nil
}

func (ws *WebSocketClient) dispatchDecodedEvent(event *Event) error {
	return ws.enqueueDecodedEvent(event, false)
}

func (ws *WebSocketClient) enqueueDecodedEvent(event *Event, ordered bool) error {
	ws.client.logger.Debugf("收到事件: 类型=%d sn=%d", event.Type, event.SN)
	select {
	case <-ws.ctx.Done():
		return ws.ctx.Err()
	default:
	}
	select {
	case ws.dispatchQueue <- queuedWebSocketEvent{event: event, ordered: ordered, generation: ws.getGeneration()}:
		return nil
	case <-ws.ctx.Done():
		return ws.ctx.Err()
	default:
		return ErrEventDispatchQueueFull
	}
}

func (ws *WebSocketClient) runEventDispatcher() {
	for {
		select {
		case <-ws.ctx.Done():
			return
		case queued := <-ws.dispatchQueue:
			if queued.ordered && queued.generation != ws.getGeneration() {
				continue
			}
			err := ws.dispatcher.dispatch(queued.event, func(recovered any) {
				ws.client.logger.Error("事件处理器发生panic")
			})
			if err != nil {
				ws.client.logger.Error("分发WebSocket事件失败")
				if queued.ordered {
					if ws.IsConnected() {
						ws.markReconnectPending()
					}
					ws.invalidateGeneration(queued.generation)
					_ = ws.closeActiveConnection()
				}
			}
			if err == nil && queued.ordered && queued.generation == ws.getGeneration() {
				ws.setSN(queued.event.SN)
			}
		}
	}
}

// handleHello 处理Hello消息
func (ws *WebSocketClient) handleHello(msg *WebSocketMessage) error {
	var hello HelloMessage
	if err := json.Unmarshal(msg.D, &hello); err != nil {
		return fmt.Errorf("解析Hello消息失败: %w", err)
	}
	if hello.Code != 0 {
		return fmt.Errorf("WebSocket Hello失败，code=%d", hello.Code)
	}
	if hello.SessionID == "" {
		return fmt.Errorf("WebSocket Hello缺少session_id")
	}

	ws.setSessionID(hello.SessionID)
	ws.client.logger.Info("WebSocket会话建立成功")
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
	if ws.IsConnected() {
		ws.markReconnectPending()
	}
	ws.stopHeartbeat()
	ws.stopGapTimer()
	ws.resetSessionStateForFreshReconnect()
	_ = ws.closeActiveConnection()

	return nil
}

// handleResumeAck 处理重连确认消息
func (ws *WebSocketClient) handleResumeAck(msg *WebSocketMessage) error {
	ws.client.logger.Info("重连成功")
	return nil
}

// startHeartbeat 启动心跳
func (ws *WebSocketClient) startHeartbeat() {
	ws.heartbeatMu.Lock()
	if ws.heartbeatCancel != nil {
		ws.heartbeatCancel()
	}
	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
	}
	heartbeatCtx, cancel := context.WithCancel(ws.ctx)
	ticker := time.NewTicker(30 * time.Second)
	ws.heartbeatCancel = cancel
	ws.heartbeatTicker = ticker
	ws.heartbeatMu.Unlock()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ws.client.logger.Error("心跳处理发生panic")
			}
		}()

		consecutiveFailures := 0
		const maxFailures = 3

		for {
			select {
			case <-heartbeatCtx.Done():
				return
			case <-ticker.C:
				ping := WebSocketMessage{
					S:  SignalPing,
					SN: ws.getSN(),
				}

				if err := ws.sendMessage(&ping); err != nil {
					consecutiveFailures++
					ws.client.logger.Errorf("发送心跳失败 (%d/%d)", consecutiveFailures, maxFailures)

					if consecutiveFailures >= maxFailures {
						ws.client.logger.Error("连续心跳失败，触发重连")
						ws.markReconnectPending()
						_ = ws.closeActiveConnection()
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

func (ws *WebSocketClient) stopHeartbeat() {
	ws.heartbeatMu.Lock()
	if ws.heartbeatCancel != nil {
		ws.heartbeatCancel()
		ws.heartbeatCancel = nil
	}
	if ws.heartbeatTicker != nil {
		ws.heartbeatTicker.Stop()
		ws.heartbeatTicker = nil
	}
	ws.heartbeatMu.Unlock()
	ws.clearPendingPing()
}

func (ws *WebSocketClient) trackPendingPing(sn int) {
	ws.pendingPingMu.Lock()
	ws.pendingPingSN = sn
	ws.pendingPing = true
	ws.pendingPingMu.Unlock()

	timeout := ws.options.PongTimeout
	if timeout <= 0 {
		return
	}

	time.AfterFunc(timeout, func() {
		ws.pendingPingMu.Lock()
		pending := ws.pendingPing && ws.pendingPingSN == sn
		ws.pendingPingMu.Unlock()
		if !pending || ws.ctx.Err() != nil {
			return
		}

		ws.client.logger.Errorf("WebSocket Pong超时，SN: %d", sn)
		ws.markReconnectPending()
		_ = ws.closeActiveConnection()
	})
}

func (ws *WebSocketClient) clearPendingPing() {
	ws.pendingPingMu.Lock()
	ws.pendingPingSN = 0
	ws.pendingPing = false
	ws.pendingPingMu.Unlock()
}

func (ws *WebSocketClient) ackPendingPing(sn int) {
	ws.pendingPingMu.Lock()
	if ws.pendingPing && (ws.pendingPingSN == sn || sn == 0) {
		ws.pendingPingSN = 0
		ws.pendingPing = false
	}
	ws.pendingPingMu.Unlock()
}

// sendMessage 发送WebSocket消息
func (ws *WebSocketClient) sendMessage(msg *WebSocketMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	ws.client.logger.Debugf("发送WebSocket消息: signal=%d sn=%d bytes=%d", msg.S, msg.SN, len(data))

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

func (ws *WebSocketClient) setReceivedSN(sn int) {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.receivedSN = sn
}

func (ws *WebSocketClient) getReceivedSN() int {
	ws.stateMu.RLock()
	defer ws.stateMu.RUnlock()
	return ws.receivedSN
}

func (ws *WebSocketClient) getGeneration() uint64 {
	ws.stateMu.RLock()
	defer ws.stateMu.RUnlock()
	return ws.generation
}

func (ws *WebSocketClient) activateConnectionGeneration() {
	ws.stopGapTimer()
	ws.stateMu.Lock()
	ws.generation++
	ws.receivedSN = ws.sn
	ws.eventBuffer = make(map[int]bufferedEvent)
	ws.stateMu.Unlock()
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

func (ws *WebSocketClient) setGatewayURL(gatewayURL string) {
	ws.stateMu.Lock()
	defer ws.stateMu.Unlock()
	ws.gatewayURL = gatewayURL
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
	ws.stopGapTimer()
	ws.stateMu.Lock()
	ws.generation++
	ws.sn = 0
	ws.receivedSN = 0
	ws.sessionID = ""
	ws.gatewayURL = ""
	ws.eventBuffer = make(map[int]bufferedEvent)
	ws.stateMu.Unlock()
	ws.clearPendingPing()
}

func (ws *WebSocketClient) invalidateGeneration(generation uint64) {
	ws.stopGapTimer()
	ws.stateMu.Lock()
	if ws.generation == generation {
		ws.generation++
		ws.receivedSN = ws.sn
		ws.eventBuffer = make(map[int]bufferedEvent)
	}
	ws.stateMu.Unlock()
}

func (ws *WebSocketClient) scheduleGapTimer() {
	ws.gapTimerMu.Lock()
	if ws.gapTimer != nil {
		ws.gapTimerMu.Unlock()
		return
	}
	generation := ws.getGeneration()
	ws.gapTimer = time.AfterFunc(ws.options.EventBufferTTL, func() {
		ws.gapTimerMu.Lock()
		ws.gapTimer = nil
		ws.gapTimerMu.Unlock()
		if ws.ctx.Err() != nil || generation != ws.getGeneration() {
			return
		}
		ws.stateMu.RLock()
		hasGap := len(ws.eventBuffer) > 0
		ws.stateMu.RUnlock()
		if hasGap {
			ws.client.logger.Warn("WebSocket事件序列缺口超时，关闭连接以重放")
			ws.markReconnectPending()
			_ = ws.closeActiveConnection()
		}
	})
	ws.gapTimerMu.Unlock()
}

func (ws *WebSocketClient) stopGapTimer() {
	ws.gapTimerMu.Lock()
	if ws.gapTimer != nil {
		ws.gapTimer.Stop()
		ws.gapTimer = nil
	}
	ws.gapTimerMu.Unlock()
}

func (ws *WebSocketClient) closeActiveConnection() error {
	ws.connMu.Lock()
	conn := ws.conn
	ws.conn = nil
	ws.isConnected = false
	ws.connMu.Unlock()
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (ws *WebSocketClient) markReconnectPending() {
	if ws.ctx.Err() != nil {
		return
	}
	ws.stateMu.Lock()
	ws.reconnecting = true
	ws.stateMu.Unlock()
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
	defer func() { _ = r.Close() }()

	decoded, err := io.ReadAll(io.LimitReader(r, ws.options.ReadLimit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(decoded)) > ws.options.ReadLimit {
		return nil, fmt.Errorf("解压消息超过读取限制")
	}
	return decoded, nil
}
