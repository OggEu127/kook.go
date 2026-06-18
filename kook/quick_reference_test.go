package kook

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestQuickReferenceHTTPAPIsReturnSuccess(t *testing.T) {
	expected := map[string]int{
		"GET /v3/guild/list":                      0,
		"GET /v3/guild/view":                      0,
		"GET /v3/guild/user-list":                 0,
		"POST /v3/guild/nickname":                 0,
		"POST /v3/guild/leave":                    0,
		"POST /v3/guild/kickout":                  0,
		"GET /v3/guild-mute/list":                 0,
		"POST /v3/guild-mute/create":              0,
		"POST /v3/guild-mute/delete":              0,
		"GET /v3/guild-boost/history":             0,
		"GET /v3/channel/list":                    0,
		"GET /v3/channel/view":                    0,
		"POST /v3/channel/create":                 0,
		"POST /v3/channel/update":                 0,
		"POST /v3/channel/delete":                 0,
		"GET /v3/channel/user-list":               0,
		"POST /v3/channel/move-user":              0,
		"POST /v3/channel/kickout":                0,
		"GET /v3/channel-role/index":              0,
		"POST /v3/channel-role/create":            0,
		"POST /v3/channel-role/update":            0,
		"POST /v3/channel-role/sync":              0,
		"POST /v3/channel-role/delete":            0,
		"GET /v3/message/list":                    0,
		"GET /v3/message/view":                    0,
		"POST /v3/message/create":                 0,
		"POST /v3/message/update":                 0,
		"POST /v3/message/delete":                 0,
		"GET /v3/message/reaction-list":           0,
		"POST /v3/message/add-reaction":           0,
		"POST /v3/message/delete-reaction":        0,
		"POST /v3/message/send-pipemsg":           0,
		"GET /v3/channel-user/get-joined-channel": 0,
		"GET /v3/user-chat/list":                  0,
		"GET /v3/user-chat/view":                  0,
		"POST /v3/user-chat/create":               0,
		"POST /v3/user-chat/delete":               0,
		"GET /v3/direct-message/list":             0,
		"GET /v3/direct-message/view":             0,
		"POST /v3/direct-message/create":          0,
		"POST /v3/direct-message/update":          0,
		"POST /v3/direct-message/delete":          0,
		"GET /v3/direct-message/reaction-list":    0,
		"POST /v3/direct-message/add-reaction":    0,
		"POST /v3/direct-message/delete-reaction": 0,
		"GET /v3/friend":                          0,
		"POST /v3/friend/request":                 0,
		"POST /v3/friend/handle-request":          0,
		"POST /v3/friend/delete":                  0,
		"POST /v3/friend/create-relation":         0,
		"POST /v3/friend/handle-relation":         0,
		"POST /v3/friend/unravel-relation":        0,
		"POST /v3/friend/block":                   0,
		"POST /v3/friend/unblock":                 0,
		"GET /v3/gateway/index":                   0,
		"GET /v3/user/me":                         0,
		"GET /v3/user/view":                       0,
		"POST /v3/user/offline":                   0,
		"POST /v3/asset/create":                   0,
		"GET /v3/guild-role/list":                 0,
		"POST /v3/guild-role/create":              0,
		"POST /v3/guild-role/update":              0,
		"POST /v3/guild-role/delete":              0,
		"POST /v3/guild-role/grant":               0,
		"POST /v3/guild-role/revoke":              0,
		"GET /v3/intimacy/index":                  0,
		"POST /v3/intimacy/update":                0,
		"GET /v3/guild-emoji/list":                0,
		"POST /v3/guild-emoji/create":             0,
		"POST /v3/guild-emoji/update":             0,
		"POST /v3/guild-emoji/delete":             0,
		"GET /v3/invite/list":                     0,
		"POST /v3/invite/create":                  0,
		"POST /v3/invite/delete":                  0,
		"GET /v3/blacklist/list":                  0,
		"POST /v3/blacklist/create":               0,
		"POST /v3/blacklist/delete":               0,
		"GET /v3/badge/guild":                     0,
		"GET /v3/game":                            0,
		"POST /v3/game/create":                    0,
		"POST /v3/game/update":                    0,
		"POST /v3/game/delete":                    0,
		"POST /v3/game/activity":                  0,
		"POST /v3/game/delete-activity":           0,
		"GET /v3/category/list":                   0,
		"POST /v3/thread/create":                  0,
		"POST /v3/thread/reply":                   0,
		"GET /v3/thread/view":                     0,
		"GET /v3/thread/list":                     0,
		"POST /v3/thread/delete":                  0,
		"GET /v3/thread/post":                     0,
	}

	client, closeServer := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if _, ok := expected[key]; !ok {
			t.Fatalf("unexpected endpoint: %s", key)
		}
		expected[key]++

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"code":    0,
			"message": "操作成功",
			"data":    quickReferenceResponseData(r.Method, r.URL.Path),
		}); err != nil {
			t.Fatalf("write response: %v", err)
		}
	})
	defer closeServer()

	ctx := context.Background()
	check := func(_ interface{}, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	check(client.Guild.GetGuildList(ctx, 1, 10, ""))
	check(client.Guild.GetGuildInfo(ctx, "guild-1"))
	check(client.Guild.GetGuildMembers(ctx, "guild-1", 1, 10, ""))
	check(nil, client.Guild.UpdateGuildMemberNickname(ctx, "guild-1", "user-1", "nick"))
	check(nil, client.Guild.LeaveGuild(ctx, "guild-1"))
	check(nil, client.Guild.KickGuildMember(ctx, "guild-1", "user-1"))
	check(client.Guild.GetGuildMuteList(ctx, "guild-1", "detail"))
	check(nil, client.Guild.CreateGuildMute(ctx, "guild-1", "user-1", GuildMuteTypeMic))
	check(nil, client.Guild.DeleteGuildMute(ctx, "guild-1", "user-1", GuildMuteTypeMic))
	check(client.Guild.GetGuildBoostHistory(ctx, "guild-1", 1, 2))

	check(client.Channel.GetChannelList(ctx, "guild-1", 1, 10, ""))
	check(client.Channel.GetChannelInfo(ctx, "channel-1"))
	check(client.Channel.CreateChannel(ctx, "guild-1", CreateChannelParams{Name: "text"}))
	check(client.Channel.UpdateChannel(ctx, "channel-1", UpdateChannelParams{Name: "text"}))
	check(nil, client.Channel.DeleteChannel(ctx, "channel-1"))
	check(client.Channel.GetChannelUserList(ctx, "channel-1"))
	check(nil, client.Channel.MoveUsers(ctx, "channel-1", []string{"user-1"}))
	check(nil, client.Channel.KickoutUser(ctx, "channel-1", "user-1"))
	check(client.Channel.GetChannelRole(ctx, "channel-1"))
	check(client.Channel.CreateChannelRole(ctx, ChannelRoleParams{ChannelID: "channel-1", Type: "user_id", Value: "user-1"}))
	check(client.Channel.UpdateChannelRole(ctx, ChannelRoleParams{ChannelID: "channel-1", Type: "user_id", Value: "user-1", Allow: 1}))
	check(client.Channel.SyncChannelRole(ctx, "channel-1"))
	check(nil, client.Channel.DeleteChannelRole(ctx, "channel-1", "user_id", "user-1"))
	check(client.Channel.GetJoinedChannels(ctx, "guild-1", "user-1"))

	check(client.Message.GetMessageList(ctx, "channel-1", GetMessageListParams{}))
	check(client.Message.GetMessage(ctx, "msg-1"))
	check(client.Message.SendMessage(ctx, SendMessageParams{TargetID: "channel-1", Content: "hello"}))
	check(client.Message.UpdateMessage(ctx, "msg-1", "hello", "", ""))
	check(nil, client.Message.DeleteMessage(ctx, "msg-1"))
	check(client.Message.GetReactionUserList(ctx, "msg-1", "😀"))
	check(nil, client.Message.AddReaction(ctx, "msg-1", "😀"))
	check(nil, client.Message.DeleteReaction(ctx, "msg-1", "😀", "user-1"))
	check(client.Message.SendPipeMessage(ctx, SendMessageParams{TargetID: "channel-1", Content: "hello", AccessToken: "pipe-token"}))

	check(client.UserChat.GetUserChatList(ctx, 1, 10))
	check(client.UserChat.GetUserChat(ctx, "chat-1"))
	check(client.UserChat.CreateUserChat(ctx, "user-1"))
	check(nil, client.UserChat.DeleteUserChat(ctx, "chat-1"))
	check(client.Message.GetMessageList(ctx, "", GetMessageListParams{Type: "private", ChatCode: "chat-1"}))
	check(client.Message.GetDirectMessage(ctx, "chat-1", "msg-1"))
	check(client.Message.SendMessage(ctx, SendMessageParams{Type: "private", TargetID: "user-1", Content: "hello"}))
	check(nil, client.Message.UpdateDirectMessage(ctx, "msg-1", "hello", ""))
	check(nil, client.Message.DeleteDirectMessage(ctx, "msg-1"))
	check(client.Message.GetDirectReactionUserList(ctx, "msg-1", "😀"))
	check(nil, client.Message.AddDirectReaction(ctx, "msg-1", "😀"))
	check(nil, client.Message.DeleteDirectReaction(ctx, "msg-1", "😀", "user-1"))

	check(client.Friend.GetFriendsList(ctx))
	check(nil, client.Friend.SendFriendRequest(ctx, SendFriendRequestParams{UserCode: "user#0001"}))
	check(nil, client.Friend.HandleFriendRequest(ctx, "request-1", true))
	check(nil, client.Friend.DeleteFriend(ctx, "user-1"))
	check(nil, client.Friend.CreateRelation(ctx, "user-1"))
	check(nil, client.Friend.HandleRelation(ctx, "request-1", true))
	check(nil, client.Friend.UnravelRelation(ctx, "user-1"))
	check(nil, client.Friend.BlockFriend(ctx, "user-1"))
	check(nil, client.Friend.UnblockFriend(ctx, "user-1"))

	check(client.Gateway.GetGateway(ctx, 0))
	check(client.User.GetMe(ctx))
	check(client.User.GetUser(ctx, "user-1", "guild-1"))
	check(nil, client.User.SetOffline(ctx))
	check(client.Asset.UploadFileContent(ctx, "test.txt", []byte("ok")))
	check(client.Role.GetRoleList(ctx, "guild-1", 1, 10))
	check(client.Role.CreateRole(ctx, "guild-1", "role"))
	check(client.Role.UpdateRole(ctx, "guild-1", 1, UpdateRoleParams{Name: "role"}))
	check(nil, client.Role.DeleteRole(ctx, "guild-1", 1))
	check(client.Role.GrantRole(ctx, "guild-1", "user-1", 1))
	check(client.Role.RevokeRole(ctx, "guild-1", "user-1", 1))

	check(client.Intimacy.GetIntimacy(ctx, "user-1"))
	check(client.Intimacy.UpdateIntimacy(ctx, "user-1", 10, "info", "img"))
	check(client.Emoji.GetEmojiList(ctx, "guild-1", 1, 10))
	check(client.Emoji.CreateEmoji(ctx, "emoji", "guild-1", "asset-id"))
	check(client.Emoji.UpdateEmoji(ctx, "emoji-1", "emoji"))
	check(nil, client.Emoji.DeleteEmoji(ctx, "emoji-1"))
	check(client.Invite.GetInviteList(ctx, "guild-1", 1, 10))
	check(client.Invite.CreateInvite(ctx, CreateInviteParams{GuildID: "guild-1", ChannelID: "channel-1"}))
	check(nil, client.Invite.DeleteInvite(ctx, "invite-code"))
	check(client.Blacklist.GetBlacklistUsers(ctx, "guild-1", 1, 10))
	check(nil, client.Blacklist.CreateBlacklistUser(ctx, "guild-1", "user-1", "", 0))
	check(nil, client.Blacklist.DeleteBlacklistUser(ctx, "guild-1", "user-1"))
	check(client.Badge.GetGuildBadges(ctx, "guild-1"))

	check(client.Game.GetGameList(ctx, ""))
	check(client.Game.CreateGame(ctx, "game", "icon"))
	check(client.Game.UpdateGame(ctx, 1, "game", "icon"))
	check(nil, client.Game.DeleteGame(ctx, 1))
	check(nil, client.Game.AddGameActivity(ctx, 1))
	check(nil, client.Game.DeleteActivity(ctx, 1))
	check(client.Thread.GetThreadCategories(ctx, "channel-1"))
	check(client.Thread.CreateThread(ctx, CreateThreadParams{ChannelID: "channel-1", GuildID: "guild-1", Title: "title", Content: "content"}))
	check(client.Thread.ReplyThread(ctx, ReplyThreadParams{ChannelID: "channel-1", ThreadID: "thread-1", Content: "reply"}))
	check(client.Thread.GetThread(ctx, "channel-1", "thread-1"))
	check(client.Thread.GetThreadList(ctx, GetThreadListParams{ChannelID: "channel-1", Sort: 1}))
	check(nil, client.Thread.DeleteThread(ctx, "channel-1", "thread-1", ""))
	check(client.Thread.GetThreadPost(ctx, "channel-1", "thread-1"))

	for endpoint, count := range expected {
		if count == 0 {
			t.Fatalf("endpoint was not exercised: %s", endpoint)
		}
	}
}

