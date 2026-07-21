package kook

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type endpointContract struct {
	method   string
	endpoint string
	special  string
}

var officialEndpointContracts = []endpointContract{
	{http.MethodGet, "gateway/index", ""},
	{http.MethodGet, "user/me", ""},
	{http.MethodGet, "user/view", ""},
	{http.MethodPost, "user/offline", ""},
	{http.MethodPost, "user/online", ""},
	{http.MethodGet, "user/get-online-status", ""},
	{http.MethodGet, "guild/list", ""},
	{http.MethodGet, "guild/view", ""},
	{http.MethodGet, "guild/user-list", ""},
	{http.MethodPost, "guild/nickname", ""},
	{http.MethodPost, "guild/leave", ""},
	{http.MethodPost, "guild/kickout", ""},
	{http.MethodGet, "guild-mute/list", ""},
	{http.MethodPost, "guild-mute/create", ""},
	{http.MethodPost, "guild-mute/delete", ""},
	{http.MethodGet, "guild-boost/history", ""},
	{http.MethodGet, "channel/list", ""},
	{http.MethodGet, "channel/view", ""},
	{http.MethodPost, "channel/create", ""},
	{http.MethodPost, "channel/update", ""},
	{http.MethodPost, "channel/delete", ""},
	{http.MethodGet, "channel/user-list", ""},
	{http.MethodPost, "channel/move-user", ""},
	{http.MethodPost, "channel/kickout", ""},
	{http.MethodGet, "channel-role/index", ""},
	{http.MethodPost, "channel-role/create", ""},
	{http.MethodPost, "channel-role/update", ""},
	{http.MethodPost, "channel-role/sync", ""},
	{http.MethodPost, "channel-role/delete", ""},
	{http.MethodGet, "channel-user/get-joined-channel", ""},
	{http.MethodGet, "message/list", ""},
	{http.MethodGet, "message/view", ""},
	{http.MethodPost, "message/create", ""},
	{http.MethodPost, "message/update", ""},
	{http.MethodPost, "message/delete", ""},
	{http.MethodGet, "message/reaction-list", ""},
	{http.MethodPost, "message/add-reaction", ""},
	{http.MethodPost, "message/delete-reaction", ""},
	{http.MethodPost, "message/send-pipemsg", ""},
	{http.MethodPost, "message/pin", ""},
	{http.MethodPost, "message/unpin", ""},
	{http.MethodGet, "user-chat/list", ""},
	{http.MethodGet, "user-chat/view", ""},
	{http.MethodPost, "user-chat/create", ""},
	{http.MethodPost, "user-chat/delete", ""},
	{http.MethodGet, "direct-message/list", ""},
	{http.MethodGet, "direct-message/view", ""},
	{http.MethodPost, "direct-message/create", ""},
	{http.MethodPost, "direct-message/update", ""},
	{http.MethodPost, "direct-message/delete", ""},
	{http.MethodGet, "direct-message/reaction-list", ""},
	{http.MethodPost, "direct-message/add-reaction", ""},
	{http.MethodPost, "direct-message/delete-reaction", ""},
	{http.MethodGet, "guild-role/list", ""},
	{http.MethodPost, "guild-role/create", ""},
	{http.MethodPost, "guild-role/update", ""},
	{http.MethodPost, "guild-role/delete", ""},
	{http.MethodPost, "guild-role/grant", ""},
	{http.MethodPost, "guild-role/revoke", ""},
	{http.MethodGet, "guild-emoji/list", ""},
	{http.MethodPost, "guild-emoji/create", ""},
	{http.MethodPost, "guild-emoji/update", ""},
	{http.MethodPost, "guild-emoji/delete", ""},
	{http.MethodGet, "blacklist/list", ""},
	{http.MethodPost, "blacklist/create", ""},
	{http.MethodPost, "blacklist/delete", ""},
	{http.MethodGet, "invite/list", ""},
	{http.MethodPost, "invite/create", ""},
	{http.MethodPost, "invite/delete", ""},
	{http.MethodPost, "asset/create", ""},
	{http.MethodGet, "intimacy/index", ""},
	{http.MethodPost, "intimacy/update", ""},
	{http.MethodGet, "friend", ""},
	{http.MethodPost, "friend/request", ""},
	{http.MethodPost, "friend/handle-request", ""},
	{http.MethodPost, "friend/delete", ""},
	{http.MethodPost, "friend/block", ""},
	{http.MethodPost, "friend/unblock", ""},
	{http.MethodGet, "game", ""},
	{http.MethodPost, "game/create", ""},
	{http.MethodPost, "game/update", ""},
	{http.MethodPost, "game/delete", ""},
	{http.MethodPost, "game/activity", ""},
	{http.MethodPost, "game/delete-activity", ""},
	{http.MethodGet, "badge/guild", "badge"},
	{http.MethodGet, "category/list", ""},
	{http.MethodPost, "thread/create", ""},
	{http.MethodPost, "thread/reply", ""},
	{http.MethodGet, "thread/view", ""},
	{http.MethodGet, "thread/list", ""},
	{http.MethodPost, "thread/delete", ""},
	{http.MethodGet, "thread/post", ""},
	{http.MethodPost, "voice/join", ""},
	{http.MethodGet, "voice/list", ""},
	{http.MethodPost, "voice/leave", ""},
	{http.MethodPost, "voice/keep-alive", ""},
	{http.MethodGet, "template/list", ""},
	{http.MethodPost, "template/create", ""},
	{http.MethodPost, "template/update", ""},
	{http.MethodPost, "template/delete", ""},
	{http.MethodPost, "oauth2/token", "oauth"},
}

