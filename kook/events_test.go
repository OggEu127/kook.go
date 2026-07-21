package kook

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllMessageTypesPreserveRawContent(t *testing.T) {
	tests := []struct {
		eventType MessageType
		content   string
		text      string
	}{
		{MessageTypeText, `"text"`, "text"},
		{MessageTypeImage, `"https://example.test/image.png"`, "https://example.test/image.png"},
		{MessageTypeVideo, `"https://example.test/video.mp4"`, "https://example.test/video.mp4"},
		{MessageTypeFile, `"https://example.test/file.zip"`, "https://example.test/file.zip"},
		{MessageTypeAudio, `"https://example.test/audio.ogg"`, "https://example.test/audio.ogg"},
		{MessageTypeKMD, `"**kmarkdown**"`, "**kmarkdown**"},
		{MessageTypeCard, `"[{\"type\":\"card\"}]"`, `[{"type":"card"}]`},
		{MessageTypeItem, `{"type":"item","data":{"user_id":"u","target_id":"t","item_id":"i"}}`, ""},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("type_%d", test.eventType), func(t *testing.T) {
			payload := fmt.Sprintf(`{"channel_type":"GROUP","type":%d,"content":%s,"extra":{},"msg_id":"m"}`, test.eventType, test.content)
			var event Event
			require.NoError(t, json.Unmarshal([]byte(payload), &event))
			require.JSONEq(t, test.content, string(event.Content))

			dispatcher := newEventDispatcher()
			var messageCalls, anyCalls atomic.Int32
			dispatcher.onMessage(test.eventType, func(received *MessageEvent) {
				messageCalls.Add(1)
				require.Equal(t, event.MsgID, received.MsgID)
			})
			dispatcher.onAnyEvent(func(received *Event) {
				anyCalls.Add(1)
				require.Equal(t, test.eventType, received.Type)
			})
			require.NoError(t, dispatcher.dispatch(&event, nil))
			require.Equal(t, int32(1), messageCalls.Load())
			require.Equal(t, int32(1), anyCalls.Load())

			if test.eventType == MessageTypeItem {
				_, err := event.TextContent()
				require.Error(t, err)
				var item struct {
					Type string `json:"type"`
					Data struct {
						UserID   string `json:"user_id"`
						TargetID string `json:"target_id"`
						ItemID   string `json:"item_id"`
					} `json:"data"`
				}
				require.NoError(t, event.DecodeContent(&item))
				require.Equal(t, "item", item.Type)
				require.Equal(t, "i", item.Data.ItemID)
				return
			}
			content, err := event.TextContent()
			require.NoError(t, err)
			require.Equal(t, test.text, content)
		})
	}
}

func TestUpdatedGuildFixtureKeepsCompleteBody(t *testing.T) {
	fixture := `{
		"s":0,
		"sn":9,
		"d":{
			"channel_type":"GROUP",
			"type":255,
			"target_id":"601630000000",
			"author_id":"1",
			"content":"[系统消息]",
			"extra":{"type":"updated_guild","body":{
				"id":"601630000000","name":"test111","user_id":"2418xxx",
				"icon":"https://example.test/icon.png","notify_type":1,"region":"shanghai",
				"enable_open":1,"open_id":1123123123,
				"default_channel_id":"4881800000000","welcome_channel_id":"4881800000000"
			}},
			"msg_id":"0108feaf","msg_timestamp":1612764956322,"nonce":"","verify_token":"verify"
		}
	}`
	var envelope WebhookMessage
	require.NoError(t, json.Unmarshal([]byte(fixture), &envelope))
	var event Event
	require.NoError(t, json.Unmarshal(envelope.D, &event))
	event.S, event.SN = envelope.S, envelope.SN

	systemEvent, err := event.AsSystemEvent()
	require.NoError(t, err)
	require.Equal(t, SystemEventUpdatedGuild, systemEvent.Type)
	var guild Guild
	require.NoError(t, systemEvent.DecodeBody(&guild))
	require.Equal(t, "601630000000", guild.ID)
	require.Equal(t, "1123123123", guild.OpenID)
	require.True(t, guild.EnableOpen)
	require.Equal(t, "4881800000000", guild.WelcomeChannelID)
}

