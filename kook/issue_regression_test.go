package kook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func TestJSONContentTypeIsSetForPutAndDeleteWithBody(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != method {
					t.Fatalf("method = %s, want %s", r.Method, method)
				}
				if got := r.Header.Get("Content-Type"); got != "application/json" {
					t.Fatalf("Content-Type = %q, want application/json", got)
				}
				writeKOOKResponse(t, w, map[string]interface{}{})
			})
			defer closeServer()

			params := map[string]interface{}{"id": "1"}
			var err error
			if method == http.MethodPut {
				_, err = client.Put(context.Background(), "test/content-type", params)
			} else {
				_, err = client.Delete(context.Background(), "test/content-type", params)
			}
			if err != nil {
				t.Fatalf("%s returned error: %v", method, err)
			}
		})
	}
}

func TestWebSocketSendMessageSerializesConcurrentWrites(t *testing.T) {
	const messageCount = 64

	received := make(chan struct{}, messageCount)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var msg WebSocketMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				t.Errorf("unmarshal websocket message: %v", err)
				return
			}
			received <- struct{}{}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	client := NewClient("test-token", WithoutRateLimit(), WithoutRetry())
	ws := NewWebSocketClient(client, false)
	ws.conn = conn
	ws.isConnected = true

	var wg sync.WaitGroup
	errs := make(chan error, messageCount)
	for i := 0; i < messageCount; i++ {
		wg.Add(1)
		go func(sn int) {
			defer wg.Done()
			payload, _ := json.Marshal(PingMessage{SN: sn})
			errs <- ws.sendMessage(&WebSocketMessage{
				S:  SignalPing,
				D:  payload,
				SN: sn,
			})
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("sendMessage returned error: %v", err)
		}
	}

	deadline := time.After(3 * time.Second)
	for i := 0; i < messageCount; i++ {
		select {
		case <-received:
		case <-deadline:
			t.Fatalf("received %d/%d websocket messages", i, messageCount)
		}
	}
}

func TestWebSocketSessionStateAccessIsConcurrentSafe(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(value int) {
			defer wg.Done()
			ws.setSN(value)
			_ = ws.getSN()
			ws.setSessionID("session")
			_ = ws.getSessionID()
			ws.setReconnectCount(value)
			_ = ws.getReconnectCount()
			_ = ws.incrementReconnectCount()
		}(i)
	}
	wg.Wait()
}

func TestWebSocketResumeGatewayURLUsesSessionAndSN(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)
	ws.gatewayURL = "wss://gateway.example.test/socket?compress=0"
	ws.setSessionID("session-1")
	ws.setSN(42)

	got := ws.resumeGatewayURL()
	if got == "" {
		t.Fatal("resumeGatewayURL returned empty URL")
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse resume URL: %v", err)
	}
	if u.Query().Get("resume") != "1" || u.Query().Get("sn") != "42" || u.Query().Get("session_id") != "session-1" {
		t.Fatalf("resume query = %s", u.RawQuery)
	}
	if u.Query().Get("compress") != "0" {
		t.Fatalf("existing query was not preserved: %s", u.RawQuery)
	}
}

func TestWebSocketReconnectSignalResetsSessionAndClosesConnection(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)
	clientConn, serverConn := netPipeWebSocket(t)
	defer clientConn.Close()
	defer serverConn.Close()

	ws.conn = clientConn
	ws.isConnected = true
	ws.gatewayURL = "wss://gateway.example.test/socket"
	ws.setSessionID("session-1")
	ws.setSN(7)
	ws.eventBuffer[8] = bufferedEvent{msg: &WebSocketMessage{S: SignalEvent, SN: 8}, created: time.Now()}

	if err := ws.handleReconnect(&WebSocketMessage{S: SignalReconnect}); err != nil {
		t.Fatalf("handleReconnect: %v", err)
	}
	if ws.getSessionID() != "" || ws.getSN() != 0 || ws.gatewayURL != "" || len(ws.eventBuffer) != 0 {
		t.Fatalf("session state was not reset: session=%q sn=%d gateway=%q buffer=%d", ws.getSessionID(), ws.getSN(), ws.gatewayURL, len(ws.eventBuffer))
	}
	if ws.IsConnected() {
		t.Fatal("websocket should be marked disconnected")
	}
}

