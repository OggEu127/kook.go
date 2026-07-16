package kook

import (
	"bytes"
	"compress/zlib"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func websocketEventMessage(t *testing.T, sn int, eventType MessageType, content any, extra any) *WebSocketMessage {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"channel_type": "GROUP",
		"type":         eventType,
		"target_id":    "channel",
		"author_id":    "user",
		"content":      content,
		"extra":        extra,
		"msg_id":       "message",
	})
	require.NoError(t, err)
	return &WebSocketMessage{S: SignalEvent, SN: sn, D: payload}
}

func TestWebSocketOrdersAndDeduplicatesSN(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false)
	defer ws.Close()
	received := make(chan int, 3)
	ws.OnMessage(MessageTypeText, func(event *MessageEvent) { received <- event.SN })

	require.NoError(t, ws.handleMessage(websocketEventMessage(t, 2, MessageTypeText, "second", map[string]any{})))
	require.NoError(t, ws.handleMessage(websocketEventMessage(t, 1, MessageTypeText, "first", map[string]any{})))
	require.Equal(t, 1, receiveInt(t, received))
	require.Equal(t, 2, receiveInt(t, received))
	require.Equal(t, 2, ws.getSN())

	require.NoError(t, ws.handleMessage(websocketEventMessage(t, 2, MessageTypeText, "duplicate", map[string]any{})))
	select {
	case sn := <-received:
		t.Fatalf("duplicate SN was dispatched: %d", sn)
	case <-time.After(100 * time.Millisecond):
	}
}

func receiveInt(t *testing.T, values <-chan int) int {
	t.Helper()
	select {
	case value := <-values:
		return value
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for WebSocket event")
		return 0
	}
}

func TestWebSocketSystemEventDispatch(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false)
	defer ws.Close()
	received := make(chan string, 1)
	ws.OnSystemEvent(SystemEventUpdatedGuild, func(event *SystemEvent) {
		var body struct {
			Name string `json:"name"`
		}
		require.NoError(t, event.DecodeBody(&body))
		received <- body.Name
	})
	extra := map[string]any{"type": SystemEventUpdatedGuild, "body": map[string]any{"name": "updated"}}
	require.NoError(t, ws.handleMessage(websocketEventMessage(t, 1, MessageTypeSystem, "[系统消息]", extra)))
	select {
	case name := <-received:
		require.Equal(t, "updated", name)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for system event")
	}
}

func TestWebSocketEventBufferLimitAndTTL(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false, WebSocketOptions{MaxEventBuffer: 1, EventBufferTTL: 10 * time.Millisecond})
	defer ws.Close()

	require.NoError(t, ws.handleOrderedEvent(websocketEventMessage(t, 2, MessageTypeText, "buffered", map[string]any{})))
	err := ws.handleOrderedEvent(websocketEventMessage(t, 3, MessageTypeText, "overflow", map[string]any{}))
	require.ErrorIs(t, err, ErrEventBufferFull)

	time.Sleep(20 * time.Millisecond)
	require.NoError(t, ws.handleOrderedEvent(websocketEventMessage(t, 3, MessageTypeText, "after-expiry", map[string]any{})))
	require.Equal(t, 3, ws.getSN())
}

func TestWebSocketBoundedDispatchQueue(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false, WebSocketOptions{DispatchQueueSize: 1})
	defer ws.Close()
	started := make(chan struct{})
	release := make(chan struct{})
	ws.OnMessage(MessageTypeText, func(*MessageEvent) {
		select {
		case <-started:
		default:
			close(started)
		}
		<-release
	})

	require.NoError(t, ws.dispatchDecodedEvent(&Event{Type: MessageTypeText, Content: json.RawMessage(`"one"`)}))
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("slow handler did not start")
	}
	require.NoError(t, ws.dispatchDecodedEvent(&Event{Type: MessageTypeText, Content: json.RawMessage(`"two"`)}))
	require.ErrorIs(t, ws.dispatchDecodedEvent(&Event{Type: MessageTypeText, Content: json.RawMessage(`"three"`)}), ErrEventDispatchQueueFull)
	close(release)
}

func TestWebSocketHelloResumeReconnectAndPong(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false, WebSocketOptions{PongTimeout: time.Second})
	defer ws.Close()

	helloData, err := json.Marshal(HelloMessage{Code: 0, SessionID: "session"})
	require.NoError(t, err)
	require.NoError(t, ws.handleMessage(&WebSocketMessage{S: SignalHello, D: helloData}))
	require.Equal(t, "session", ws.getSessionID())

	ws.stateMu.Lock()
	ws.gatewayURL = "wss://example.test/gateway?compress=1"
	ws.sn = 9
	ws.stateMu.Unlock()
	resumeURL, err := url.Parse(ws.resumeGatewayURL())
	require.NoError(t, err)
	require.Equal(t, "1", resumeURL.Query().Get("resume"))
	require.Equal(t, "9", resumeURL.Query().Get("sn"))
	require.Equal(t, "session", resumeURL.Query().Get("session_id"))

	ws.trackPendingPing(9)
	require.NoError(t, ws.handleMessage(&WebSocketMessage{S: SignalPong, SN: 9}))
	ws.pendingPingMu.Lock()
	require.Zero(t, ws.pendingPingSN)
	ws.pendingPingMu.Unlock()

	require.NoError(t, ws.handleMessage(&WebSocketMessage{S: SignalReconnect}))
	require.Empty(t, ws.getSessionID())
	require.Zero(t, ws.getSN())
	require.Empty(t, ws.resumeGatewayURL())
}

func TestWebSocketRespondsToServerPing(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	serverConnection := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			serverConnection <- connection
		}
	}))
	defer server.Close()

	websocketURL := "ws" + strings.TrimPrefix(server.URL, "http")
	clientConnection, _, err := websocket.DefaultDialer.Dial(websocketURL, nil)
	require.NoError(t, err)
	defer clientConnection.Close()
	serverConn := <-serverConnection
	defer serverConn.Close()

	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false)
	defer ws.Close()
	ws.connMu.Lock()
	ws.conn = clientConnection
	ws.isConnected = true
	ws.connMu.Unlock()

	require.NoError(t, ws.handleMessage(&WebSocketMessage{S: SignalPing, SN: 12}))
	_ = serverConn.SetReadDeadline(time.Now().Add(time.Second))
	var pong WebSocketMessage
	require.NoError(t, serverConn.ReadJSON(&pong))
	require.Equal(t, SignalPong, pong.S)
	require.Equal(t, 12, pong.SN)
}

func TestWebSocketZlibDecompression(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, true)
	defer ws.Close()

	plain := []byte(`{"s":1,"d":{"code":0,"session_id":"session"}}`)
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	_, err := writer.Write(plain)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	decoded, err := ws.decompress(buffer.Bytes())
	require.NoError(t, err)
	require.Equal(t, plain, decoded)
}

func TestWebSocketDispatchReturnsContextErrorAfterClose(t *testing.T) {
	client := newWebhookTestClient(t)
	defer client.Close()
	ws := NewWebSocketClient(client, false)
	require.NoError(t, ws.Close())
	err := ws.dispatchDecodedEvent(&Event{Type: MessageTypeText, Content: json.RawMessage(`"closed"`)})
	require.True(t, errors.Is(err, ws.ctx.Err()))
}
