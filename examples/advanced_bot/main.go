package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/OggEu127/kook.go/kook"
)

func main() {
	// 获取环境变量
	token := os.Getenv("KOOK_TOKEN")
	if token == "" {
		log.Fatal("请设置环境变量 KOOK_TOKEN")
	}

	// 创建客户端
	client := kook.NewClient(token)
	defer client.Close()

	// 创建WebSocket客户端
	wsClient := kook.NewWebSocketClient(client, false)

	// 设置消息处理器
	wsClient.OnMessage(kook.MessageTypeText, func(event *kook.MessageEvent) {
		var extra messageExtra
		if err := event.DecodeExtra(&extra); err != nil {
			log.Printf("解析消息扩展字段失败: %v", err)
			return
		}
		author := extra.Author
		guildID := extra.GuildID

		// 忽略机器人消息
		if author.Bot {
			return
		}

		contentValue, err := event.TextContent()
		if err != nil {
			log.Printf("解析消息内容失败: %v", err)
			return
		}
		content := strings.TrimSpace(contentValue)
		channelID := event.TargetID
		userID := author.ID

		log.Printf("收到用户 %s 的消息: %s", author.Username, content)

		// 处理不同的命令
		switch {
		case strings.HasPrefix(content, "!help"):
			handleHelpCommand(client, channelID)

		case strings.HasPrefix(content, "!roles"):
			handleRolesCommand(client, channelID, guildID)

		case strings.HasPrefix(content, "!emojis"):
			handleEmojisCommand(client, channelID, guildID)

		case strings.HasPrefix(content, "!blacklist"):
			handleBlacklistCommand(client, channelID, guildID)

		case strings.HasPrefix(content, "!pin"):
			handlePinCommand(client, channelID, event.MsgID)

		case strings.HasPrefix(content, "!game"):
			parts := strings.Split(content, " ")
			if len(parts) > 1 {
				handleGameCommand(client, channelID, parts[1])
			}

		case strings.HasPrefix(content, "!music"):
			parts := strings.Split(content, " ")
			if len(parts) > 2 {
				handleMusicCommand(client, channelID, parts[1], parts[2])
			}

		case strings.HasPrefix(content, "!invites"):
			handleInvitesCommand(client, channelID, guildID)

		case strings.HasPrefix(content, "!badges"):
			handleBadgesCommand(client, channelID, guildID)

		case strings.HasPrefix(content, "!nickname"):
			parts := strings.Split(content, " ")
			if len(parts) > 1 {
				nickname := strings.Join(parts[1:], " ")
				handleNicknameCommand(client, channelID, guildID, userID, nickname)
			}

		case strings.HasPrefix(content, "!upload"):
			handleUploadCommand(client, channelID)

		default:
			// 默认回复
			if strings.Contains(content, "你好") || strings.Contains(content, "hello") {
				sendReply(client, channelID, "你好！我是KOOK机器人，输入 !help 查看可用命令。")
			}
		}
	})

	// 连接WebSocket
	log.Println("正在连接KOOK WebSocket...")
	if err := wsClient.Connect(); err != nil {
		log.Fatalf("连接WebSocket失败: %v", err)
	}

	// 等待中断信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	log.Println("正在关闭机器人...")
	_ = wsClient.Close()
}

// 帮助命令
func handleHelpCommand(client *kook.Client, channelID string) {
	helpText := `
**KOOK机器人帮助**

**基础命令：**
• !help - 显示此帮助信息
• !roles - 查看服务器角色列表
• !emojis - 查看服务器表情列表
• !blacklist - 查看服务器屏蔽用户
• !invites - 查看服务器邀请
• !badges - 查看服务器 Badge 图片信息

**消息操作：**
• !pin - 置顶当前消息

**动态设置：**
• !game <游戏名> - 设置游戏动态
• !music <歌手> <歌名> - 设置音乐动态

**用户操作：**
• !nickname <昵称> - 修改你的昵称

**文件操作：**
• !upload - 上传文件示例

所有命令都需要相应的权限才能执行。
	`
	sendReply(client, channelID, helpText)
}

