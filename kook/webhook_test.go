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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newWebhookTestClient(t *testing.T) *Client {
	t.Helper()
	return NewClient("test-token", WithoutRateLimit(), WithoutRetry())
}

func webhookPayload(t *testing.T, sn int, event map[string]any) []byte {
	t.Helper()
	payload, err := json.Marshal(map[string]any{"s": SignalEvent, "sn": sn, "d": event})
	require.NoError(t, err)
	return payload
}

func textWebhookEvent(verifyToken, content string) map[string]any {
	return map[string]any{
		"channel_type": "GROUP",
		"type":         MessageTypeText,
		"target_id":    "channel",
		"author_id":    "user",
		"content":      content,
		"extra":        map[string]any{},
		"msg_id":       "message",
		"verify_token": verifyToken,
	}
}

func performWebhookRequest(handler *WebhookHandler, body []byte, contentEncoding string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	if contentEncoding != "" {
		request.Header.Set("Content-Encoding", contentEncoding)
	}
	recorder := httptest.NewRecorder()
	handler.HandleRequest(recorder, request)
	return recorder
}

func TestWebhookChallengeAndVerifyToken(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler := NewWebhookHandler(client, "", "verify")

	challenge := webhookPayload(t, 0, map[string]any{
		"channel_type": "WEBHOOK_CHALLENGE",
		"verify_token": "verify",
		"challenge":    "challenge-value",
	})
	recorder := performWebhookRequest(handler, challenge, "")
	require.Equal(t, http.StatusOK, recorder.Code)
	require.JSONEq(t, `{"challenge":"challenge-value"}`, recorder.Body.String())

	invalid := webhookPayload(t, 1, textWebhookEvent("wrong", "hello"))
	recorder = performWebhookRequest(handler, invalid, "")
	require.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestWebhookCompressionAndHeaderlessZlib(t *testing.T) {
	tests := []struct {
		name     string
		encoding string
		compress func(*testing.T, []byte) []byte
	}{
		{"gzip", "gzip", gzipWebhookPayload},
		{"deflate", "deflate", zlibWebhookPayload},
		{"headerless-zlib", "", zlibWebhookPayload},
	}

	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := newWebhookTestClient(t)
			defer func() { _ = client.Close() }()
			handler := NewWebhookHandler(client, "", "verify")
			received := make(chan string, 1)
			handler.OnMessage(MessageTypeText, func(event *MessageEvent) {
				content, err := event.TextContent()
				require.NoError(t, err)
				received <- content
			})

			plain := webhookPayload(t, index+1, textWebhookEvent("verify", test.name))
			recorder := performWebhookRequest(handler, test.compress(t, plain), test.encoding)
			require.Equal(t, http.StatusOK, recorder.Code)
			select {
			case content := <-received:
				require.Equal(t, test.name, content)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for compressed webhook")
			}
		})
	}
}