func TestWebSocketEventsAreDispatchedInSNOrder(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)
	received := make(chan string, 2)
	ws.OnEvent(EventTypeTextMessage, func(event *Event) {
		received <- event.Content
	})

	second := mustRawMessage(t, Event{Type: EventTypeTextMessage, Content: "second"})
	first := mustRawMessage(t, Event{Type: EventTypeTextMessage, Content: "first"})
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 2, D: second}); err != nil {
		t.Fatalf("handle second: %v", err)
	}
	select {
	case got := <-received:
		t.Fatalf("received out-of-order event before SN 1: %s", got)
	default:
	}

	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 1, D: first}); err != nil {
		t.Fatalf("handle first: %v", err)
	}
	if got := readEventContent(t, received); got != "first" {
		t.Fatalf("first event = %q", got)
	}
	if got := readEventContent(t, received); got != "second" {
		t.Fatalf("second event = %q", got)
	}
	if ws.getSN() != 2 {
		t.Fatalf("SN = %d, want 2", ws.getSN())
	}
}

func TestWebSocketEventBufferRejectsOverflowAndDuplicate(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false, WebSocketOptions{
		MaxEventBuffer: 1,
	})
	event := mustRawMessage(t, Event{Type: EventTypeTextMessage, Content: "buffered"})

	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 3, D: event}); err != nil {
		t.Fatalf("first buffered event returned error: %v", err)
	}
	if len(ws.eventBuffer) != 1 {
		t.Fatalf("buffer len = %d, want 1", len(ws.eventBuffer))
	}
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 3, D: event}); err != nil {
		t.Fatalf("duplicate buffered event returned error: %v", err)
	}
	if len(ws.eventBuffer) != 1 {
		t.Fatalf("buffer len after duplicate = %d, want 1", len(ws.eventBuffer))
	}
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 4, D: event}); !errors.Is(err, ErrEventBufferFull) {
		t.Fatalf("overflow error = %v, want ErrEventBufferFull", err)
	}
}

func TestWebSocketEventBufferTTLExpiresStaleGap(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false, WebSocketOptions{
		EventBufferTTL: time.Millisecond,
	})
	received := make(chan string, 1)
	ws.OnEvent(EventTypeTextMessage, func(event *Event) {
		received <- event.Content
	})
	event := mustRawMessage(t, Event{Type: EventTypeTextMessage, Content: "fresh"})

	ws.eventBuffer[2] = bufferedEvent{
		msg:     &WebSocketMessage{S: SignalEvent, SN: 2, D: event},
		created: time.Now().Add(-time.Second),
	}
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 3, D: event}); err != nil {
		t.Fatalf("handle event after stale gap: %v", err)
	}
	if got := readEventContent(t, received); got != "fresh" {
		t.Fatalf("event content = %q, want fresh", got)
	}
	if ws.getSN() != 3 {
		t.Fatalf("SN = %d, want 3", ws.getSN())
	}
}

func TestWebSocketDispatchCopiesHandlerSlice(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)
	calls := 0
	ws.OnEvent(EventTypeTextMessage, func(event *Event) {
		calls++
		ws.OnEvent(EventTypeTextMessage, func(event *Event) {
			calls++
		})
	})

	event := mustRawMessage(t, Event{Type: EventTypeTextMessage, Content: "hello"})
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 1, D: event}); err != nil {
		t.Fatalf("handle first event: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls after first event = %d, want 1", calls)
	}
	if err := ws.handleEvent(&WebSocketMessage{S: SignalEvent, SN: 2, D: event}); err != nil {
		t.Fatalf("handle second event: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls after second event = %d, want 3", calls)
	}
}