func TestOfficialEndpointInventoryAndTransportContract(t *testing.T) {
	require.Len(t, officialEndpointContracts, 101)
	seen := make(map[string]struct{}, len(officialEndpointContracts))
	for _, contract := range officialEndpointContracts {
		key := contract.method + " " + contract.endpoint
		_, duplicate := seen[key]
		require.Falsef(t, duplicate, "duplicate endpoint contract: %s", key)
		seen[key] = struct{}{}
		if contract.special != "" || contract.endpoint == "asset/create" {
			continue
		}

		contract := contract
		t.Run(strings.ReplaceAll(key, "/", "_"), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, contract.method, r.Method)
				require.Equal(t, "/api/v3/"+contract.endpoint, r.URL.Path)
				require.Equal(t, "Bot test-token", r.Header.Get("Authorization"))
				if r.Method == http.MethodGet {
					require.Equal(t, "value", r.URL.Query().Get("contract"))
				} else {
					require.Equal(t, "application/json", r.Header.Get("Content-Type"))
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, "value", body["contract"])
				}
				_, _ = io.WriteString(w, `{"code":0,"message":"ok","data":{}}`)
			}))
			defer server.Close()

			client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
			defer func() { _ = client.Close() }()
			if contract.method == http.MethodGet {
				_, err := client.Get(context.Background(), contract.endpoint, map[string]string{"contract": "value"})
				require.NoError(t, err)
				return
			}
			_, err := client.Post(context.Background(), contract.endpoint, map[string]interface{}{"contract": "value"})
			require.NoError(t, err)
		})
	}
}

func TestUserDecorationsIDMapAcceptsScalarAndArrayValues(t *testing.T) {
	var user User
	require.NoError(t, json.Unmarshal([]byte(`{
		"id":"user",
		"decorations_id_map":{
			"background":1,
			"join_voice":"2",
			"nameplates":[3,"4"],
			"empty":[],
			"missing":null
		}
	}`), &user))

	require.Equal(t, map[string]int{
		"background": 1,
		"join_voice": 2,
		"nameplates": 3,
	}, user.DecorationsIDMap)
	require.Equal(t, map[string][]int{
		"background": {1},
		"join_voice": {2},
		"nameplates": {3, 4},
		"empty":      {},
		"missing":    nil,
	}, user.DecorationIDs)

	require.NoError(t, json.Unmarshal([]byte(`{"decorations_id_map":[]}`), &user))
	require.Empty(t, user.DecorationsIDMap)
	require.Empty(t, user.DecorationIDs)
}

func TestOAuthTransportAndExpiryCompatibility(t *testing.T) {
	for _, expiryField := range []string{"expire_in", "expires_in"} {
		t.Run(expiryField, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/api/oauth2/token", r.URL.Path)
				require.Empty(t, r.Header.Get("Authorization"))
				var body OAuthTokenParams
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "authorization_code", body.GrantType)
				_, _ = io.WriteString(w, `{"access_token":"access","token_type":"Bearer","`+expiryField+`":3600,"scope":"get_user_info"}`)
			}))
			defer server.Close()

			client := NewOAuthClient(WithOAuthBaseURL(server.URL + "/api"))
			token, err := client.ExchangeToken(context.Background(), OAuthTokenParams{
				ClientID: "client", ClientSecret: "secret", Code: "code", RedirectURI: "https://example.test/callback",
			})
			require.NoError(t, err)
			require.Equal(t, 3600, token.ExpiresIn)
		})
	}
}

func TestBadgeReturnsBinaryMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v3/badge/guild", r.URL.Path)
		require.Equal(t, "guild", r.URL.Query().Get("guild_id"))
		require.Equal(t, "2", r.URL.Query().Get("style"))
		require.Equal(t, "Bot test-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("ETag", `"badge-etag"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{0x89, 'P', 'N', 'G'})
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
	response, err := client.Badge.GetGuildBadge(context.Background(), BadgeParams{GuildID: "guild", Style: BadgeStyleOnlineAndTotal})
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Equal(t, "image/png", response.ContentType)
	require.Equal(t, `"badge-etag"`, response.ETag)
	require.Equal(t, []byte{0x89, 'P', 'N', 'G'}, response.Data)
}

func testPtr[T any](value T) *T { return &value }

func TestExplicitZeroValuesAreSent(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		data   json.RawMessage
		invoke func(*Client) error
		assert func(*testing.T, map[string]any)
	}{
		{
			name: "role", path: "/api/v3/guild-role/update", data: json.RawMessage(`{}`),
			invoke: func(client *Client) error {
				_, err := client.GuildRole.UpdateRole(context.Background(), UpdateRoleParams{
					GuildID: "guild", RoleID: 1,
					Name: testPtr(""), Color: testPtr(0), Hoist: testPtr(0), Mentionable: testPtr(0), Permissions: testPtr(0),
				})
				return err
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Contains(t, body, "name")
				require.Equal(t, float64(0), body["color"])
				require.Equal(t, float64(0), body["permissions"])
			},
		},
		{
			name: "channel", path: "/api/v3/channel/update", data: json.RawMessage(`{}`),
			invoke: func(client *Client) error {
				_, err := client.Channel.UpdateChannel(context.Background(), UpdateChannelParams{
					ChannelID: "channel",
					Name:      testPtr(""), Level: testPtr(0), ParentID: testPtr(""), Topic: testPtr(""), SlowMode: testPtr(0),
					LimitAmount: testPtr(0), VoiceQuality: testPtr(""), Password: testPtr(""),
				})
				return err
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Contains(t, body, "name")
				require.Equal(t, float64(0), body["level"])
				require.Contains(t, body, "password")
			},
		},
		{
			name: "template", path: "/api/v3/template/update", data: json.RawMessage(`{"model":{}}`),
			invoke: func(client *Client) error {
				_, err := client.Template.UpdateTemplate(context.Background(), UpdateTemplateParams{
					ID: "template", Title: testPtr(""), Content: testPtr(""), Type: testPtr(0), MsgType: testPtr(0),
					TestData: testPtr(""), TestChannel: testPtr(""),
				})
				return err
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Equal(t, float64(0), body["type"])
				require.Equal(t, "", body["test_data"])
			},
		},
		{
			name: "invite", path: "/api/v3/invite/create", data: json.RawMessage(`{}`),
			invoke: func(client *Client) error {
				_, err := client.Invite.CreateInvite(context.Background(), CreateInviteParams{
					GuildID: "guild", Duration: testPtr(0), SettingTimes: testPtr(-1),
				})
				return err
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Equal(t, float64(0), body["duration"])
				require.Equal(t, float64(-1), body["setting_times"])
			},
		},
		{
			name: "intimacy", path: "/api/v3/intimacy/update", data: json.RawMessage(`{}`),
			invoke: func(client *Client) error {
				return client.Intimacy.UpdateIntimacy(context.Background(), UpdateIntimacyParams{
					UserID: "user", Score: testPtr(0), SocialInfo: testPtr(""), ImgID: testPtr(""),
				})
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Equal(t, float64(0), body["score"])
				require.Contains(t, body, "social_info")
			},
		},
		{
			name: "channel-role", path: "/api/v3/channel-role/update", data: json.RawMessage(`{}`),
			invoke: func(client *Client) error {
				_, err := client.Channel.UpdateChannelRole(context.Background(), UpdateChannelRoleParams{
					ChannelID: "channel", Type: "role_id", Value: "0", Allow: testPtr(0), Deny: testPtr(0),
				})
				return err
			},
			assert: func(t *testing.T, body map[string]any) {
				require.Equal(t, float64(0), body["allow"])
				require.Equal(t, float64(0), body["deny"])
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, test.path, r.URL.Path)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				test.assert(t, body)
				response := map[string]any{"code": 0, "message": "ok", "data": test.data}
				encoded, err := json.Marshal(response)
				require.NoError(t, err)
				_, _ = w.Write(encoded)
			}))
			defer server.Close()
			client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
			require.NoError(t, test.invoke(client))
		})
	}
}

