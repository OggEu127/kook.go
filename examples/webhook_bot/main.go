package main

import (
	"context"
	"log"
	"os"

	"github.com/OggEu127/kook.go/kook"
)

func main() {
	// 从环境变量获取配置
	token := os.Getenv("KOOK_TOKEN")
	verifyToken := os.Getenv("KOOK_VERIFY_TOKEN")

	if token == "" {
		log.Fatal("请设置环境变量 KOOK_TOKEN")
	}

	// 创建客户端
	client := kook.NewClient(token)
	defer client.Close()

	// 获取机器人信息
	user, err := client.User.GetMe(context.Background())
	if err != nil {
		log.Fatalf("获取机器人信息失败: %v", err)
	}

	log.Printf("机器人启动成功: %s#%s", user.Username, user.IdentifyNum)

	// 创建Webhook处理器
	webhook := kook.NewWebhookHandler(client, "", verifyToken)

	// 注册消息事件处理器
	webhook.OnMessage(kook.MessageTypeText, func(event *kook.MessageEvent) {
		content, err := event.TextContent()
		if err != nil {
			log.Printf("解析消息失败: %v", err)
			return
		}
		log.Printf("收到消息: %s", content)

		// 简单的回复逻辑
		if content == "hello" {
			// 发送回复消息
			messageType := kook.MessageTypeText
			params := kook.MessageCreateParams{
				TargetID: event.TargetID,
				Content:  "Hello! 我是KOOK机器人 🤖",
				Type:     &messageType,
			}

			_, err := client.Message.Create(context.Background(), params)
			if err != nil {
				log.Printf("发送消息失败: %v", err)
			}
		}
	})

	// 启动Webhook服务器
	log.Println("启动Webhook服务器在 :8080/webhook")
	if err := webhook.StartWebhookServer(":8080", "/webhook"); err != nil {
		log.Fatalf("启动Webhook服务器失败: %v", err)
	}
}