func TestCreateCategoryChannelOnlySendsOfficialFields(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		body := readBodyMap(t, r)
		want := map[string]interface{}{
			"guild_id":    "guild-1",
			"name":        "category",
			"is_category": float64(1),
		}
		if len(body) != len(want) {
			t.Fatalf("body = %#v, want only %#v", body, want)
		}
		for key, value := range want {
			if body[key] != value {
				t.Fatalf("%s = %#v, want %#v; body=%#v", key, body[key], value, body)
			}
		}
		writeKOOKResponse(t, w, map[string]interface{}{})
	})
	defer closeServer()

	_, err := client.Channel.CreateChannel(context.Background(), "guild-1", CreateChannelParams{
		Name:         "category",
		Type:         2,
		ParentID:     "parent-1",
		LimitAmount:  10,
		VoiceQuality: "3",
		IsCategory:   true,
	})
	if err != nil {
		t.Fatalf("CreateChannel returned error: %v", err)
	}
}

func TestVoiceQualityIsSentAsString(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		body := readBodyMap(t, r)
		if got := body["voice_quality"]; got != "3" {
			t.Fatalf("voice_quality = %#v (%T), want string 3", got, got)
		}
		writeKOOKResponse(t, w, map[string]interface{}{})
	})
	defer closeServer()

	_, err := client.Channel.UpdateChannel(context.Background(), "channel-1", UpdateChannelParams{VoiceQuality: "3"})
	if err != nil {
		t.Fatalf("UpdateChannel returned error: %v", err)
	}
}

func TestMessageAttachmentsAcceptOfficialShapesAndAuthorID(t *testing.T) {
	for name, payload := range map[string]string{
		"object": `{"id":"msg-1","attachments":{"type":"image","url":"https://example.test/a.png"},"author_id":"user-1"}`,
		"null":   `{"id":"msg-1","attachments":null,"author_id":"user-1"}`,
		"false":  `{"id":"msg-1","attachments":false,"author_id":"user-1"}`,
	} {
		t.Run(name, func(t *testing.T) {
			var msg Message
			if err := json.Unmarshal([]byte(payload), &msg); err != nil {
				t.Fatalf("unmarshal Message: %v", err)
			}
			if msg.AuthorID != "user-1" {
				t.Fatalf("AuthorID = %q, want user-1", msg.AuthorID)
			}
		})
	}
}

func TestNon2xxHTTPStatusReturnsKOOKError(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>bad gateway</html>"))
	})
	defer closeServer()

	_, err := client.Get(context.Background(), "test/non-json", nil)
	if err == nil {
		t.Fatal("Get returned nil error")
	}
	kookErr, ok := IsKOOKError(err)
	if !ok {
		t.Fatalf("error type = %T, want KOOKError: %v", err, err)
	}
	if kookErr.HTTPStatus != http.StatusBadGateway || kookErr.Code != http.StatusBadGateway {
		t.Fatalf("unexpected KOOKError: %#v", kookErr)
	}
}

func TestKOOKErrorHelpersRecognizeWrappedErrors(t *testing.T) {
	base := NewKOOKError(42900, "rate limited")
	wrapped := errors.Join(errors.New("outer"), base)

	if got, ok := IsKOOKError(wrapped); !ok || got != base {
		t.Fatalf("IsKOOKError = %#v, %v; want base,true", got, ok)
	}
	if !IsRetryableError(wrapped) || !IsRateLimitError(wrapped) {
		t.Fatal("wrapped KOOKError should be retryable and rate limited")
	}
}

func TestNewClientWithErrorReturnsErrorForEmptyToken(t *testing.T) {
	client, err := NewClientWithError("")
	if err == nil {
		t.Fatal("NewClientWithError returned nil error")
	}
	if client != nil {
		t.Fatalf("client = %#v, want nil", client)
	}
}