func TestMixedResponseShapes(t *testing.T) {
	var user User
	require.NoError(t, json.Unmarshal([]byte(`{"id":123,"online":1,"bot":0,"is_vip":1,"mobile_verified":0,"roles":[1,"2"]}`), &user))
	require.Equal(t, "123", user.ID)
	require.True(t, user.Online)
	require.False(t, user.Bot)
	require.Equal(t, []int{1, 2}, user.Roles)

	var guild Guild
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"user_id":2,"enable_open":1,"open_id":123,"notify_type":"1","default_channel_id":3}`), &guild))
	require.True(t, guild.EnableOpen)
	require.Equal(t, "1", guild.ID)
	require.Equal(t, "123", guild.OpenID)

	var channel Channel
	require.NoError(t, json.Unmarshal([]byte(`{"id":123,"guild_id":456,"user_id":789,"parent_id":0,"voice_quality":2,"is_category":1,"has_password":0,"is_private":1,"type":"2","children":[1,"2"]}`), &channel))
	require.Equal(t, "123", channel.ID)
	require.Equal(t, "0", channel.ParentID)
	require.Equal(t, "2", channel.VoiceQuality)
	require.True(t, channel.IsCategory)
	require.True(t, channel.IsPrivate)
	require.Equal(t, 2, channel.Type)
	require.Equal(t, []string{"1", "2"}, channel.Children)

	var message Message
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"author_id":2,"mention":[3,"4"],"mention_roles":[5,"6"],"mention_all":1,"attachments":{"type":"image","url":"u","size":1},"reactions":[{"emoji":{"id":"e"},"count":1,"me":1}]}`), &message))
	require.Equal(t, "1", message.ID)
	require.Equal(t, []string{"3", "4"}, message.Mention)
	require.True(t, message.MentionAll)
	require.Len(t, message.Attachments, 1)
	require.Equal(t, "u", message.Attachments[0].URL)
	require.True(t, message.Reactions[0].Me)
	require.NoError(t, json.Unmarshal([]byte(`{"id":"m","attachments":false}`), &message))
	require.Empty(t, message.Attachments)

	var list ListChannelsResponse
	require.NoError(t, json.Unmarshal([]byte(`{"items":[],"meta":{},"sort":[]}`), &list))
	require.NotNil(t, list.Sort)

	var voice VoiceConnectionInfo
	require.NoError(t, json.Unmarshal([]byte(`{"port":1000,"rtcp_port":"1001"}`), &voice))
	require.Equal(t, "1000", voice.Port)
	require.Equal(t, "1001", voice.RTCPPort)

	var thread Thread
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"post_id":2,"category":{"id":7,"roles":[]},"medias":[{"type":2,"src":"u"}],"is_updated":1,"mention":[8,"9"]}`), &thread))
	require.Equal(t, "7", thread.Category.ID)
	require.Equal(t, "2", thread.Medias[0].Type)
	require.True(t, thread.IsUpdated)
	require.Equal(t, []string{"8", "9"}, thread.Mention)

	var post Post
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"thread_id":2,"mention":[3],"mention_here":1}`), &post))
	require.Equal(t, "1", post.ID)
	require.True(t, post.MentionHere)

	var friends FriendsListResponse
	require.NoError(t, json.Unmarshal([]byte(`{"friend":[],"request":[],"blocked":[{"id":1,"type":"block","friend_info":{"id":"u"}}]}`), &friends))
	require.Len(t, friends.Block, 1)
}

func TestKOOKErrorClassificationAndContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Request-Id", "request-id")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":40300,"message":"forbidden"}`)
	}))
	defer server.Close()

	client := NewClient("test-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
	defer func() { _ = client.Close() }()
	_, err := client.User.GetMe(context.Background())
	require.Error(t, err)
	apiErr, ok := IsKOOKError(err)
	require.True(t, ok)
	require.Equal(t, 40300, apiErr.Code)
	require.Equal(t, http.StatusForbidden, apiErr.HTTPStatus)
	require.Equal(t, http.MethodGet, apiErr.Method)
	require.Equal(t, "user/me", apiErr.Endpoint)
	require.Equal(t, "request-id", apiErr.RequestID)
	require.True(t, apiErr.IsPermissionError())
	require.False(t, apiErr.IsServerError())
	require.False(t, apiErr.IsRetryable())
}
