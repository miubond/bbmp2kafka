package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	api "github.com/bio-routing/bio-rd/net/api"
	"google.golang.org/protobuf/proto"
)

// ConsumerGroupHandler 实现 sarama.ConsumerGroupHandler 接口
type ConsumerGroupHandler struct {
	ready chan bool
	mu    sync.Mutex // 防止日志混乱
}

// Setup 在消费者组重平衡开始前调用
func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup 在消费者组重平衡结束后调用
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 处理消息
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
			// 标记消息已处理（自动提交偏移量）
			session.MarkMessage(message, "")
		}
	}
}

// processMessage 处理单个消息
func (h *ConsumerGroupHandler) processMessage(session sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📨 [主题: %s | 分区: %d | 偏移: %d | 大小: %d字节]",
		msg.Topic, msg.Partition, msg.Offset, len(msg.Value))
	log.Printf("⏰ 时间戳: %v", msg.Timestamp)

	// 解析 protobuf 消息
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

func displayPeerEvent(pe *BBMPUnicastMonitoringMessage, isUp bool) {
	if pe == nil {
		return
	}

	emoji := "🟢"
	status := "up"
	if !isUp {
		emoji = "🔴"
		status = "down"
	}

	log.Printf("    %s 对等体事件:", emoji)
	if pe.RouterIp != nil {
		log.Printf("      路由器 IP: %s", formatIP(pe.RouterIp))
	}
	if pe.LocalBpgIp != nil {
		log.Printf("      本地 IP: %s", formatIP(pe.LocalBpgIp))
	}
	if pe.NeighborBgpIp != nil {
		log.Printf("      对等体 IP: %s", formatIP(pe.NeighborBgpIp))
	}
	if pe.LocalAs != 0 {
		log.Printf("      本地 AS: %d", pe.LocalAs)
	}
	if pe.RemoteAs != 0 {
		log.Printf("      对等体 AS: %d", pe.RemoteAs)
	}
	log.Printf("      状态: %s", status)
	if pe.Timestamp != 0 {
		t := time.Unix(int64(pe.Timestamp), 0)
		log.Printf("      时间: %v", t.Format("2006-01-02 15:04:05"))
	}
}

func displayRouteMonitoring(rm *BBMPUnicastMonitoringMessage) {
	if rm == nil {
		return
	}

	log.Printf("    🛣️ 路由信息:")
	if rm.RouterIp != nil {
		log.Printf("      路由器 IP: %s", formatIP(rm.RouterIp))
	}
	if rm.LocalBpgIp != nil {
		log.Printf("      本地 BGP IP: %s", formatIP(rm.LocalBpgIp))
	}
	if rm.NeighborBgpIp != nil {
		log.Printf("      邻居 BGP IP: %s", formatIP(rm.NeighborBgpIp))
	}
	if rm.LocalAs != 0 {
		log.Printf("      本地 AS: %d", rm.LocalAs)
	}
	if rm.RemoteAs != 0 {
		log.Printf("      远程 AS: %d", rm.RemoteAs)
	}

	log.Printf("      公告: %v", rm.Announcement)

	if rm.Pfx != nil {
		log.Printf("      前缀: %s", formatPrefix(rm.Pfx))
	}

	if rm.Timestamp != 0 {
		t := time.Unix(int64(rm.Timestamp), 0)
		log.Printf("      时间戳: %v", t.Format("2006-01-02 15:04:05"))
	}

	if rm.BgpPath != nil {
		log.Printf("      🛤️ BGP 路径详情:")
		if rm.BgpPath.NextHop != nil {
			log.Printf("        下一跳: %s", formatIP(rm.BgpPath.NextHop))
		}
		if rm.BgpPath.LocalPref != 0 {
			log.Printf("        本地优先级: %d", rm.BgpPath.LocalPref)
		}
		if rm.BgpPath.AsPath != nil && len(rm.BgpPath.AsPath) > 0 {
			var asns []uint32
			for _, segment := range rm.BgpPath.AsPath {
				if segment.Asns != nil {
					asns = append(asns, segment.Asns...)
				}
			}
			if len(asns) > 0 {
				log.Printf("        AS 路径: %v", asns)
			}
		}
		if rm.BgpPath.Source != nil {
			log.Printf("        源地址: %s", formatIP(rm.BgpPath.Source))
		}
		if rm.BgpPath.Communities != nil && len(rm.BgpPath.Communities) > 0 {
			communities := make([]string, 0, len(rm.BgpPath.Communities))
			for _, c := range rm.BgpPath.Communities {
				communities = append(communities, fmt.Sprintf("%d:%d", c>>16, c&0xFFFF))
			}
			log.Printf("        社区属性: %v", communities)
		}
	}
}

func formatIP(ip *api.IP) string {
	if ip == nil {
		return "N/A"
	}

	// Check if it's an IPv4 address (stored in Lower field)
	if ip.Lower != 0 {
		// Convert uint64 to IPv4 address
		ipv4 := uint32(ip.Lower)
		return fmt.Sprintf("%d.%d.%d.%d",
			(ipv4>>24)&0xFF,
			(ipv4>>16)&0xFF,
			(ipv4>>8)&0xFF,
			ipv4&0xFF)
	}

	return "unknown"
}

func formatPrefix(pfx *api.Prefix) string {
	if pfx == nil || pfx.Address == nil {
		return "N/A"
	}
	return fmt.Sprintf("%s/%d", formatIP(pfx.Address), pfx.Length)
}

func main() {
	// 配置参数
	brokers := []string{"localhost:9092"}
	topics := []string{
		"bmp.pre-policy",
		"bmp.post-policy",
		"bmp.mirroring",
	}
	groupID := "bgp-proto-consumer-group-v2" // Consumer Group ID (修改版本号可重新从最新消息开始)

	log.Printf("🚀 启动 Consumer Group: %s", groupID)
	log.Printf("📡 Kafka 代理: %v", brokers)
	log.Printf("📋 订阅主题: %v", topics)

	// Sarama 配置
	config := sarama.NewConfig()
	config.Version = sarama.V2_0_0_0

	// Consumer Group 配置
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetNewest // 从最旧消息开始消费（处理所有历史消息）
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second

	// 网络配置
	config.Net.DialTimeout = 10 * time.Second
	config.Net.ReadTimeout = 10 * time.Second
	config.Net.WriteTimeout = 10 * time.Second

	// 消息大小限制
	config.Consumer.Fetch.Max = 1024 * 100 // 100KB
	config.Consumer.Fetch.Default = 1024 * 100

	// 创建 Consumer Group
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		log.Fatalf("❌ 创建 Consumer Group 失败: %v", err)
	}
	defer consumerGroup.Close()

	// 创建 handler
	handler := &ConsumerGroupHandler{
		ready: make(chan bool),
	}

	// 设置上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动消费循环
	go func() {
		for {
			// Consume 会一直运行直到出错或者 context 取消
			if err := consumerGroup.Consume(ctx, topics, handler); err != nil {
				log.Printf("❌ Consumer Group 错误: %v", err)
			}

			// 检查 context 是否已取消
			if ctx.Err() != nil {
				return
			}

			// 重置 ready channel
			handler.ready = make(chan bool)
		}
	}()

	// 等待 handler 准备就绪
	<-handler.ready
	log.Println("✅ Consumer Group 已就绪，开始消费消息...")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 监听系统信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("\n🛑 收到停止信号，正在优雅关闭...")
	cancel()

	// 等待一段时间让消费者优雅关闭
	time.Sleep(2 * time.Second)
	log.Println("👋 程序已退出")
}