func TestDoWithRetryReturnsNonRetryableErrorUnwrapped(t *testing.T) {
	want := errors.New("validation failed")
	_, err := DoWithRetry(context.Background(), func(context.Context) (*Response, error) {
		return nil, want
	}, withoutRetryConfig(), logrus.New())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want original %v", err, want)
	}
	if err.Error() != want.Error() {
		t.Fatalf("err = %q, want %q", err.Error(), want.Error())
	}
}

func TestDebugRequestLogRedactsAuthorization(t *testing.T) {
	var logs bytes.Buffer
	logger := logrus.New()
	logger.SetOutput(&logs)
	logger.SetLevel(logrus.DebugLevel)

	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		writeKOOKResponse(t, w, map[string]interface{}{})
	})
	defer closeServer()
	client.logger = logger

	if _, err := client.Get(context.Background(), "test/logs", nil); err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if bytes.Contains(logs.Bytes(), []byte("test-token")) {
		t.Fatalf("debug logs leaked token: %s", logs.String())
	}
	if !bytes.Contains(logs.Bytes(), []byte("[REDACTED]")) {
		t.Fatalf("debug logs did not include redacted marker: %s", logs.String())
	}
}

func TestRateLimiterWaitHonorsContextCancellation(t *testing.T) {
	limiter := NewRateLimiter(time.Hour, 1)
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatalf("initial Wait returned error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Wait error = %v, want context.Canceled", err)
	}
}

func TestRateLimiterCloseStopsRefillLoop(t *testing.T) {
	limiter := NewRateLimiter(time.Millisecond, 1)
	limiter.Close()
	limiter.Close()
}

func TestWebSocketPingAndPongUseTopLevelSN(t *testing.T) {
	client := NewClient("test-token", WithoutRateLimit(), WithoutRetry())
	ws := NewWebSocketClient(client, false)

	clientConn, serverConn := netPipeWebSocket(t)
	defer clientConn.Close()
	defer serverConn.Close()

	ws.conn = clientConn
	ws.isConnected = true
	ws.setSN(7)

	if err := ws.sendMessage(&WebSocketMessage{S: SignalPing, SN: ws.getSN()}); err != nil {
		t.Fatalf("send ping: %v", err)
	}
	_, data, err := serverConn.ReadMessage()
	if err != nil {
		t.Fatalf("read ping: %v", err)
	}
	var ping map[string]interface{}
	if err := json.Unmarshal(data, &ping); err != nil {
		t.Fatalf("unmarshal ping: %v", err)
	}
	if ping["sn"] != float64(7) {
		t.Fatalf("top-level sn = %#v, want 7; raw=%s", ping["sn"], data)
	}
	if _, exists := ping["d"]; exists {
		t.Fatalf("unexpected d field in ping: %s", data)
	}

	if err := ws.handlePing(&WebSocketMessage{S: SignalPing, SN: 8}); err != nil {
		t.Fatalf("handlePing: %v", err)
	}
	_, data, err = serverConn.ReadMessage()
	if err != nil {
		t.Fatalf("read pong: %v", err)
	}
	var pong map[string]interface{}
	if err := json.Unmarshal(data, &pong); err != nil {
		t.Fatalf("unmarshal pong: %v", err)
	}
	if pong["sn"] != float64(8) {
		t.Fatalf("pong top-level sn = %#v, want 8; raw=%s", pong["sn"], data)
	}
	if _, exists := pong["d"]; exists {
		t.Fatalf("unexpected d field in pong: %s", data)
	}
}

