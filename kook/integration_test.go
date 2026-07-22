package kook

import (
	"context"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var requiredKOOKIntegrationEnv = []string{
	"KOOK_TOKEN",
	"KOOK_TEST_GUILD_ID",
	"KOOK_TEST_TEXT_CHANNEL_ID",
	"KOOK_TEST_VOICE_CHANNEL_ID",
	"KOOK_TEST_THREAD_CHANNEL_ID",
	"KOOK_TEST_USER_ID",
	"KOOK_TEST_CHAT_CODE",
	"KOOK_TEST_MESSAGE_ID",
	"KOOK_TEST_ROLE_ID",
	"KOOK_TEST_EMOJI_ID",
}

func requireKOOKIntegrationEnv(t *testing.T) map[string]string {
	t.Helper()
	if testing.Short() {
		t.Skip("-short skips real KOOK integration tests")
	}
	values := make(map[string]string, len(requiredKOOKIntegrationEnv))
	var missing []string
	for _, name := range requiredKOOKIntegrationEnv {
		value := os.Getenv(name)
		if value == "" {
			missing = append(missing, name)
			continue
		}
		values[name] = value
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Fatalf("real KOOK read-only tests require: %v; use go test -short ./... for offline tests", missing)
	}
	return values
}

func TestKOOKReadOnlyIntegration(t *testing.T) {
	env := requireKOOKIntegrationEnv(t)
	client := NewClient(env["KOOK_TOKEN"])
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	tests := []struct {
		name string
		run  func() error
	}{
		{"gateway", func() error {
			_, err := client.Gateway.GetGateway(ctx, GatewayParams{Compress: testPtr(0)})
			return err
		}},
		{"user-me", func() error { _, err := client.User.GetMe(ctx); return err }},
		{"user-view", func() error {
			_, err := client.User.GetUser(ctx, UserViewParams{UserID: env["KOOK_TEST_USER_ID"], GuildID: env["KOOK_TEST_GUILD_ID"]})
			return err
		}},
		{"user-online-status", func() error { _, err := client.User.GetOnlineStatus(ctx); return err }},
		{"guild-list", func() error {
			_, err := client.Guild.GetGuildList(ctx, GuildListParams{Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"guild-view", func() error {
			_, err := client.Guild.GetGuildInfo(ctx, GuildViewParams{GuildID: env["KOOK_TEST_GUILD_ID"]})
			return err
		}},
		{"guild-members", func() error {
			_, err := client.Guild.GetGuildMembers(ctx, GuildMembersParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"guild-mute-list", func() error {
			_, err := client.Guild.GetGuildMuteList(ctx, GuildMuteListParams{GuildID: env["KOOK_TEST_GUILD_ID"], ReturnType: "detail"})
			return err
		}},
		{"guild-boost-history", func() error {
			_, err := client.Guild.GetGuildBoostHistory(ctx, GuildBoostHistoryParams{GuildID: env["KOOK_TEST_GUILD_ID"]})
			return err
		}},
		{"channel-list", func() error {
			_, err := client.Channel.GetChannelList(ctx, ChannelListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"channel-view", func() error {
			_, err := client.Channel.View(ctx, ChannelViewParams{TargetID: env["KOOK_TEST_TEXT_CHANNEL_ID"], NeedChildren: testPtr(true)})
			return err
		}},
		{"voice-channel-users", func() error {
			_, err := client.Channel.GetChannelUserList(ctx, ChannelUserListParams{ChannelID: env["KOOK_TEST_VOICE_CHANNEL_ID"]})
			return err
		}},
		{"channel-role", func() error {
			_, err := client.Channel.GetChannelRole(ctx, ChannelRoleViewParams{ChannelID: env["KOOK_TEST_TEXT_CHANNEL_ID"]})
			return err
		}},
		{"joined-channel", func() error {
			_, err := client.ChannelUser.GetJoinedChannels(ctx, JoinedChannelParams{GuildID: env["KOOK_TEST_GUILD_ID"], UserID: env["KOOK_TEST_USER_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"message-list", func() error {
			_, err := client.Message.GetMessageList(ctx, MessageListParams{TargetID: env["KOOK_TEST_TEXT_CHANNEL_ID"], PageSize: testPtr(10)})
			return err
		}},
		{"message-view", func() error {
			_, err := client.Message.GetMessage(ctx, MessageViewParams{MsgID: env["KOOK_TEST_MESSAGE_ID"]})
			return err
		}},
		{"message-reactions", func() error {
			_, err := client.Message.ReactionUsers(ctx, MessageReactionUsersParams{MsgID: env["KOOK_TEST_MESSAGE_ID"], Emoji: env["KOOK_TEST_EMOJI_ID"]})
			return err
		}},
		{"user-chat-list", func() error {
			_, err := client.UserChat.GetUserChatList(ctx, UserChatListParams{Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"user-chat-view", func() error {
			_, err := client.UserChat.GetUserChat(ctx, UserChatViewParams{ChatCode: env["KOOK_TEST_CHAT_CODE"]})
			return err
		}},
		{"direct-message-list", func() error {
			_, err := client.DirectMessage.List(ctx, DirectMessageListParams{ChatCode: env["KOOK_TEST_CHAT_CODE"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"role-list", func() error {
			_, err := client.GuildRole.GetRoleList(ctx, RoleListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"emoji-list", func() error {
			_, err := client.GuildEmoji.GetEmojiList(ctx, EmojiListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"blacklist-list", func() error {
			_, err := client.Blacklist.GetBlacklistUsers(ctx, BlacklistListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"invite-list", func() error {
			_, err := client.Invite.GetInviteList(ctx, InviteListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Page: testPtr(1), PageSize: testPtr(10)})
			return err
		}},
		{"invitees", func() error {
			_, err := client.Invite.GetInvitees(ctx, InviteeListParams{GuildID: env["KOOK_TEST_GUILD_ID"], Status: testPtr(InviteeStatusAll), Page: 1, PageSize: 10})
			return err
		}},
		{"intimacy", func() error {
			_, err := client.Intimacy.GetIntimacy(ctx, IntimacyViewParams{UserID: env["KOOK_TEST_USER_ID"]})
			return err
		}},
		{"friend-list", func() error { _, err := client.Friend.GetFriendsList(ctx, FriendListParams{}); return err }},
		{"game-list", func() error { _, err := client.Game.GetGameList(ctx, GameListParams{Type: GameTypeAll}); return err }},
		{"badge", func() error {
			_, err := client.Badge.GetGuildBadge(ctx, BadgeParams{GuildID: env["KOOK_TEST_GUILD_ID"]})
			return err
		}},
		{"thread-categories", func() error {
			_, err := client.Thread.GetThreadCategories(ctx, ThreadCategoryListParams{ChannelID: env["KOOK_TEST_THREAD_CHANNEL_ID"]})
			return err
		}},
		{"thread-list", func() error {
			_, err := client.Thread.GetThreadList(ctx, GetThreadListParams{ChannelID: env["KOOK_TEST_THREAD_CHANNEL_ID"], PageSize: testPtr(10)})
			return err
		}},
		{"voice-list", func() error { _, err := client.Voice.GetJoinedVoiceChannels(ctx); return err }},
		{"template-list", func() error { _, err := client.Template.GetTemplateList(ctx); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) { require.NoError(t, test.run()) })
	}
}

func TestKOOKMutationIntegration(t *testing.T) {
	if os.Getenv("KOOK_ENABLE_MUTATION_TESTS") != "1" {
		t.Skip("set KOOK_ENABLE_MUTATION_TESTS=1 to run write tests")
	}
	env := requireKOOKIntegrationEnv(t)
	client := NewClient(env["KOOK_TOKEN"])
	defer func() { _ = client.Close() }()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	message, err := client.Message.Create(ctx, MessageCreateParams{
		TargetID: env["KOOK_TEST_TEXT_CHANNEL_ID"], Content: "KOOK Go SDK mutation contract test", Type: testPtr(MessageTypeText),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		_ = client.Message.DeleteMessage(cleanupCtx, MessageDeleteParams{MsgID: message.MsgID})
	})
	require.NoError(t, client.Message.Update(ctx, UpdateMessageParams{MsgID: message.MsgID, Content: "KOOK Go SDK mutation contract test updated"}))
}

func TestKOOKOAuthIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("-short skips OAuth integration")
	}
	if os.Getenv("KOOK_ENABLE_OAUTH_TEST") != "1" {
		t.Skip("set KOOK_ENABLE_OAUTH_TEST=1 to run the one-time authorization-code test")
	}
	names := []string{"KOOK_OAUTH_CLIENT_ID", "KOOK_OAUTH_CLIENT_SECRET", "KOOK_OAUTH_CODE", "KOOK_OAUTH_REDIRECT_URI"}
	values := make(map[string]string, len(names))
	for _, name := range names {
		values[name] = os.Getenv(name)
		require.NotEmpty(t, values[name], fmt.Sprintf("%s is required", name))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	token, err := NewOAuthClient().ExchangeToken(ctx, OAuthTokenParams{
		ClientID: values["KOOK_OAUTH_CLIENT_ID"], ClientSecret: values["KOOK_OAUTH_CLIENT_SECRET"],
		Code: values["KOOK_OAUTH_CODE"], RedirectURI: values["KOOK_OAUTH_REDIRECT_URI"],
	})
	require.NoError(t, err)
	require.NotEmpty(t, token.AccessToken)
}
