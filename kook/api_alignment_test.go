package kook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func newTestClient(handler http.HandlerFunc) (*Client, func()) {
	server := httptest.NewServer(handler)
	client := NewClient(
		"test-token",
		WithBaseURL(server.URL),
		WithoutRateLimit(),
		WithoutRetry(),
	)
	return client, server.Close
}

func writeKOOKResponse(t *testing.T, w http.ResponseWriter, data interface{}) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"code":    0,
		"message": "操作成功",
		"data":    data,
	}); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func readBodyMap(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func TestKickGuildMemberUsesTargetID(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/guild/kickout" {
			t.Fatalf("path = %s, want /v3/guild/kickout", r.URL.Path)
		}

		body := readBodyMap(t, r)
		if body["guild_id"] != "guild-1" {
			t.Fatalf("guild_id = %v, want guild-1", body["guild_id"])
		}
		if body["target_id"] != "user-1" {
			t.Fatalf("target_id = %v, want user-1", body["target_id"])
		}
		if _, exists := body["user_id"]; exists {
			t.Fatalf("unexpected user_id in body: %#v", body)
		}

		writeKOOKResponse(t, w, map[string]interface{}{})
	})
	defer closeServer()

	if err := client.Guild.KickGuildMember(context.Background(), "guild-1", "user-1"); err != nil {
		t.Fatalf("KickGuildMember returned error: %v", err)
	}
}

func TestMoveUsersUsesOfficialBody(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/channel/move-user" {
			t.Fatalf("path = %s, want /v3/channel/move-user", r.URL.Path)
		}

		body := readBodyMap(t, r)
		if body["target_id"] != "voice-1" {
			t.Fatalf("target_id = %v, want voice-1", body["target_id"])
		}
		userIDs, ok := body["user_ids"].([]interface{})
		if !ok {
			t.Fatalf("user_ids type = %T, want []interface{}", body["user_ids"])
		}
		if !reflect.DeepEqual(userIDs, []interface{}{"user-1", "user-2"}) {
			t.Fatalf("user_ids = %#v, want user-1/user-2", userIDs)
		}
		if _, exists := body["channel_id"]; exists {
			t.Fatalf("unexpected channel_id in body: %#v", body)
		}
		if _, exists := body["user_id"]; exists {
			t.Fatalf("unexpected user_id in body: %#v", body)
		}

		writeKOOKResponse(t, w, []interface{}{})
	})
	defer closeServer()

	if err := client.Channel.MoveUsers(context.Background(), "voice-1", []string{"user-1", "user-2"}); err != nil {
		t.Fatalf("MoveUsers returned error: %v", err)
	}
}

func TestVoiceJoinParsesOfficialResponse(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/voice/join" {
			t.Fatalf("path = %s, want /v3/voice/join", r.URL.Path)
		}
		body := readBodyMap(t, r)
		if body["channel_id"] != "voice-1" {
			t.Fatalf("channel_id = %v, want voice-1", body["channel_id"])
		}

		writeKOOKResponse(t, w, map[string]interface{}{
			"ip":         "127.0.0.1",
			"port":       "1000",
			"rtcp_port":  "1001",
			"rtcp_mux":   false,
			"bitrate":    48000,
			"audio_ssrc": "1111",
			"audio_pt":   "111",
		})
	})
	defer closeServer()

	info, err := client.Voice.JoinVoiceChannel(context.Background(), "voice-1")
	if err != nil {
		t.Fatalf("JoinVoiceChannel returned error: %v", err)
	}
	if info.IP != "127.0.0.1" || info.Port != "1000" || info.RTCPPort != "1001" {
		t.Fatalf("unexpected voice endpoint: %#v", info)
	}
	if info.RTCPMux || info.Bitrate != 48000 || info.AudioSSRC != "1111" || info.AudioPT != "111" {
		t.Fatalf("unexpected voice metadata: %#v", info)
	}
}

func TestGetJoinedVoiceChannelsParsesListResponse(t *testing.T) {
	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v3/voice/list" {
			t.Fatalf("path = %s, want /v3/voice/list", r.URL.Path)
		}

		writeKOOKResponse(t, w, map[string]interface{}{
			"items": []map[string]interface{}{
				{
					"id":        "voice-1",
					"guild_id":  "guild-1",
					"parent_id": "parent-1",
					"name":      "语音频道",
				},
			},
			"meta": map[string]interface{}{
				"page":       1,
				"page_total": 1,
				"page_size":  50,
				"total":      1,
			},
			"sort": map[string]interface{}{},
		})
	})
	defer closeServer()

	channels, err := client.Voice.GetJoinedVoiceChannels(context.Background())
	if err != nil {
		t.Fatalf("GetJoinedVoiceChannels returned error: %v", err)
	}
	if len(channels.Items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(channels.Items))
	}
	if channels.Items[0].ID != "voice-1" || channels.Items[0].GuildID != "guild-1" {
		t.Fatalf("unexpected channel: %#v", channels.Items[0])
	}
	if channels.Meta.Total != 1 {
		t.Fatalf("meta.total = %d, want 1", channels.Meta.Total)
	}
}

func TestPermissionConstantsMatchOfficialBits(t *testing.T) {
	cases := map[string]int{
		"administrator":   PermissionAdministrator,
		"manage_guild":    PermissionManageGuild,
		"view_channel":    PermissionViewChannel,
		"send_messages":   PermissionSendMessages,
		"manage_messages": PermissionManageMessages,
		"upload_files":    PermissionUploadFiles,
		"connect_voice":   PermissionConnectVoice,
		"manage_voice":    PermissionManageVoice,
	}

	want := map[string]int{
		"administrator":   1,
		"manage_guild":    2,
		"view_channel":    2048,
		"send_messages":   4096,
		"manage_messages": 8192,
		"upload_files":    16384,
		"connect_voice":   32768,
		"manage_voice":    65536,
	}

	for name, got := range cases {
		if got != want[name] {
			t.Fatalf("%s = %d, want %d", name, got, want[name])
		}
	}
}
