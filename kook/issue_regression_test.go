package kook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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