func gzipWebhookPayload(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func zlibWebhookPayload(t *testing.T, payload []byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zlib.NewWriter(&buffer)
	_, err := writer.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func TestWebhookAES256CBC(t *testing.T) {
	for _, key := range []string{"short-key", "0123456789abcdef0123456789abcdef"} {
		key := key
		t.Run(fmt.Sprintf("key-length-%d", len(key)), func(t *testing.T) {
			client := newWebhookTestClient(t)
			defer func() { _ = client.Close() }()
			handler := NewWebhookHandler(client, key, "verify")
			defer func() { _ = handler.Shutdown(context.Background()) }()
			received := make(chan string, 1)
			handler.OnMessage(MessageTypeText, func(event *MessageEvent) {
				content, err := event.TextContent()
				require.NoError(t, err)
				received <- content
			})

			plain := webhookPayload(t, 11, textWebhookEvent("verify", "encrypted"))
			encrypted := encryptWebhookPayloadForTest(t, plain, key)
			recorder := performWebhookRequest(handler, encrypted, "")
			require.Equal(t, http.StatusOK, recorder.Code)
			select {
			case content := <-received:
				require.Equal(t, "encrypted", content)
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for encrypted webhook")
			}
		})
	}
}

func encryptWebhookPayloadForTest(t *testing.T, plain []byte, key string) []byte {
	t.Helper()
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		padded := make([]byte, 32)
		copy(padded, keyBytes)
		keyBytes = padded
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}
	block, err := aes.NewCipher(keyBytes)
	require.NoError(t, err)
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	padded := append(append([]byte(nil), plain...), bytes.Repeat([]byte{byte(padding)}, padding)...)
	iv := []byte("0123456789abcdef")
	cipherText := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(cipherText, padded)
	inner := base64.StdEncoding.EncodeToString(cipherText)
	payload := append(append([]byte(nil), iv...), []byte(inner)...)
	outer := base64.StdEncoding.EncodeToString(payload)
	encoded, err := json.Marshal(map[string]string{"encrypt": outer})
	require.NoError(t, err)
	return encoded
}

func TestWebhookSNDeduplication(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler := NewWebhookHandler(client, "", "verify", WithWebhookDedupTTL(time.Minute))
	defer func() { _ = handler.Shutdown(context.Background()) }()
	received := make(chan struct{}, 2)
	handler.OnMessage(MessageTypeText, func(*MessageEvent) { received <- struct{}{} })
	payload := webhookPayload(t, 42, textWebhookEvent("verify", "duplicate"))

	require.Equal(t, http.StatusOK, performWebhookRequest(handler, payload, "").Code)
	require.Equal(t, http.StatusOK, performWebhookRequest(handler, payload, "").Code)
	select {
	case <-received:
	case <-time.After(time.Second):
		t.Fatal("first webhook was not dispatched")
	}
	select {
	case <-received:
		t.Fatal("duplicate webhook was dispatched")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWebhookDeduplicationAllowsSNReuseForDifferentPayload(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler := NewWebhookHandler(client, "", "verify", WithWebhookDedupTTL(time.Minute))
	defer func() { _ = handler.Shutdown(context.Background()) }()
	received := make(chan string, 2)
	handler.OnMessage(MessageTypeText, func(event *MessageEvent) {
		content, err := event.TextContent()
		require.NoError(t, err)
		received <- content
	})

	first := webhookPayload(t, 1, textWebhookEvent("verify", "before-wrap"))
	second := webhookPayload(t, 1, textWebhookEvent("verify", "after-wrap"))
	require.Equal(t, http.StatusOK, performWebhookRequest(handler, first, "").Code)
	require.Equal(t, http.StatusOK, performWebhookRequest(handler, second, "").Code)
	require.Equal(t, "before-wrap", <-received)
	require.Equal(t, "after-wrap", <-received)
}

func TestMemoryWebhookDeduplicatorReservationOwnership(t *testing.T) {
	deduplicator := NewMemoryWebhookDeduplicator(2)
	now := time.Unix(100, 0)
	deduplicator.now = func() time.Time { return now }

	state, firstReservation, err := deduplicator.Reserve(context.Background(), "event", time.Second)
	require.NoError(t, err)
	require.Equal(t, WebhookDedupAcquired, state)
	now = now.Add(2 * time.Second)
	state, secondReservation, err := deduplicator.Reserve(context.Background(), "event", time.Second)
	require.NoError(t, err)
	require.Equal(t, WebhookDedupAcquired, state)
	require.NotEqual(t, firstReservation, secondReservation)

	require.ErrorIs(t, deduplicator.Complete(context.Background(), "event", firstReservation, time.Minute), ErrWebhookReservationLost)
	require.ErrorIs(t, deduplicator.Release(context.Background(), "event", firstReservation), ErrWebhookReservationLost)
	require.NoError(t, deduplicator.Complete(context.Background(), "event", secondReservation, time.Minute))
	state, _, err = deduplicator.Reserve(context.Background(), "event", time.Second)
	require.NoError(t, err)
	require.Equal(t, WebhookDedupCompleted, state)
}

func TestMemoryWebhookDeduplicatorDoesNotEvictInFlightReservation(t *testing.T) {
	deduplicator := NewMemoryWebhookDeduplicator(1)
	state, reservation, err := deduplicator.Reserve(context.Background(), "first", time.Minute)
	require.NoError(t, err)
	require.Equal(t, WebhookDedupAcquired, state)
	_, _, err = deduplicator.Reserve(context.Background(), "second", time.Minute)
	require.ErrorIs(t, err, ErrWebhookDedupCapacity)
	require.NoError(t, deduplicator.Release(context.Background(), "first", reservation))
	state, _, err = deduplicator.Reserve(context.Background(), "second", time.Minute)
	require.NoError(t, err)
	require.Equal(t, WebhookDedupAcquired, state)
}

type duplicateWebhookStore struct {
	calls atomic.Int32
}

func (s *duplicateWebhookStore) CheckAndStore(context.Context, int, time.Duration) (bool, error) {
	return s.calls.Add(1) > 1, nil
}

func TestWebhookInjectedDeduplicator(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	store := &duplicateWebhookStore{}
	_, err := NewWebhookHandlerWithError(client, "", "verify", WithWebhookDeduplicator(store))
	require.ErrorContains(t, err, "事务型去重器")
	handler, err := NewWebhookHandlerWithError(
		client,
		"",
		"verify",
		WithWebhookDeduplicator(store),
		WithWebhookAckMode(WebhookAckAfterEnqueue),
	)
	require.NoError(t, err)
	defer func() { _ = handler.Shutdown(context.Background()) }()
	payload := webhookPayload(t, 7, textWebhookEvent("verify", "hello"))
	require.Equal(t, http.StatusOK, performWebhookRequest(handler, payload, "").Code)
	require.Equal(t, http.StatusOK, performWebhookRequest(handler, payload, "").Code)
	require.Equal(t, int32(2), store.calls.Load())
}

func TestWebhookHandlerPanicAndConcurrency(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler := NewWebhookHandler(client, "", "verify")
	defer func() { _ = handler.Shutdown(context.Background()) }()
	const total = 24
	received := make(chan int, total)
	handler.OnMessage(MessageTypeText, func(event *MessageEvent) { received <- event.SN })

	var requestGroup sync.WaitGroup
	statuses := make(chan int, total)
	for index := 1; index <= total; index++ {
		index := index
		requestGroup.Add(1)
		go func() {
			defer requestGroup.Done()
			payload := webhookPayload(t, index, textWebhookEvent("verify", fmt.Sprintf("message-%d", index)))
			statuses <- performWebhookRequest(handler, payload, "").Code
		}()
	}
	requestGroup.Wait()
	close(statuses)
	for status := range statuses {
		require.Equal(t, http.StatusOK, status)
	}

	seen := make(map[int]struct{}, total)
	deadline := time.After(2 * time.Second)
	for len(seen) < total {
		select {
		case sn := <-received:
			seen[sn] = struct{}{}
		case <-deadline:
			t.Fatalf("received %d/%d concurrent webhooks", len(seen), total)
		}
	}
}

func TestWebhookHandlerPanicReleasesDedupReservation(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler := NewWebhookHandler(client, "", "verify")
	defer func() { _ = handler.Shutdown(context.Background()) }()
	var calls atomic.Int32
	handler.OnMessage(MessageTypeText, func(*MessageEvent) {
		calls.Add(1)
		panic("expected panic")
	})
	payload := webhookPayload(t, 99, textWebhookEvent("verify", "retry-after-panic"))

	require.Equal(t, http.StatusInternalServerError, performWebhookRequest(handler, payload, "").Code)
	require.Equal(t, http.StatusInternalServerError, performWebhookRequest(handler, payload, "").Code)
	require.Equal(t, int32(2), calls.Load())
}

func TestWebhookRejectsInvalidConfigurationAndOversizedBodies(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()

	_, err := NewWebhookHandlerWithError(client, "", "")
	require.ErrorContains(t, err, "verifyToken")
	invalid := NewWebhookHandler(client, "", "")
	recorder := performWebhookRequest(invalid, []byte(`{}`), "")
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	shortKeyHandler, err := NewWebhookHandlerWithError(client, "short", "verify")
	require.NoError(t, err)
	require.NoError(t, shortKeyHandler.Shutdown(context.Background()))
	_, err = NewWebhookHandlerWithError(client, strings.Repeat("x", 33), "verify")
	require.ErrorContains(t, err, "encryptKey")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookDeduplicator(nil))
	require.ErrorContains(t, err, "去重器")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookTransactionalDeduplicator(nil))
	require.ErrorContains(t, err, "事务型去重器")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookDedupTTL(0))
	require.ErrorContains(t, err, "TTL")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookProcessingTTL(0))
	require.ErrorContains(t, err, "处理租约")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookAckMode(WebhookAckMode(99)))
	require.ErrorContains(t, err, "确认模式")
	_, err = NewWebhookHandlerWithError(client, "", "verify", WithWebhookDispatch(1, 2))
	require.ErrorContains(t, err, "派发")

	handler, err := NewWebhookHandlerWithError(client, "", "verify", WithWebhookBodyLimits(64, 128))
	require.NoError(t, err)
	defer func() { _ = handler.Shutdown(context.Background()) }()
	recorder = performWebhookRequest(handler, bytes.Repeat([]byte("x"), 65), "")
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)

	bomb := gzipWebhookPayload(t, bytes.Repeat([]byte("x"), 1024))
	require.Less(t, len(bomb), 64)
	recorder = performWebhookRequest(handler, bomb, "gzip")
	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
}