func TestAllSystemEventTypesDispatchByExtraType(t *testing.T) {
	eventTypes := []SystemEventType{
		SystemEventAddedReaction,
		SystemEventDeletedReaction,
		SystemEventUpdatedMessage,
		SystemEventDeletedMessage,
		SystemEventAddedChannel,
		SystemEventUpdatedChannel,
		SystemEventDeletedChannel,
		SystemEventPinnedMessage,
		SystemEventUnpinnedMessage,
		SystemEventUpdatedPrivateMessage,
		SystemEventDeletedPrivateMessage,
		SystemEventPrivateAddedReaction,
		SystemEventPrivateDeletedReaction,
		SystemEventJoinedGuild,
		SystemEventExitedGuild,
		SystemEventUpdatedGuildMember,
		SystemEventGuildMemberOnline,
		SystemEventGuildMemberOffline,
		SystemEventAddedRole,
		SystemEventDeletedRole,
		SystemEventUpdatedRole,
		SystemEventUpdatedGuild,
		SystemEventDeletedGuild,
		SystemEventAddedBlockList,
		SystemEventDeletedBlockList,
		SystemEventAddedEmoji,
		SystemEventRemovedEmoji,
		SystemEventUpdatedEmoji,
		SystemEventJoinedChannel,
		SystemEventExitedChannel,
		SystemEventUserUpdated,
		SystemEventSelfJoinedGuild,
		SystemEventSelfExitedGuild,
		SystemEventMessageButtonClick,
	}
	require.Len(t, eventTypes, 34)

	for sequence, eventType := range eventTypes {
		t.Run(string(eventType), func(t *testing.T) {
			payload := fmt.Sprintf(`{"channel_type":"GROUP","type":255,"content":"[系统消息]","extra":{"type":%q,"body":{"sequence":%d}}}`, eventType, sequence)
			var event Event
			require.NoError(t, json.Unmarshal([]byte(payload), &event))

			dispatcher := newEventDispatcher()
			called := false
			dispatcher.onSystemEvent(eventType, func(received *SystemEvent) {
				called = true
				require.Equal(t, eventType, received.Type)
				var body struct {
					Sequence int `json:"sequence"`
				}
				require.NoError(t, received.DecodeBody(&body))
				require.Equal(t, sequence, body.Sequence)
			})
			require.NoError(t, dispatcher.dispatch(&event, nil))
			require.True(t, called)
		})
	}
}

func TestEventDispatcherContainsHandlerPanics(t *testing.T) {
	dispatcher := newEventDispatcher()
	var panicCalls, safeCalls atomic.Int32
	dispatcher.onMessage(MessageTypeText, func(*MessageEvent) { panic("handler failed") })
	dispatcher.onMessage(MessageTypeText, func(*MessageEvent) { safeCalls.Add(1) })

	event := &Event{Type: MessageTypeText, Content: json.RawMessage(`"hello"`)}
	require.ErrorIs(t, dispatcher.dispatch(event, func(any) { panicCalls.Add(1) }), ErrEventHandlerPanic)
	require.Equal(t, int32(1), panicCalls.Load())
	require.Equal(t, int32(1), safeCalls.Load())
}

func TestMalformedSystemEventStillReachesAnyHandler(t *testing.T) {
	dispatcher := newEventDispatcher()
	var calls atomic.Int32
	dispatcher.onAnyEvent(func(*Event) { calls.Add(1) })
	err := dispatcher.dispatch(&Event{Type: MessageTypeSystem, Extra: json.RawMessage(`{}`)}, nil)
	require.Error(t, err)
	require.Equal(t, int32(1), calls.Load())
}