func TestWebSocketOptionsDefaultsAndOverrides(t *testing.T) {
	defaults := DefaultWebSocketOptions()
	if defaults.ReadLimit != 8<<20 || defaults.ReadTimeout != 90*time.Second || defaults.HelloTimeout != 6*time.Second ||
		defaults.PongTimeout != 6*time.Second || defaults.MaxEventBuffer != 1024 || defaults.EventBufferTTL != 30*time.Second {
		t.Fatalf("unexpected defaults: %#v", defaults)
	}

	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false, WebSocketOptions{
		ReadLimit:      64,
		ReadTimeout:    time.Second,
		HelloTimeout:   2 * time.Second,
		PongTimeout:    3 * time.Second,
		MaxEventBuffer: 4,
		EventBufferTTL: 5 * time.Second,
	})
	if ws.options.ReadLimit != 64 || ws.options.ReadTimeout != time.Second || ws.options.HelloTimeout != 2*time.Second ||
		ws.options.PongTimeout != 3*time.Second || ws.options.MaxEventBuffer != 4 || ws.options.EventBufferTTL != 5*time.Second {
		t.Fatalf("unexpected options: %#v", ws.options)
	}
}

func TestWebSocketReadLimitClosesOversizedMessages(t *testing.T) {
	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverConnCh <- conn
		<-r.Context().Done()
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[len("http"):]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer clientConn.Close()
	serverConn := <-serverConnCh
	defer serverConn.Close()

	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false, WebSocketOptions{
		ReadLimit:   16,
		ReadTimeout: time.Second,
	})
	ws.configureRead(clientConn)

	if err := serverConn.WriteMessage(websocket.TextMessage, []byte(`{"s":0,"sn":1,"d":{"type":1,"content":"this message exceeds the read limit"}}`)); err != nil {
		t.Fatalf("write oversized message: %v", err)
	}
	if _, _, err := clientConn.ReadMessage(); err == nil {
		t.Fatal("ReadMessage returned nil error for oversized message")
	}
}

func TestWebSocketPongAcknowledgesPendingPing(t *testing.T) {
	ws := NewWebSocketClient(NewClient("test-token", WithoutRateLimit(), WithoutRetry()), false)
	ws.trackPendingPing(9)
	if err := ws.handleMessage(&WebSocketMessage{S: SignalPong, SN: 9}); err != nil {
		t.Fatalf("handle pong: %v", err)
	}
	if ws.pendingPingSN != 0 {
		t.Fatalf("pendingPingSN = %d, want 0", ws.pendingPingSN)
	}
}

func TestDeprecatedUserMethodsDoNotCallHTTP(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("deprecated method made HTTP request to %s %s", r.Method, r.URL.Path)
	})
	defer closeServer()

	ctx := context.Background()
	if _, err := client.User.GetUserOnlineStatus(ctx, "user-1"); err == nil {
		t.Fatal("GetUserOnlineStatus returned nil error")
	}
	if _, err := client.User.UpdateUserInfo(ctx, UpdateUserParams{Username: "name"}); err == nil {
		t.Fatal("UpdateUserInfo returned nil error")
	}
	if err := client.User.BlockUser(ctx, "user-1"); err == nil {
		t.Fatal("BlockUser returned nil error")
	}
	if err := client.User.UnblockUser(ctx, "user-1"); err == nil {
		t.Fatal("UnblockUser returned nil error")
	}
	if _, err := client.User.GetBlockedUsers(ctx); err == nil {
		t.Fatal("GetBlockedUsers returned nil error")
	}
}

func TestGetGuildMemberUsesUserView(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v3/user/view" {
			t.Fatalf("path = %s, want /v3/user/view", r.URL.Path)
		}
		if r.URL.Query().Get("guild_id") != "guild-1" || r.URL.Query().Get("user_id") != "user-1" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		writeKOOKResponse(t, w, map[string]interface{}{"id": "user-1", "username": "tester"})
	})
	defer closeServer()

	member, err := client.Guild.GetGuildMember(context.Background(), "guild-1", "user-1")
	if err != nil {
		t.Fatalf("GetGuildMember returned error: %v", err)
	}
	if member.ID != "user-1" {
		t.Fatalf("member.ID = %q, want user-1", member.ID)
	}
}

