package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/OggEu127/kook.go/kook"
)

func main() {
	// 获取环境变量
	token := os.Getenv("KOOK_TOKEN")
	if token == "" {
		log.Fatal("请设置环境变量 KOOK_TOKEN")
	}

	// 创建客户端
	client, err := kook.NewClientWithError(token)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer func() { _ = client.Close() }()

	// 获取当前用户信息
	fmt.Println("=== 获取机器人信息 ===")
	user, err := client.User.GetMe(context.Background())
	if err != nil {
		log.Printf("获取用户信息失败: %v", err)
	} else {
		fmt.Printf("机器人名称: %s#%s\n", user.Username, user.IdentifyNum)
		fmt.Printf("机器人ID: %s\n", user.ID)
	}

	// 获取在线状态
	fmt.Println("\n=== 获取在线状态 ===")
	status, err := client.User.GetOnlineStatus(context.Background())
	if err != nil {
		log.Printf("获取在线状态失败: %v", err)
	} else {
		fmt.Printf("在线状态: %t\n", status.Online)
		fmt.Printf("在线平台: %v\n", status.OnlineOS)
	}

	// 获取服务器列表
	fmt.Println("\n=== 获取服务器列表 ===")
	page, pageSize := 1, 10
	guilds, err := client.Guild.GetGuildList(context.Background(), kook.GuildListParams{
		Page: &page, PageSize: &pageSize,
	})
	if err != nil {
		log.Printf("获取服务器列表失败: %v", err)
	} else {
		fmt.Printf("服务器数量: %d\n", len(guilds.Items))
		if len(guilds.Items) > 0 {
			guild := guilds.Items[0]
			fmt.Printf("- %s (ID: %s)\n", guild.Name, guild.ID)

			// 演示角色管理API
			fmt.Printf("\n=== 服务器 %s 的角色管理 ===\n", guild.Name)
			roles, err := client.GuildRole.GetRoleList(context.Background(), kook.RoleListParams{
				GuildID: guild.ID, Page: &page, PageSize: &pageSize,
			})
			if err != nil {
				log.Printf("获取角色列表失败: %v", err)
			} else {
				fmt.Printf("角色数量: %d\n", len(roles.Items))
				for _, role := range roles.Items {
					fmt.Printf("- %s (ID: %d, 权限: %d)\n", role.Name, role.RoleID, role.Permissions)
				}
			}

			// 演示频道管理API
			fmt.Printf("\n=== 服务器 %s 的频道列表 ===\n", guild.Name)
			channels, err := client.Channel.GetChannelList(context.Background(), kook.ChannelListParams{
				GuildID: guild.ID, Page: &page, PageSize: &pageSize,
			})
			if err != nil {
				log.Printf("获取频道列表失败: %v", err)
			} else {
				fmt.Printf("频道数量: %d\n", len(channels.Items))
				for _, channel := range channels.Items {
					fmt.Printf("- %s (ID: %s, 类型: %d)\n", channel.Name, channel.ID, channel.Type)
				}
			}

			// 演示邀请管理API
			fmt.Printf("\n=== 服务器 %s 的邀请管理 ===\n", guild.Name)
			invites, err := client.Invite.GetInviteList(context.Background(), kook.InviteListParams{
				GuildID: guild.ID, Page: &page, PageSize: &pageSize,
			})
			if err != nil {
				log.Printf("获取邀请列表失败: %v", err)
			} else {
				fmt.Printf("邀请数量: %d\n", len(invites.Items))
				for _, invite := range invites.Items {
					fmt.Printf("- 邀请码: %s, 创建者: %s\n", invite.URLCode, invite.User.Username)
				}
			}

			inviteeStatus := kook.InviteeStatusAll
			invitees, err := client.Invite.GetInvitees(context.Background(), kook.InviteeListParams{
				GuildID: guild.ID, Status: &inviteeStatus, Page: 1, PageSize: pageSize,
			})
			if err != nil {
				log.Printf("获取受邀用户统计失败: %v", err)
			} else {
				fmt.Printf("受邀统计: 总数=%d, 留存=%d, 流失=%d\n", invitees.Count, invitees.KeepCount, invitees.LossCount)
			}

		}
	}

	// 演示游戏API
	fmt.Println("\n=== 游戏管理 ===")
	games, err := client.Game.GetGameList(context.Background(), kook.GameListParams{})
	if err != nil {
		log.Printf("获取游戏列表失败: %v", err)
	} else {
		fmt.Printf("游戏数量: %d\n", len(games.Items))
		for i, game := range games.Items {
			if i < 5 { // 只显示前5个游戏
				fmt.Printf("- %s (ID: %d, 类型: %d)\n", game.Name, game.ID, game.Type)
			}
		}
	}

	// 演示好友API
	fmt.Println("\n=== 好友管理 ===")
	friends, err := client.Friend.GetFriendsList(context.Background(), kook.FriendListParams{})
	if err != nil {
		log.Printf("获取好友列表失败: %v", err)
	} else {
		fmt.Printf("好友数量: %d\n", len(friends.Friend))
		fmt.Printf("好友请求数量: %d\n", len(friends.Request))
		fmt.Printf("屏蔽用户数量: %d\n", len(friends.Blocked))
	}

	// 演示消息模板API
	fmt.Println("\n=== 消息模板 ===")
	templates, err := client.Template.GetTemplateList(context.Background())
	if err != nil {
		log.Printf("获取消息模板列表失败: %v", err)
	} else {
		fmt.Printf("模板数量: %d\n", len(templates.Items))
	}

	fmt.Println("\n=== API演示完成 ===")
	fmt.Println("所有API接口已成功调用，详细的错误处理和功能展示请查看日志输出。")
	fmt.Println("请根据实际需要调用相应的API接口。")
}