// 角色列表命令
func handleRolesCommand(client *kook.Client, channelID, guildID string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	roles, err := client.GuildRole.GetRoleList(context.Background(), kook.RoleListParams{
		GuildID: guildID, Page: intPtr(1), PageSize: intPtr(10),
	})
	if err != nil {
		log.Printf("获取角色列表失败: %v", err)
		sendReply(client, channelID, "获取角色列表失败："+err.Error())
		return
	}

	if len(roles.Items) == 0 {
		sendReply(client, channelID, "此服务器没有自定义角色。")
		return
	}

	roleText := "**服务器角色列表：**\n"
	for _, role := range roles.Items {
		roleText += fmt.Sprintf("• %s (ID: %d, 权限: %d)\n", role.Name, role.RoleID, role.Permissions)
	}

	sendReply(client, channelID, roleText)
}

// 表情列表命令
func handleEmojisCommand(client *kook.Client, channelID, guildID string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	emojis, err := client.GuildEmoji.GetEmojiList(context.Background(), kook.EmojiListParams{
		GuildID: guildID, Page: intPtr(1), PageSize: intPtr(10),
	})
	if err != nil {
		log.Printf("获取表情列表失败: %v", err)
		sendReply(client, channelID, "获取表情列表失败："+err.Error())
		return
	}

	if len(emojis.Items) == 0 {
		sendReply(client, channelID, "此服务器没有自定义表情。")
		return
	}

	emojiText := "**服务器表情列表：**\n"
	for _, emoji := range emojis.Items {
		emojiText += fmt.Sprintf("• %s (ID: %s)\n", emoji.Name, emoji.ID)
	}

	sendReply(client, channelID, emojiText)
}

// 屏蔽用户列表命令
func handleBlacklistCommand(client *kook.Client, channelID, guildID string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	blacklist, err := client.Blacklist.GetBlacklistUsers(context.Background(), kook.BlacklistListParams{
		GuildID: guildID, Page: intPtr(1), PageSize: intPtr(10),
	})
	if err != nil {
		log.Printf("获取屏蔽用户列表失败: %v", err)
		sendReply(client, channelID, "获取屏蔽用户列表失败："+err.Error())
		return
	}

	if len(blacklist.Items) == 0 {
		sendReply(client, channelID, "此服务器没有屏蔽用户。")
		return
	}

	blacklistText := "**服务器屏蔽用户列表：**\n"
	for _, user := range blacklist.Items {
		blacklistText += fmt.Sprintf("• %s (备注: %s)\n", user.User.Username, user.Remark)
	}

	sendReply(client, channelID, blacklistText)
}

// 置顶消息命令
func handlePinCommand(client *kook.Client, channelID, msgID string) {
	err := client.Message.PinMessage(context.Background(), kook.MessagePinParams{MsgID: msgID, TargetID: channelID})
	if err != nil {
		log.Printf("置顶消息失败: %v", err)
		sendReply(client, channelID, "置顶消息失败："+err.Error())
		return
	}

	sendReply(client, channelID, "消息已置顶！")
}

// 游戏动态命令
func handleGameCommand(client *kook.Client, channelID, gameName string) {
	// 首先获取游戏列表，查找匹配的游戏
	games, err := client.Game.GetGameList(context.Background(), kook.GameListParams{})
	if err != nil {
		sendReply(client, channelID, "获取游戏列表失败："+err.Error())
		return
	}

	var gameID int
	for _, game := range games.Items {
		if strings.Contains(strings.ToLower(game.Name), strings.ToLower(gameName)) {
			gameID = game.ID
			break
		}
	}

	if gameID == 0 {
		sendReply(client, channelID, fmt.Sprintf("未找到游戏：%s", gameName))
		return
	}

	err = client.Game.AddActivity(context.Background(), kook.GameActivityParams{
		ID: &gameID, DataType: kook.GameActivityTypeGame,
	})
	if err != nil {
		sendReply(client, channelID, "设置游戏动态失败："+err.Error())
		return
	}

	sendReply(client, channelID, fmt.Sprintf("已设置游戏动态：%s", gameName))
}