func TestSendPipeMessageUsesOfficialQueryAndBody(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/message/send-pipemsg" {
			t.Fatalf("path = %s, want /v3/message/send-pipemsg", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("access_token") != "pipe-token" || query.Get("target_id") != "channel-1" || query.Get("type") != "9" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		body := readBodyMap(t, r)
		if body["content"] != "hello" {
			t.Fatalf("content = %#v, want hello", body["content"])
		}
		if _, exists := body["target_id"]; exists {
			t.Fatalf("target_id should be query-only: %#v", body)
		}
		if _, exists := body["type"]; exists {
			t.Fatalf("type should be query-only: %#v", body)
		}
		writeKOOKResponse(t, w, map[string]interface{}{"msg_id": "msg-1"})
	})
	defer closeServer()

	msg, err := client.Message.SendPipeMessage(context.Background(), SendMessageParams{
		TargetID:    "channel-1",
		Content:     "hello",
		AccessToken: "pipe-token",
	})
	if err != nil {
		t.Fatalf("SendPipeMessage returned error: %v", err)
	}
	if msg.ID != "msg-1" {
		t.Fatalf("msg.ID = %q, want msg-1", msg.ID)
	}
}

func TestSendPipeTemplateInputUsesRawBody(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/message/send-pipemsg" {
			t.Fatalf("path = %s, want /v3/message/send-pipemsg", r.URL.Path)
		}
		query := r.URL.Query()
		if query.Get("access_token") != "pipe-token" || query.Get("target_id") != "channel-1" || query.Get("type") != "9" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		body := readBodyMap(t, r)
		want := map[string]interface{}{"name": "tester", "count": float64(2)}
		if !reflect.DeepEqual(body, want) {
			t.Fatalf("body = %#v, want %#v", body, want)
		}
		if _, exists := body["content"]; exists {
			t.Fatalf("template body should not contain content wrapper: %#v", body)
		}
		writeKOOKResponse(t, w, map[string]interface{}{"msg_id": "msg-1"})
	})
	defer closeServer()

	msg, err := client.Message.SendPipe(context.Background(), SendPipeMessageParams{
		TargetID:    "channel-1",
		AccessToken: "pipe-token",
		TemplateInput: map[string]interface{}{
			"name":  "tester",
			"count": 2,
		},
	})
	if err != nil {
		t.Fatalf("SendPipe returned error: %v", err)
	}
	if msg.ID != "msg-1" {
		t.Fatalf("msg.ID = %q, want msg-1", msg.ID)
	}
}

func TestUploadFileContentUsesUnifiedHTTPError(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/asset/create" {
			t.Fatalf("path = %s, want /v3/asset/create", r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "multipart/form-data;") {
			t.Fatalf("Content-Type = %q, want multipart/form-data", got)
		}
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	})
	defer closeServer()

	_, err := client.Asset.UploadFileContent(context.Background(), "test.txt", []byte("ok"))
	if err == nil {
		t.Fatal("UploadFileContent returned nil error")
	}
	kookErr, ok := IsKOOKError(err)
	if !ok || kookErr.HTTPStatus != http.StatusBadGateway {
		t.Fatalf("err = %#v, want KOOKError with 502", err)
	}
}

func withoutRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:     0,
		InitialDelay:   time.Millisecond,
		MaxDelay:       time.Millisecond,
		BackoffFactor:  1,
		RetryableError: IsRetryableError,
	}
}

func netPipeWebSocket(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()

	serverConnCh := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverConnCh <- conn
		<-r.Context().Done()
		_ = conn.Close()
	}))
	t.Cleanup(server.Close)

	wsURL := "ws" + server.URL[len("http"):]
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial client websocket: %v", err)
	}

	select {
	case serverConn := <-serverConnCh:
		return clientConn, serverConn
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for server websocket")
	}
	return nil, nil
}

func mustRawMessage(t *testing.T, value interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal raw message: %v", err)
	}
	return data
}

func readEventContent(t *testing.T, ch <-chan string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
	return ""
}
