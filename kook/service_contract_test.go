package kook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type serviceContract struct {
	Name            string
	Method          string
	Endpoint        string
	Query           map[string]string
	Body            map[string]any
	MultipartFields map[string]string
	MultipartFiles  map[string][]byte
	MultipartNames  map[string]string
	Data            string
	Binary          []byte
	OAuth           bool
	Invoke          func(context.Context, *Client, string) error
}

func TestOfficialServiceContracts(t *testing.T) {
	contracts := officialServiceContracts()
	require.Len(t, contracts, 102)
	seen := make(map[string]struct{}, len(contracts))

	for _, contract := range contracts {
		contract := contract
		key := contract.Method + " " + contract.Endpoint
		require.NotContains(t, seen, key)
		seen[key] = struct{}{}

		t.Run(contract.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, contract.Method, r.Method)
				expectedPath := "/api/v3/" + contract.Endpoint
				if contract.OAuth {
					expectedPath = "/api/" + contract.Endpoint
					require.Empty(t, r.Header.Get("Authorization"))
				} else {
					require.Equal(t, "Bot contract-token", r.Header.Get("Authorization"))
				}
				require.Equal(t, expectedPath, r.URL.Path)
				require.Equal(t, queryValues(contract.Query), r.URL.Query())

				switch {
				case contract.MultipartFiles != nil:
					require.NoError(t, r.ParseMultipartForm(2<<20))
					for name, value := range contract.MultipartFields {
						require.Equal(t, value, r.FormValue(name))
					}
					for name, content := range contract.MultipartFiles {
						file, header, err := r.FormFile(name)
						require.NoError(t, err)
						if expectedName := contract.MultipartNames[name]; expectedName != "" {
							require.Equal(t, expectedName, header.Filename)
						}
						actual, err := io.ReadAll(file)
						require.NoError(t, err)
						require.NoError(t, file.Close())
						require.Equal(t, content, actual)
					}
				case contract.Body != nil:
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					expected, err := json.Marshal(contract.Body)
					require.NoError(t, err)
					require.JSONEq(t, string(expected), string(body))
				default:
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					require.Empty(t, body)
				}

				if contract.Binary != nil {
					w.Header().Set("Content-Type", "image/png")
					w.Header().Set("ETag", `"contract"`)
					_, _ = w.Write(contract.Binary)
					return
				}
				if contract.OAuth {
					_, _ = io.WriteString(w, `{"access_token":"access","token_type":"Bearer","expire_in":3600,"scope":"get_user_info"}`)
					return
				}
				data := contract.Data
				if data == "" {
					data = `{}`
				}
				_, _ = fmt.Fprintf(w, `{"code":0,"message":"ok","data":%s}`, data)
			}))
			defer server.Close()

			client := NewClient("contract-token", WithBaseURL(server.URL+"/api"), WithoutRateLimit(), WithoutRetry())
			defer func() { _ = client.Close() }()
			require.NoError(t, contract.Invoke(context.Background(), client, server.URL))
		})
	}
}

func queryValues(values map[string]string) url.Values {
	result := make(url.Values, len(values))
	for key, value := range values {
		result.Set(key, value)
	}
	return result
}

func contractListData() string {
	return `{"items":[],"meta":{},"sort":{}}`
}

func officialServiceContracts() []serviceContract {
	var contracts []serviceContract
	contracts = append(contracts, gatewayUserContracts()...)
	contracts = append(contracts, guildContracts()...)
	contracts = append(contracts, channelContracts()...)
	contracts = append(contracts, messageContracts()...)
	contracts = append(contracts, roleEmojiContracts()...)
	contracts = append(contracts, utilityContracts()...)
	contracts = append(contracts, threadVoiceTemplateOAuthContracts()...)
	return contracts
}

func contractName(endpoint string) string {
	return strings.ReplaceAll(endpoint, "/", "-")
}

func gatewayUserContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("gateway/index"), Method: http.MethodGet, Endpoint: "gateway/index",
			Query: map[string]string{"compress": "1"}, Data: `{"url":"wss://gateway.test"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Gateway.GetGateway(ctx, GatewayParams{Compress: testPtr(1)})
				return err
			},
		},
		{
			Name: contractName("user/me"), Method: http.MethodGet, Endpoint: "user/me", Data: `{"id":"u"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.User.GetMe(ctx)
				return err
			},
		},
		{
			Name: contractName("user/view"), Method: http.MethodGet, Endpoint: "user/view",
			Query: map[string]string{"user_id": "u", "guild_id": "g"}, Data: `{"id":"u"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.User.GetUser(ctx, UserViewParams{UserID: "u", GuildID: "g"})
				return err
			},
		},
		{
			Name: contractName("user/offline"), Method: http.MethodPost, Endpoint: "user/offline",
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.User.SetOffline(ctx)
			},
		},
		{
			Name: contractName("user/online"), Method: http.MethodPost, Endpoint: "user/online",
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.User.SetOnline(ctx)
			},
		},
		{
			Name: contractName("user/get-online-status"), Method: http.MethodGet, Endpoint: "user/get-online-status", Data: `{"online":true,"online_os":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.User.GetOnlineStatus(ctx)
				return err
			},
		},
	}
}

func guildContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("guild/list"), Method: http.MethodGet, Endpoint: "guild/list",
			Query: map[string]string{"page": "1", "page_size": "20", "sort": "-id"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Guild.GetGuildList(ctx, GuildListParams{Page: testPtr(1), PageSize: testPtr(20), Sort: "-id"})
				return err
			},
		},
		{
			Name: contractName("guild/view"), Method: http.MethodGet, Endpoint: "guild/view",
			Query: map[string]string{"guild_id": "g"}, Data: `{"id":"g"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Guild.GetGuildInfo(ctx, GuildViewParams{GuildID: "g"})
				return err
			},
		},
		{
			Name: contractName("guild/user-list"), Method: http.MethodGet, Endpoint: "guild/user-list",
			Query: map[string]string{
				"guild_id": "g", "channel_id": "c", "search": "name", "role_id": "1", "mobile_verified": "0",
				"active_time": "1", "joined_at": "0", "page": "1", "page_size": "20", "filter_user_id": "u",
			},
			Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Guild.GetGuildMembers(ctx, GuildMembersParams{
					GuildID: "g", ChannelID: "c", Search: "name", RoleID: testPtr(1), MobileVerified: testPtr(0),
					ActiveTime: testPtr(1), JoinedAt: testPtr(0), Page: testPtr(1), PageSize: testPtr(20), FilterUserID: "u",
				})
				return err
			},
		},
		{
			Name: contractName("guild/nickname"), Method: http.MethodPost, Endpoint: "guild/nickname",
			Body: map[string]any{"guild_id": "g", "user_id": "u", "nickname": ""},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Guild.UpdateGuildMemberNickname(ctx, GuildNicknameParams{GuildID: "g", UserID: "u", Nickname: testPtr("")})
			},
		},
		{
			Name: contractName("guild/leave"), Method: http.MethodPost, Endpoint: "guild/leave",
			Body: map[string]any{"guild_id": "g"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Guild.LeaveGuild(ctx, GuildLeaveParams{GuildID: "g"})
			},
		},
		{
			Name: contractName("guild/kickout"), Method: http.MethodPost, Endpoint: "guild/kickout",
			Body: map[string]any{"guild_id": "g", "target_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Guild.KickGuildMember(ctx, GuildKickoutParams{GuildID: "g", TargetID: "u"})
			},
		},
		{
			Name: contractName("guild-mute/list"), Method: http.MethodGet, Endpoint: "guild-mute/list",
			Query: map[string]string{"guild_id": "g", "return_type": "detail"}, Data: `{"mic":{"type":1,"user_ids":[]},"headset":{"type":2,"user_ids":[]}}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Guild.GetGuildMuteList(ctx, GuildMuteListParams{GuildID: "g", ReturnType: "detail"})
				return err
			},
		},
		{
			Name: contractName("guild-mute/create"), Method: http.MethodPost, Endpoint: "guild-mute/create",
			Body: map[string]any{"guild_id": "g", "user_id": "u", "type": 1},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Guild.CreateGuildMute(ctx, GuildMuteCreateParams{GuildID: "g", UserID: "u", Type: GuildMuteTypeMic})
			},
		},
		{
			Name: contractName("guild-mute/delete"), Method: http.MethodPost, Endpoint: "guild-mute/delete",
			Body: map[string]any{"guild_id": "g", "user_id": "u", "type": 2},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Guild.DeleteGuildMute(ctx, GuildMuteDeleteParams{GuildID: "g", UserID: "u", Type: GuildMuteTypeHeadset})
			},
		},
		{
			Name: contractName("guild-boost/history"), Method: http.MethodGet, Endpoint: "guild-boost/history",
			Query: map[string]string{"guild_id": "g", "start_time": "0", "end_time": "10"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Guild.GetGuildBoostHistory(ctx, GuildBoostHistoryParams{GuildID: "g", StartTime: testPtr(int64(0)), EndTime: testPtr(int64(10))})
				return err
			},
		},
	}
}

func channelContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("channel/list"), Method: http.MethodGet, Endpoint: "channel/list",
			Query: map[string]string{"guild_id": "g", "page": "1", "page_size": "20", "type": "2", "parent_id": "p"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.GetChannelList(ctx, ChannelListParams{
					GuildID: "g", Page: testPtr(1), PageSize: testPtr(20), Type: testPtr(2), ParentID: "p",
				})
				return err
			},
		},
		{
			Name: contractName("channel/view"), Method: http.MethodGet, Endpoint: "channel/view",
			Query: map[string]string{"target_id": "c", "need_children": "true"}, Data: `{"id":"c","children":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.View(ctx, ChannelViewParams{TargetID: "c", NeedChildren: testPtr(true)})
				return err
			},
		},
		{
			Name: contractName("channel/create"), Method: http.MethodPost, Endpoint: "channel/create",
			Body: map[string]any{
				"guild_id": "g", "name": "voice", "type": 2, "parent_id": "p", "limit_amount": 0,
				"voice_quality": "2", "is_category": 0,
			},
			Data: `{"id":"c"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.CreateChannel(ctx, CreateChannelParams{
					GuildID: "g", Name: "voice", Type: testPtr(2), ParentID: "p", LimitAmount: testPtr(0),
					VoiceQuality: testPtr("2"), IsCategory: testPtr(false),
				})
				return err
			},
		},
		{
			Name: contractName("channel/update"), Method: http.MethodPost, Endpoint: "channel/update",
			Body: map[string]any{
				"channel_id": "c", "name": "", "level": 0, "parent_id": "", "topic": "", "slow_mode": 0,
				"limit_amount": 0, "voice_quality": "", "password": "",
			},
			Data: `{"id":"c"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.UpdateChannel(ctx, UpdateChannelParams{
					ChannelID: "c", Name: testPtr(""), Level: testPtr(0), ParentID: testPtr(""), Topic: testPtr(""),
					SlowMode: testPtr(0), LimitAmount: testPtr(0), VoiceQuality: testPtr(""), Password: testPtr(""),
				})
				return err
			},
		},
		{
			Name: contractName("channel/delete"), Method: http.MethodPost, Endpoint: "channel/delete",
			Body: map[string]any{"channel_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Channel.DeleteChannel(ctx, ChannelDeleteParams{ChannelID: "c"})
			},
		},
		{
			Name: contractName("channel/user-list"), Method: http.MethodGet, Endpoint: "channel/user-list",
			Query: map[string]string{"channel_id": "c"}, Data: `[]`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.GetChannelUserList(ctx, ChannelUserListParams{ChannelID: "c"})
				return err
			},
		},
		{
			Name: contractName("channel/move-user"), Method: http.MethodPost, Endpoint: "channel/move-user",
			Body: map[string]any{"target_id": "c", "user_ids": []string{"u1", "u2"}},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Channel.MoveUsers(ctx, ChannelMoveUserParams{TargetID: "c", UserIDs: []string{"u1", "u2"}})
			},
		},
		{
			Name: contractName("channel/kickout"), Method: http.MethodPost, Endpoint: "channel/kickout",
			Body: map[string]any{"channel_id": "c", "user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Channel.KickoutUser(ctx, ChannelKickoutParams{ChannelID: "c", UserID: "u"})
			},
		},
		{
			Name: contractName("channel-role/index"), Method: http.MethodGet, Endpoint: "channel-role/index",
			Query: map[string]string{"channel_id": "c"}, Data: `{"permission_overwrites":[],"permission_users":[],"permission_sync":0}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.GetChannelRole(ctx, ChannelRoleViewParams{ChannelID: "c"})
				return err
			},
		},
		{
			Name: contractName("channel-role/create"), Method: http.MethodPost, Endpoint: "channel-role/create",
			Body: map[string]any{"channel_id": "c", "type": "role_id", "value": "1"}, Data: `{"role_id":"1","allow":0,"deny":0}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.CreateChannelRole(ctx, CreateChannelRoleParams{ChannelID: "c", Type: "role_id", Value: "1"})
				return err
			},
		},
		{
			Name: contractName("channel-role/update"), Method: http.MethodPost, Endpoint: "channel-role/update",
			Body: map[string]any{"channel_id": "c", "type": "role_id", "value": "1", "allow": 0, "deny": 0},
			Data: `{"role_id":1,"allow":0,"deny":0}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.UpdateChannelRole(ctx, UpdateChannelRoleParams{
					ChannelID: "c", Type: "role_id", Value: "1", Allow: testPtr(0), Deny: testPtr(0),
				})
				return err
			},
		},
		{
			Name: contractName("channel-role/sync"), Method: http.MethodPost, Endpoint: "channel-role/sync",
			Body: map[string]any{"channel_id": "c"}, Data: `{"permission_overwrites":[],"permission_users":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Channel.SyncChannelRole(ctx, ChannelRoleSyncParams{ChannelID: "c"})
				return err
			},
		},
		{
			Name: contractName("channel-role/delete"), Method: http.MethodPost, Endpoint: "channel-role/delete",
			Body: map[string]any{"channel_id": "c", "type": "role_id", "value": "1"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Channel.DeleteChannelRole(ctx, DeleteChannelRoleParams{ChannelID: "c", Type: "role_id", Value: "1"})
			},
		},
		{
			Name: contractName("channel-user/get-joined-channel"), Method: http.MethodGet, Endpoint: "channel-user/get-joined-channel",
			Query: map[string]string{"guild_id": "g", "user_id": "u", "page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.ChannelUser.GetJoinedChannels(ctx, JoinedChannelParams{
					GuildID: "g", UserID: "u", Page: testPtr(1), PageSize: testPtr(20),
				})
				return err
			},
		},
	}
}

func messageContracts() []serviceContract {
	messageType := MessageTypeText
	return []serviceContract{
		{
			Name: contractName("message/list"), Method: http.MethodGet, Endpoint: "message/list",
			Query: map[string]string{"target_id": "c", "msg_id": "m", "pin": "0", "flag": "around", "page_size": "20"}, Data: `{"items":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Message.GetMessageList(ctx, MessageListParams{
					TargetID: "c", MsgID: "m", Pin: testPtr(0), Flag: "around", PageSize: testPtr(20),
				})
				return err
			},
		},
		{
			Name: contractName("message/view"), Method: http.MethodGet, Endpoint: "message/view",
			Query: map[string]string{"msg_id": "m"}, Data: `{"id":"m"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Message.GetMessage(ctx, MessageViewParams{MsgID: "m"})
				return err
			},
		},
		{
			Name: contractName("message/create"), Method: http.MethodPost, Endpoint: "message/create",
			Body: map[string]any{
				"target_id": "c", "content": "hello", "type": 1, "quote": "q", "nonce": "n",
				"temp_target_id": "u", "template_id": "tpl", "reply_msg_id": "r",
			},
			Data: `{"msg_id":"m","msg_timestamp":1,"nonce":"n"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Message.Create(ctx, MessageCreateParams{
					TargetID: "c", Content: "hello", Type: &messageType, Quote: "q", Nonce: "n",
					TempTargetID: "u", TemplateID: "tpl", ReplyMsgID: "r",
				})
				return err
			},
		},
		{
			Name: contractName("message/update"), Method: http.MethodPost, Endpoint: "message/update",
			Body: map[string]any{
				"msg_id": "m", "content": "updated", "quote": "", "temp_target_id": "",
				"template_id": "", "reply_msg_id": "",
			},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.Update(ctx, UpdateMessageParams{
					MsgID: "m", Content: "updated", Quote: testPtr(""), TempTargetID: testPtr(""),
					TemplateID: testPtr(""), ReplyMsgID: testPtr(""),
				})
			},
		},
		{
			Name: contractName("message/delete"), Method: http.MethodPost, Endpoint: "message/delete",
			Body: map[string]any{"msg_id": "m"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.DeleteMessage(ctx, MessageDeleteParams{MsgID: "m"})
			},
		},
		{
			Name: contractName("message/reaction-list"), Method: http.MethodGet, Endpoint: "message/reaction-list",
			Query: map[string]string{"msg_id": "m", "emoji": "e"}, Data: `[]`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Message.ReactionUsers(ctx, MessageReactionUsersParams{MsgID: "m", Emoji: "e"})
				return err
			},
		},
		{
			Name: contractName("message/add-reaction"), Method: http.MethodPost, Endpoint: "message/add-reaction",
			Body: map[string]any{"msg_id": "m", "emoji": "e"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.AddReaction(ctx, MessageReactionParams{MsgID: "m", Emoji: "e"})
			},
		},
		{
			Name: contractName("message/delete-reaction"), Method: http.MethodPost, Endpoint: "message/delete-reaction",
			Body: map[string]any{"msg_id": "m", "emoji": "e", "user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.DeleteReaction(ctx, MessageDeleteReactionParams{MsgID: "m", Emoji: "e", UserID: "u"})
			},
		},
		{
			Name: contractName("message/send-pipemsg"), Method: http.MethodPost, Endpoint: "message/send-pipemsg",
			Query: map[string]string{"access_token": "pipe", "target_id": "c", "type": "1"},
			Body:  map[string]any{"content": "hello"}, Data: `{"msg_id":"m"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Message.SendPipe(ctx, SendPipeMessageParams{
					AccessToken: "pipe", TargetID: "c", Type: &messageType, Content: "hello",
				})
				return err
			},
		},
		{
			Name: contractName("message/pin"), Method: http.MethodPost, Endpoint: "message/pin",
			Body: map[string]any{"msg_id": "m", "target_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.PinMessage(ctx, MessagePinParams{MsgID: "m", TargetID: "c"})
			},
		},
		{
			Name: contractName("message/unpin"), Method: http.MethodPost, Endpoint: "message/unpin",
			Body: map[string]any{"msg_id": "m", "target_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Message.UnpinMessage(ctx, MessageUnpinParams{MsgID: "m", TargetID: "c"})
			},
		},
		{
			Name: contractName("user-chat/list"), Method: http.MethodGet, Endpoint: "user-chat/list",
			Query: map[string]string{"page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.UserChat.GetUserChatList(ctx, UserChatListParams{Page: testPtr(1), PageSize: testPtr(20)})
				return err
			},
		},
		{
			Name: contractName("user-chat/view"), Method: http.MethodGet, Endpoint: "user-chat/view",
			Query: map[string]string{"chat_code": "chat"}, Data: `{"code":"chat"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.UserChat.GetUserChat(ctx, UserChatViewParams{ChatCode: "chat"})
				return err
			},
		},
		{
			Name: contractName("user-chat/create"), Method: http.MethodPost, Endpoint: "user-chat/create",
			Body: map[string]any{"target_id": "u"}, Data: `{"code":"chat"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.UserChat.CreateUserChat(ctx, UserChatCreateParams{TargetID: "u"})
				return err
			},
		},
		{
			Name: contractName("user-chat/delete"), Method: http.MethodPost, Endpoint: "user-chat/delete",
			Body: map[string]any{"chat_code": "chat"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.UserChat.DeleteUserChat(ctx, UserChatDeleteParams{ChatCode: "chat"})
			},
		},
		{
			Name: contractName("direct-message/list"), Method: http.MethodGet, Endpoint: "direct-message/list",
			Query: map[string]string{
				"chat_code": "chat", "target_id": "u", "msg_id": "m", "flag": "before", "page": "1", "page_size": "20",
			},
			Data: `{"items":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.DirectMessage.List(ctx, DirectMessageListParams{
					ChatCode: "chat", TargetID: "u", MsgID: "m", Flag: "before", Page: testPtr(1), PageSize: testPtr(20),
				})
				return err
			},
		},
		{
			Name: contractName("direct-message/view"), Method: http.MethodGet, Endpoint: "direct-message/view",
			Query: map[string]string{"chat_code": "chat", "msg_id": "m"}, Data: `{"id":"m"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.DirectMessage.View(ctx, DirectMessageViewParams{ChatCode: "chat", MsgID: "m"})
				return err
			},
		},
		{
			Name: contractName("direct-message/create"), Method: http.MethodPost, Endpoint: "direct-message/create",
			Body: map[string]any{
				"target_id": "u", "chat_code": "chat", "content": "hello", "type": 1,
				"quote": "q", "nonce": "n", "template_id": "tpl", "reply_msg_id": "r",
			},
			Data: `{"msg_id":"m"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.DirectMessage.Create(ctx, DirectMessageCreateParams{
					TargetID: "u", ChatCode: "chat", Content: "hello", Type: &messageType,
					Quote: "q", Nonce: "n", TemplateID: "tpl", ReplyMsgID: "r",
				})
				return err
			},
		},
		{
			Name: contractName("direct-message/update"), Method: http.MethodPost, Endpoint: "direct-message/update",
			Body: map[string]any{"msg_id": "m", "content": "updated", "quote": "", "template_id": "", "reply_msg_id": ""},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.DirectMessage.Update(ctx, DirectMessageUpdateParams{
					MsgID: "m", Content: "updated", Quote: testPtr(""), TemplateID: testPtr(""), ReplyMsgID: testPtr(""),
				})
			},
		},
		{
			Name: contractName("direct-message/delete"), Method: http.MethodPost, Endpoint: "direct-message/delete",
			Body: map[string]any{"msg_id": "m"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.DirectMessage.Delete(ctx, DirectMessageDeleteParams{MsgID: "m"})
			},
		},
		{
			Name: contractName("direct-message/reaction-list"), Method: http.MethodGet, Endpoint: "direct-message/reaction-list",
			Query: map[string]string{"msg_id": "m", "emoji": "e"}, Data: `[]`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.DirectMessage.ReactionUsers(ctx, DirectMessageReactionUsersParams{MsgID: "m", Emoji: "e"})
				return err
			},
		},
		{
			Name: contractName("direct-message/add-reaction"), Method: http.MethodPost, Endpoint: "direct-message/add-reaction",
			Body: map[string]any{"msg_id": "m", "emoji": "e"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.DirectMessage.AddReaction(ctx, DirectMessageReactionParams{MsgID: "m", Emoji: "e"})
			},
		},
		{
			Name: contractName("direct-message/delete-reaction"), Method: http.MethodPost, Endpoint: "direct-message/delete-reaction",
			Body: map[string]any{"msg_id": "m", "emoji": "e", "user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.DirectMessage.DeleteReaction(ctx, DirectMessageDeleteReactionParams{MsgID: "m", Emoji: "e", UserID: "u"})
			},
		},
	}
}

func roleEmojiContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("guild-role/list"), Method: http.MethodGet, Endpoint: "guild-role/list",
			Query: map[string]string{"guild_id": "g", "page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildRole.GetRoleList(ctx, RoleListParams{GuildID: "g", Page: testPtr(1), PageSize: testPtr(20)})
				return err
			},
		},
		{
			Name: contractName("guild-role/create"), Method: http.MethodPost, Endpoint: "guild-role/create",
			Body: map[string]any{"guild_id": "g", "name": "role"}, Data: `{"role_id":1,"name":"role"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildRole.CreateRole(ctx, CreateRoleParams{GuildID: "g", Name: testPtr("role")})
				return err
			},
		},
		{
			Name: contractName("guild-role/update"), Method: http.MethodPost, Endpoint: "guild-role/update",
			Body: map[string]any{
				"guild_id": "g", "role_id": 1, "name": "", "color": 0, "hoist": 0, "mentionable": 0, "permissions": 0,
			},
			Data: `{"role_id":1}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildRole.UpdateRole(ctx, UpdateRoleParams{
					GuildID: "g", RoleID: 1, Name: testPtr(""), Color: testPtr(0), Hoist: testPtr(0),
					Mentionable: testPtr(0), Permissions: testPtr(0),
				})
				return err
			},
		},
		{
			Name: contractName("guild-role/delete"), Method: http.MethodPost, Endpoint: "guild-role/delete",
			Body: map[string]any{"guild_id": "g", "role_id": 1},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.GuildRole.DeleteRole(ctx, DeleteRoleParams{GuildID: "g", RoleID: 1})
			},
		},
		{
			Name: contractName("guild-role/grant"), Method: http.MethodPost, Endpoint: "guild-role/grant",
			Body: map[string]any{"guild_id": "g", "user_id": "u", "role_id": 1}, Data: `{"user_id":"u","guild_id":"g","roles":[1]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildRole.GrantRole(ctx, GrantRoleParams{GuildID: "g", UserID: "u", RoleID: 1})
				return err
			},
		},
		{
			Name: contractName("guild-role/revoke"), Method: http.MethodPost, Endpoint: "guild-role/revoke",
			Body: map[string]any{"guild_id": "g", "user_id": "u", "role_id": 1}, Data: `{"user_id":"u","guild_id":"g","roles":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildRole.RevokeRole(ctx, RevokeRoleParams{GuildID: "g", UserID: "u", RoleID: 1})
				return err
			},
		},
		{
			Name: contractName("guild-emoji/list"), Method: http.MethodGet, Endpoint: "guild-emoji/list",
			Query: map[string]string{"guild_id": "g", "page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildEmoji.GetEmojiList(ctx, EmojiListParams{GuildID: "g", Page: testPtr(1), PageSize: testPtr(20)})
				return err
			},
		},
		{
			Name: contractName("guild-emoji/create"), Method: http.MethodPost, Endpoint: "guild-emoji/create",
			MultipartFields: map[string]string{"guild_id": "g", "name": "emoji"},
			MultipartFiles:  map[string][]byte{"emoji": {0x89, 'P', 'N', 'G'}}, MultipartNames: map[string]string{"emoji": "emoji.png"},
			Data: `{"id":"e","name":"emoji","user_info":{"id":"u"}}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.GuildEmoji.CreateEmoji(ctx, EmojiCreateParams{
					GuildID: "g", Name: "emoji", FileName: "emoji.png", Emoji: []byte{0x89, 'P', 'N', 'G'},
				})
				return err
			},
		},
		{
			Name: contractName("guild-emoji/update"), Method: http.MethodPost, Endpoint: "guild-emoji/update",
			Body: map[string]any{"id": "e", "name": "renamed"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.GuildEmoji.UpdateEmoji(ctx, EmojiUpdateParams{ID: "e", Name: "renamed"})
			},
		},
		{
			Name: contractName("guild-emoji/delete"), Method: http.MethodPost, Endpoint: "guild-emoji/delete",
			Body: map[string]any{"id": "e"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.GuildEmoji.DeleteEmoji(ctx, EmojiDeleteParams{ID: "e"})
			},
		},
	}
}

func utilityContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("blacklist/list"), Method: http.MethodGet, Endpoint: "blacklist/list",
			Query: map[string]string{"guild_id": "g", "page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Blacklist.GetBlacklistUsers(ctx, BlacklistListParams{GuildID: "g", Page: testPtr(1), PageSize: testPtr(20)})
				return err
			},
		},
		{
			Name: contractName("blacklist/create"), Method: http.MethodPost, Endpoint: "blacklist/create",
			Body: map[string]any{"guild_id": "g", "target_id": "u", "remark": "reason", "del_msg_days": 0},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Blacklist.CreateBlacklistUser(ctx, BlacklistCreateParams{
					GuildID: "g", TargetID: "u", Remark: "reason", DelMsgDays: testPtr(0),
				})
			},
		},
		{
			Name: contractName("blacklist/delete"), Method: http.MethodPost, Endpoint: "blacklist/delete",
			Body: map[string]any{"guild_id": "g", "target_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Blacklist.DeleteBlacklistUser(ctx, BlacklistDeleteParams{GuildID: "g", TargetID: "u"})
			},
		},
		{
			Name: contractName("invite/list"), Method: http.MethodGet, Endpoint: "invite/list",
			Query: map[string]string{"guild_id": "g", "channel_id": "c", "page": "1", "page_size": "20"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Invite.GetInviteList(ctx, InviteListParams{
					GuildID: "g", ChannelID: "c", Page: testPtr(1), PageSize: testPtr(20),
				})
				return err
			},
		},
		{
			Name: contractName("invite/invitees"), Method: http.MethodGet, Endpoint: "invite/invitees",
			Query: map[string]string{
				"id": "code", "invite_url": "https://kook.vip/code", "guild_id": "g", "status": "-1",
				"start_time": "2026-06-01 12:00:00", "end_time": "2026-07-01 12:00:00", "page": "1", "page_size": "20",
			},
			Data: `{"items":[{"status":0,"joined_time":1773643290000,"active_time":1773643289899,"show_name":"user#0001"}],"meta":{"page":1,"page_total":1,"page_size":20,"total":1},"sort":{},"count":1,"keep_count":1,"loss_count":0}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Invite.GetInvitees(ctx, InviteeListParams{
					ID: "code", InviteURL: "https://kook.vip/code", GuildID: "g", Status: testPtr(InviteeStatusAll),
					StartTime: "2026-06-01 12:00:00", EndTime: "2026-07-01 12:00:00", Page: 1, PageSize: 20,
				})
				return err
			},
		},
		{
			Name: contractName("invite/create"), Method: http.MethodPost, Endpoint: "invite/create",
			Body: map[string]any{"guild_id": "g", "channel_id": "c", "duration": 0, "setting_times": -1}, Data: `{"url":"https://kook.vip/x"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Invite.CreateInvite(ctx, CreateInviteParams{
					GuildID: "g", ChannelID: "c", Duration: testPtr(0), SettingTimes: testPtr(-1),
				})
				return err
			},
		},
		{
			Name: contractName("invite/delete"), Method: http.MethodPost, Endpoint: "invite/delete",
			Body: map[string]any{"url_code": "x", "guild_id": "g", "channel_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Invite.DeleteInvite(ctx, DeleteInviteParams{URLCode: "x", GuildID: "g", ChannelID: "c"})
			},
		},
		{
			Name: contractName("asset/create"), Method: http.MethodPost, Endpoint: "asset/create",
			MultipartFiles: map[string][]byte{"file": {'f', 'i', 'l', 'e'}}, MultipartNames: map[string]string{"file": "file.txt"},
			Data: `{"url":"https://asset.test/file"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Asset.Create(ctx, AssetCreateParams{FileName: "file.txt", Content: []byte("file")})
				return err
			},
		},
		{
			Name: contractName("intimacy/index"), Method: http.MethodGet, Endpoint: "intimacy/index",
			Query: map[string]string{"user_id": "u"}, Data: `{"user_id":"u","score":0}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Intimacy.GetIntimacy(ctx, IntimacyViewParams{UserID: "u"})
				return err
			},
		},
		{
			Name: contractName("intimacy/update"), Method: http.MethodPost, Endpoint: "intimacy/update",
			Body: map[string]any{"user_id": "u", "score": 0, "social_info": "", "img_id": ""},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Intimacy.UpdateIntimacy(ctx, UpdateIntimacyParams{
					UserID: "u", Score: testPtr(0), SocialInfo: testPtr(""), ImgID: testPtr(""),
				})
			},
		},
		{
			Name: contractName("friend"), Method: http.MethodGet, Endpoint: "friend",
			Query: map[string]string{"type": "friend"}, Data: `{"friend":[],"request":[],"block":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Friend.GetFriendsList(ctx, FriendListParams{Type: "friend"})
				return err
			},
		},
		{
			Name: contractName("friend/request"), Method: http.MethodPost, Endpoint: "friend/request",
			Body: map[string]any{"user_code": "name#0001", "from": 2, "guild_id": "g"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Friend.SendFriendRequest(ctx, SendFriendRequestParams{
					UserCode: "name#0001", From: FriendRequestFromGuild, GuildID: "g",
				})
			},
		},
		{
			Name: contractName("friend/handle-request"), Method: http.MethodPost, Endpoint: "friend/handle-request",
			Body: map[string]any{"id": 1, "accept": false}, Data: `true`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Friend.HandleFriendRequest(ctx, HandleFriendRequestParams{ID: 1, Accept: false})
				return err
			},
		},
		{
			Name: contractName("friend/delete"), Method: http.MethodPost, Endpoint: "friend/delete",
			Body: map[string]any{"user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Friend.DeleteFriend(ctx, DeleteFriendParams{UserID: "u"})
			},
		},
		{
			Name: contractName("friend/block"), Method: http.MethodPost, Endpoint: "friend/block",
			Body: map[string]any{"user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Friend.BlockFriend(ctx, BlockFriendParams{UserID: "u"})
			},
		},
		{
			Name: contractName("friend/unblock"), Method: http.MethodPost, Endpoint: "friend/unblock",
			Body: map[string]any{"user_id": "u"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Friend.UnblockFriend(ctx, UnblockFriendParams{UserID: "u"})
			},
		},
		{
			Name: contractName("game"), Method: http.MethodGet, Endpoint: "game",
			Query: map[string]string{"type": "0"}, Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Game.GetGameList(ctx, GameListParams{Type: GameTypeAll})
				return err
			},
		},
		{
			Name: contractName("game/create"), Method: http.MethodPost, Endpoint: "game/create",
			Body: map[string]any{"name": "game", "icon": "https://icon.test"}, Data: `{"id":1,"name":"game"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Game.CreateGame(ctx, GameCreateParams{Name: "game", Icon: "https://icon.test"})
				return err
			},
		},
		{
			Name: contractName("game/update"), Method: http.MethodPost, Endpoint: "game/update",
			Body: map[string]any{"id": 1, "name": "", "icon": ""}, Data: `{"id":1}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Game.UpdateGame(ctx, GameUpdateParams{ID: 1, Name: testPtr(""), Icon: testPtr("")})
				return err
			},
		},
		{
			Name: contractName("game/delete"), Method: http.MethodPost, Endpoint: "game/delete",
			Body: map[string]any{"id": 1},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Game.DeleteGame(ctx, GameDeleteParams{ID: 1})
			},
		},
		{
			Name: contractName("game/activity"), Method: http.MethodPost, Endpoint: "game/activity",
			Body: map[string]any{"id": 1, "data_type": 2, "software": "cloudmusic", "singer": "singer", "music_name": "song"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Game.AddActivity(ctx, GameActivityParams{
					ID: testPtr(1), DataType: GameActivityTypeMusic, Software: SoftwareCloudMusic, Singer: "singer", MusicName: "song",
				})
			},
		},
		{
			Name: contractName("game/delete-activity"), Method: http.MethodPost, Endpoint: "game/delete-activity",
			Body: map[string]any{"data_type": 1},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Game.DeleteActivity(ctx, GameDeleteActivityParams{DataType: GameActivityTypeGame})
			},
		},
		{
			Name: contractName("badge/guild"), Method: http.MethodGet, Endpoint: "badge/guild",
			Query: map[string]string{"guild_id": "g", "style": "2"}, Binary: []byte{0x89, 'P', 'N', 'G'},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Badge.GetGuildBadge(ctx, BadgeParams{GuildID: "g", Style: BadgeStyleOnlineAndTotal})
				return err
			},
		},
	}
}