// 音乐动态命令
func handleMusicCommand(client *kook.Client, channelID, singer, songName string) {
	params := kook.GameActivityParams{
		DataType:  kook.GameActivityTypeMusic,
		Software:  kook.SoftwareCloudMusic,
		Singer:    singer,
		MusicName: songName,
	}

	err := client.Game.AddActivity(context.Background(), params)
	if err != nil {
		sendReply(client, channelID, "设置音乐动态失败："+err.Error())
		return
	}

	sendReply(client, channelID, fmt.Sprintf("已设置音乐动态：%s - %s", singer, songName))
}

// 邀请列表命令
func handleInvitesCommand(client *kook.Client, channelID, guildID string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	invites, err := client.Invite.GetInviteList(context.Background(), kook.InviteListParams{
		GuildID: guildID, Page: intPtr(1), PageSize: intPtr(10),
	})
	if err != nil {
		log.Printf("获取邀请列表失败: %v", err)
		sendReply(client, channelID, "获取邀请列表失败："+err.Error())
		return
	}

	if len(invites.Items) == 0 {
		sendReply(client, channelID, "此服务器没有邀请链接。")
		return
	}

	inviteText := "**服务器邀请列表：**\n"
	for _, invite := range invites.Items {
		inviteText += fmt.Sprintf("• %s (创建者: %s)\n", invite.URLCode, invite.User.Username)
	}

	sendReply(client, channelID, inviteText)
}

// 徽章列表命令
func handleBadgesCommand(client *kook.Client, channelID, guildID string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	badge, err := client.Badge.GetGuildBadge(context.Background(), kook.BadgeParams{
		GuildID: guildID,
		Style:   kook.BadgeStyleOnlineAndTotal,
	})
	if err != nil {
		log.Printf("获取徽章列表失败: %v", err)
		sendReply(client, channelID, "获取徽章列表失败："+err.Error())
		return
	}

	sendReply(client, channelID, fmt.Sprintf("Badge 已获取：%s，%d 字节，ETag=%s", badge.ContentType, len(badge.Data), badge.ETag))
}

// 昵称修改命令
func handleNicknameCommand(client *kook.Client, channelID, guildID, userID, nickname string) {
	if guildID == "" {
		sendReply(client, channelID, "此命令只能在服务器中使用。")
		return
	}

	err := client.Guild.UpdateGuildMemberNickname(context.Background(), kook.GuildNicknameParams{
		GuildID: guildID, UserID: userID, Nickname: &nickname,
	})
	if err != nil {
		log.Printf("修改昵称失败: %v", err)
		sendReply(client, channelID, "修改昵称失败："+err.Error())
		return
	}

	sendReply(client, channelID, fmt.Sprintf("昵称已修改为：%s", nickname))
}

// 文件上传命令
func handleUploadCommand(client *kook.Client, channelID string) {
	// 这里只提示调用方式，实际使用时应传入真实文件路径。
	sendReply(client, channelID, "文件上传请使用 client.Asset.CreateFromPath(ctx, kook.AssetPathParams{Path: filePath})")

	// 实际的文件上传示例：
	// asset, err := client.Asset.CreateFromPath(context.Background(), kook.AssetPathParams{Path: "path/to/file.txt"})
	// if err != nil {
	//     log.Printf("上传文件失败: %v", err)
	//     sendReply(client, channelID, "上传文件失败："+err.Error())
	//     return
	// }
	// message := fmt.Sprintf("文件上传成功！\n文件链接：%s", asset.URL)
	// sendReply(client, channelID, message)
}

type messageExtra struct {
	Author  kook.User `json:"author"`
	GuildID string    `json:"guild_id"`
}

// 发送回复消息
func sendReply(client *kook.Client, channelID, content string) {
	messageType := kook.MessageTypeText
	params := kook.MessageCreateParams{
		TargetID: channelID,
		Content:  content,
		Type:     &messageType,
	}

	_, err := client.Message.Create(context.Background(), params)
	if err != nil {
		log.Printf("发送消息失败: %v", err)
	}
}

func intPtr(value int) *int {
	return &value
}