func TestWebhookQueueBackpressureDoesNotDeduplicateRejectedEvent(t *testing.T) {
	client := newWebhookTestClient(t)
	defer func() { _ = client.Close() }()
	handler, err := NewWebhookHandlerWithError(client, "", "verify", WithWebhookDispatch(1, 1))
	require.NoError(t, err)

	started := make(chan struct{})
	release := make(chan struct{})
	received := make(chan int, 2)
	handler.OnMessage(MessageTypeText, func(event *MessageEvent) {
		if event.SN == 1 {
			close(started)
			<-release
		}
		received <- event.SN
	})

	first := webhookPayload(t, 1, textWebhookEvent("verify", "first"))
	second := webhookPayload(t, 2, textWebhookEvent("verify", "second"))
	firstResult := make(chan int, 1)
	go func() {
		firstResult <- performWebhookRequest(handler, first, "").Code
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not start")
	}
	require.Equal(t, http.StatusServiceUnavailable, performWebhookRequest(handler, second, "").Code)
	close(release)
	require.Equal(t, http.StatusOK, <-firstResult)
	require.Equal(t, 1, receiveInt(t, received))

	require.Equal(t, http.StatusOK, performWebhookRequest(handler, second, "").Code)
	require.Equal(t, 2, receiveInt(t, received))
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	require.NoError(t, handler.Shutdown(ctx))
	require.Equal(t, http.StatusServiceUnavailable, performWebhookRequest(handler, second, "").Code)
}