func threadVoiceTemplateOAuthContracts() []serviceContract {
	return []serviceContract{
		{
			Name: contractName("category/list"), Method: http.MethodGet, Endpoint: "category/list",
			Query: map[string]string{"channel_id": "c"}, Data: `{"list":[]}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.GetThreadCategories(ctx, ThreadCategoryListParams{ChannelID: "c"})
				return err
			},
		},
		{
			Name: contractName("thread/create"), Method: http.MethodPost, Endpoint: "thread/create",
			Body: map[string]any{
				"channel_id": "c", "guild_id": "g", "category_id": "category", "title": "title",
				"cover": "https://cover.test", "content": "card",
			},
			Data: `{"id":"thread","post_id":"post"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.CreateThread(ctx, CreateThreadParams{
					ChannelID: "c", GuildID: "g", CategoryID: "category", Title: "title",
					Cover: "https://cover.test", Content: "card",
				})
				return err
			},
		},
		{
			Name: contractName("thread/reply"), Method: http.MethodPost, Endpoint: "thread/reply",
			Body: map[string]any{"channel_id": "c", "thread_id": "thread", "reply_id": "post", "content": "reply"},
			Data: `{"id":"reply","thread_id":"thread"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.ReplyThread(ctx, ReplyThreadParams{
					ChannelID: "c", ThreadID: "thread", ReplyID: "post", Content: "reply",
				})
				return err
			},
		},
		{
			Name: contractName("thread/view"), Method: http.MethodGet, Endpoint: "thread/view",
			Query: map[string]string{"channel_id": "c", "thread_id": "thread"}, Data: `{"id":"thread"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.GetThread(ctx, ThreadViewParams{ChannelID: "c", ThreadID: "thread"})
				return err
			},
		},
		{
			Name: contractName("thread/list"), Method: http.MethodGet, Endpoint: "thread/list",
			Query: map[string]string{"channel_id": "c", "category_id": "category", "sort": "1", "time": "0", "page_size": "20"},
			Data:  contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.GetThreadList(ctx, GetThreadListParams{
					ChannelID: "c", CategoryID: "category", Sort: testPtr(1), Time: testPtr(int64(0)), PageSize: testPtr(20),
				})
				return err
			},
		},
		{
			Name: contractName("thread/delete"), Method: http.MethodPost, Endpoint: "thread/delete",
			Body: map[string]any{"channel_id": "c", "thread_id": "thread", "post_id": "post"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Thread.DeleteThread(ctx, ThreadDeleteParams{ChannelID: "c", ThreadID: "thread", PostID: "post"})
			},
		},
		{
			Name: contractName("thread/post"), Method: http.MethodGet, Endpoint: "thread/post",
			Query: map[string]string{
				"channel_id": "c", "thread_id": "thread", "post_id": "post", "time": "0",
				"page_size": "20", "order": "asc", "page": "1",
			},
			Data: `{"items":[],"meta":{}}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Thread.GetThreadPosts(ctx, GetThreadPostsParams{
					ChannelID: "c", ThreadID: "thread", PostID: "post", Time: "0", PageSize: testPtr(20), Order: "asc", Page: 1,
				})
				return err
			},
		},
		{
			Name: contractName("voice/join"), Method: http.MethodPost, Endpoint: "voice/join",
			Body: map[string]any{
				"channel_id": "c", "audio_ssrc": "1111", "audio_pt": "111", "rtcp_mux": false, "password": "password",
			},
			Data: `{"ip":"127.0.0.1","port":1000,"rtcp_mux":false,"rtcp_port":"1001","bitrate":64000,"audio_ssrc":"1111","audio_pt":"111"}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Voice.JoinVoiceChannel(ctx, VoiceJoinParams{
					ChannelID: "c", AudioSSRC: "1111", AudioPT: "111", RTCPMux: testPtr(false), Password: "password",
				})
				return err
			},
		},
		{
			Name: contractName("voice/list"), Method: http.MethodGet, Endpoint: "voice/list", Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Voice.GetJoinedVoiceChannels(ctx)
				return err
			},
		},
		{
			Name: contractName("voice/leave"), Method: http.MethodPost, Endpoint: "voice/leave",
			Body: map[string]any{"channel_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Voice.LeaveVoiceChannel(ctx, VoiceLeaveParams{ChannelID: "c"})
			},
		},
		{
			Name: contractName("voice/keep-alive"), Method: http.MethodPost, Endpoint: "voice/keep-alive",
			Body: map[string]any{"channel_id": "c"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Voice.KeepAliveVoiceChannel(ctx, VoiceKeepAliveParams{ChannelID: "c"})
			},
		},
		{
			Name: contractName("template/list"), Method: http.MethodGet, Endpoint: "template/list", Data: contractListData(),
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Template.GetTemplateList(ctx)
				return err
			},
		},
		{
			Name: contractName("template/create"), Method: http.MethodPost, Endpoint: "template/create",
			Body: map[string]any{
				"title": "title", "content": "content", "type": 0, "msgtype": 1, "test_data": "", "test_channel": "",
			},
			Data: `{"model":{"id":"template"}}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Template.CreateTemplate(ctx, CreateTemplateParams{
					Title: "title", Content: "content", Type: testPtr(0), MsgType: testPtr(1),
					TestData: testPtr(""), TestChannel: testPtr(""),
				})
				return err
			},
		},
		{
			Name: contractName("template/update"), Method: http.MethodPost, Endpoint: "template/update",
			Body: map[string]any{
				"id": "template", "title": "", "content": "", "type": 0, "msgtype": 0, "test_data": "", "test_channel": "",
			},
			Data: `{"model":{"id":"template"}}`,
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				_, err := client.Template.UpdateTemplate(ctx, UpdateTemplateParams{
					ID: "template", Title: testPtr(""), Content: testPtr(""), Type: testPtr(0), MsgType: testPtr(0),
					TestData: testPtr(""), TestChannel: testPtr(""),
				})
				return err
			},
		},
		{
			Name: contractName("template/delete"), Method: http.MethodPost, Endpoint: "template/delete",
			Body: map[string]any{"id": "template"},
			Invoke: func(ctx context.Context, client *Client, _ string) error {
				return client.Template.DeleteTemplate(ctx, DeleteTemplateParams{ID: "template"})
			},
		},
		{
			Name: contractName("oauth2/token"), Method: http.MethodPost, Endpoint: "oauth2/token", OAuth: true,
			Body: map[string]any{
				"grant_type": "authorization_code", "client_id": "client", "client_secret": "secret",
				"code": "code", "redirect_uri": "https://callback.test",
			},
			Invoke: func(ctx context.Context, _ *Client, serverURL string) error {
				_, err := NewOAuthClient(WithOAuthBaseURL(serverURL+"/api")).ExchangeToken(ctx, OAuthTokenParams{
					ClientID: "client", ClientSecret: "secret", Code: "code", RedirectURI: "https://callback.test",
				})
				return err
			},
		},
	}
}