func quickReferenceResponseData(method, path string) interface{} {
	switch method + " " + path {
	case "GET /v3/guild/list", "GET /v3/guild/user-list", "GET /v3/guild-boost/history",
		"GET /v3/channel/list", "GET /v3/message/list", "GET /v3/direct-message/list",
		"GET /v3/user-chat/list", "GET /v3/guild-role/list", "GET /v3/intimacy/index",
		"GET /v3/guild-emoji/list", "GET /v3/invite/list", "GET /v3/blacklist/list",
		"GET /v3/game", "GET /v3/thread/list":
		return map[string]interface{}{"items": []interface{}{}, "meta": map[string]interface{}{}, "sort": map[string]interface{}{}}
	case "GET /v3/guild/view":
		return map[string]interface{}{"id": "guild-1", "name": "guild"}
	case "GET /v3/guild-mute/list":
		return map[string]interface{}{"mic": map[string]interface{}{"type": 1, "user_ids": []string{}}, "headset": map[string]interface{}{"type": 2, "user_ids": []string{}}}
	case "GET /v3/channel/view", "POST /v3/channel/create", "POST /v3/channel/update":
		return map[string]interface{}{"id": "channel-1", "name": "channel"}
	case "GET /v3/channel/user-list", "GET /v3/channel-user/get-joined-channel", "GET /v3/message/reaction-list",
		"GET /v3/direct-message/reaction-list", "GET /v3/badge/guild":
		return []interface{}{}
	case "GET /v3/channel-role/index", "POST /v3/channel-role/sync":
		return map[string]interface{}{"permission_overwrites": []interface{}{}, "permission_users": []interface{}{}, "permission_sync": 0}
	case "POST /v3/channel-role/create", "POST /v3/channel-role/update":
		return map[string]interface{}{"user_id": "user-1", "allow": 1, "deny": 0}
	case "GET /v3/message/view", "GET /v3/direct-message/view":
		return map[string]interface{}{"id": "msg-1", "content": "hello"}
	case "POST /v3/message/create", "POST /v3/direct-message/create", "POST /v3/message/send-pipemsg":
		return map[string]interface{}{"msg_id": "msg-1", "msg_timestamp": 1, "nonce": "nonce"}
	case "GET /v3/user-chat/view", "POST /v3/user-chat/create":
		return map[string]interface{}{"code": "chat-1", "target_info": map[string]interface{}{"id": "user-1"}}
	case "GET /v3/friend":
		return map[string]interface{}{"request": []interface{}{}, "friend": []interface{}{}, "blocked": []interface{}{}}
	case "GET /v3/gateway/index":
		return map[string]interface{}{"url": "ws://example.test"}
	case "GET /v3/user/me", "GET /v3/user/view":
		return map[string]interface{}{"id": "user-1", "username": "user"}
	case "POST /v3/asset/create":
		return map[string]interface{}{"url": "https://example.test/test.txt", "name": "test.txt", "size": 2}
	case "POST /v3/guild-role/create", "POST /v3/guild-role/update":
		return []interface{}{map[string]interface{}{"role_id": 1, "name": "role"}}
	case "POST /v3/guild-role/grant", "POST /v3/guild-role/revoke":
		return map[string]interface{}{"user_id": "user-1", "role_id": 1}
	case "POST /v3/intimacy/update":
		return map[string]interface{}{"user_id": "user-1", "score": 10}
	case "POST /v3/guild-emoji/create", "POST /v3/guild-emoji/update":
		return map[string]interface{}{"id": "emoji-1", "name": "emoji"}
	case "POST /v3/invite/create":
		return map[string]interface{}{"url_code": "invite-code", "guild_id": "guild-1", "channel_id": "channel-1"}
	case "POST /v3/game/create", "POST /v3/game/update":
		return map[string]interface{}{"id": 1, "name": "game"}
	case "GET /v3/category/list":
		return map[string]interface{}{"list": []interface{}{}}
	case "POST /v3/thread/create", "GET /v3/thread/view":
		return map[string]interface{}{"id": "post-1", "post_id": "post-1", "title": "title", "content": "content"}
	case "POST /v3/thread/reply":
		return map[string]interface{}{"id": "reply-1", "thread_id": "thread-1", "content": "reply"}
	case "GET /v3/thread/post":
		return map[string]interface{}{"items": []interface{}{}, "meta": map[string]interface{}{}}
	default:
		return []interface{}{}
	}
}
