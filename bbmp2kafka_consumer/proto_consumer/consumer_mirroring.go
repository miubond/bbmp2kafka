package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"
)

// ConsumerGroupHandler 实现 sarama.ConsumerGroupHandler 接口
type ConsumerGroupHandler struct {
	ready chan bool
	mu    sync.Mutex
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case <-session.Context().Done():
			return nil
		case message := <-claim.Messages():
			if message == nil {
				return nil
			}
			h.processMessage(session, message)
			session.MarkMessage(message, "")
		}
	}
}

func (h *ConsumerGroupHandler) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📨 [主题: %s | 分区: %d | 偏移: %d | 大小: %d字节]",
		msg.Topic, msg.Partition, msg.Offset, len(msg.Value))
	log.Printf("⏰ 时间戳: %v", msg.Timestamp)

	bbmpMsg := &BBMPMessage{}
	err := proto.Unmarshal(msg.Value, bbmpMsg)
	if err != nil {
		log.Printf("❌ Protobuf 解析失败: %v", err)
		return
	}

	log.Printf("  📋 解析结果:")

	switch bbmpMsg.MessageType {
	case BBMPMessage_RouteMonitoringMessage:
		log.Printf("    消息类型: RouteMonitoringMessage")
		log.Printf("    ✅ 路由监控消息")
		if bbmpMsg.BbmpUnicastMonitoringMessage != nil {
			displayRouteMonitoring(bbmpMsg.BbmpUnicastMonitoringMessage)
		}
	case BBMPMessage_PeerUpNotification:
		log.Printf("    消息类型: PeerUpNotification")
		log.Printf("    ✅ BGP 对等体上线")
		if bbmpMsg.BbmpUnicastMonitoringMessage != nil {
			displayPeerEvent(bbmpMsg.BbmpUnicastMonitoringMessage, true)
		}
	case BBMPMessage_PeerDownNotification:
		log.Printf("    消息类型: PeerDownNotification")
		log.Printf("    ⚠️  BGP 对等体下线")
		if bbmpMsg.BbmpUnicastMonitoringMessage != nil {
			displayPeerEvent(bbmpMsg.BbmpUnicastMonitoringMessage, false)
		}
	case BBMPMessage_RouteMirroringMessage:
		log.Printf("    消息类型: RouteMirroringMessage")
		log.Printf("    📊 路由镜像消息")
	case BBMPMessage_InitiationMessage:
		log.Printf("    消息类型: InitiationMessage")
		log.Printf("    🚀 BMP 会话初始化")
	case BBMPMessage_TerminationMessage:
		log.Printf("    消息类型: TerminationMessage")
		log.Printf("    🛑 BMP 会话终止")
	default:
		log.Printf("    ❓ 未知消息类型: %v", bbmpMsg.MessageType)
	}
}

func main() {
	brokers := []string{"localhost:9092"}
	topics := []string{"bmp.mirroring"}
	groupID := "bgp-mirroring-consumer-group"

	log.Printf("🚀 启动 Consumer Group: %s", groupID)
	log.Printf("📡 Kafka 代理: %v", brokers)
	log.Printf("📋 订阅主题: %v", topics)

	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second
	config.Consumer.Fetch.Max = 1024 * 100
	config.Consumer.Fetch.Default = 1024 * 100

	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		log.Fatalf("❌ 创建 Consumer Group 失败: %v", err)
	}
	defer consumerGroup.Close()

	handler := &ConsumerGroupHandler{ready: make(chan bool)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			if err := consumerGroup.Consume(ctx, topics, handler); err != nil {
				log.Printf("❌ Consumer Group 错误: %v", err)
			}
			if ctx.Err() != nil {
				return
			}
			handler.ready = make(chan bool)
		}
	}()

	<-handler.ready
	log.Println("✅ Consumer Group 已就绪，开始消费消息...")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 收到停止信号，正在优雅关闭...")
	cancel()
	time.Sleep(2 * time.Second)
	log.Println("👋 程序已退出")
}
