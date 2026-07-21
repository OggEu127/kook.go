package kook

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

// compileV111Calls is intentionally not executed. It makes representative
// v1.1.1 direct calls part of the package's compile-time compatibility gate.
func compileV111Calls(ctx context.Context, client *Client) {
	_, _ = client.Gateway.GetGateway(ctx, 1)
	_, _ = client.Gateway.GetVoiceGateway(ctx, "channel")
	_, _ = client.User.GetUser(ctx, "user", "guild")
	_, _ = client.Guild.GetGuildList(ctx, 1, 20, "-id")
	_, _ = client.Guild.GetGuildInfo(ctx, "guild")
	_, _ = client.Guild.GetGuildMembers(ctx, "guild", 1, 20, "-id")
	_, _ = client.Guild.GetGuildMember(ctx, "guild", "user")
	_, _ = client.Guild.GetRegions(ctx)
	_, _ = client.Guild.GetGuildBoostInfo(ctx, "guild")
	_ = client.Guild.UpdateNickname(ctx, "guild", "user", "nickname")
	_, _ = client.Channel.GetChannelList(ctx, "guild", 1, 20, "-id")
	_, _ = client.Channel.GetChannelInfo(ctx, "channel")
	_, _ = client.Channel.CreateChannel(ctx, "guild", CreateChannelParams{Name: "name"})
	_, _ = client.Channel.UpdateChannel(ctx, "channel", UpdateChannelParams{})
	_ = client.Channel.MoveUsers(ctx, "channel", []string{"user"})
	_, _ = client.Channel.CreateChannelRole(ctx, ChannelRoleParams{ChannelID: "channel"})
	_, _ = client.Channel.GetJoinedChannels(ctx, "guild", "user")
	_, _ = client.Message.SendMessage(ctx, SendMessageParams{TargetID: "channel", Content: "hello"})
	_, _ = client.Message.GetMessageList(ctx, "channel", GetMessageListParams{})
	_, _ = client.Message.GetMessage(ctx, "message")
	_ = client.Message.AddReaction(ctx, "message", "emoji")
	_ = client.Message.DeleteReaction(ctx, "message", "emoji", "user")
	_ = client.Message.PinMessage(ctx, "message", "channel")
	_, _ = client.UserChat.GetUserChatList(ctx, 1, 20)
	_, _ = client.UserChat.GetUserChat(ctx, "chat")
	_, _ = client.Role.GetRoleList(ctx, "guild", 1, 20)
	_, _ = client.Emoji.GetEmojiList(ctx, "guild", 1, 20)
	_, _ = client.Blacklist.GetBlacklistUsers(ctx, "guild", 1, 20)
	_, _ = client.Invite.GetInviteList(ctx, "guild", 1, 20)
	_, _ = client.Intimacy.UpdateIntimacyLegacy(ctx, "user", 1, "social", "image")
	_, _ = client.Friend.GetFriendsList(ctx)
	_ = client.Friend.HandleFriendRequestLegacy(ctx, "1", true)
	_, _ = client.Game.CreateGame(ctx, "game", "icon")
	_, _ = client.Thread.GetThreadCategories(ctx, "channel")
	_, _ = client.Thread.ReplyThreadLegacy(ctx, ReplyThreadParams{})
	_, _ = client.Voice.JoinVoiceChannel(ctx, "channel")
	_, _ = client.Asset.UploadFileContent(ctx, "asset.png", []byte("asset"))
	_ = client.Admin.BanUser(ctx, "guild", "user", "reason", 0)
	_, _ = client.Region.GetRegionList(ctx)
	_, _ = client.Live.GetLiveInfo(ctx, "channel")
	_, _ = client.Security.GetVerificationLevel(ctx, "guild")
	_, _ = client.Item.GetBag(ctx)
	_, _ = client.Order.GetOrders(ctx, 1, 20)
	_, _ = client.Coupon.GetCoupons(ctx, 1, 20)
	_, _ = client.Boost.GetUnusedBoostNum(ctx)
	client.WebSocketCompatCompileOnly(ctx)
}

var _ = compileV111Calls

// WebSocketCompatCompileOnly keeps legacy event registration source-checked.
func (c *Client) WebSocketCompatCompileOnly(context.Context) {
	ws := NewWebSocketClient(c, false)
	ws.OnEvent(EventTypeTextMessage, func(*Event) {})
	webhook := NewWebhookHandler(c, "", "verify")
	webhook.OnEvent(EventTypeTextMessage, func(*Event) {})
	_ = Asset{URL: "url", Type: "image", Name: "name", Size: 1}
	_ = OAuthTokenResponse{RefreshToken: "refresh"}
	_ = APIError{}
	_ = GuildMember{}
	_ = FriendRequest{}
}

func TestLegacyAliasesAndUnsupportedEndpoint(t *testing.T) {
	client := NewClient("test", WithoutRateLimit(), WithoutRetry())
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	require.Same(t, client.GuildRole, client.Role)
	require.Same(t, client.GuildEmoji, client.Emoji)
	err := client.Admin.BanUser(context.Background(), "guild", "user", "reason", 0)
	require.ErrorIs(t, err, ErrUnsupportedEndpoint)
}

func TestCompatibilityArgumentsReturnValidationError(t *testing.T) {
	client := NewClient("test", WithoutRateLimit(), WithoutRetry())
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	_, err := client.Guild.GetGuildInfo(context.Background(), 123)
	var validationErr *ValidationError
	require.Error(t, err)
	require.True(t, errors.As(err, &validationErr))
}
