package main

import (
	"fmt"
	"log"
	"time"

	api "github.com/bio-routing/bio-rd/net/api"
	api1 "github.com/bio-routing/bio-rd/route/api"
)

// 避免未使用导入错误
var _ = api1.BGPPath{}

// displayPeerEvent 显示 Peer 事件详情（共享函数）
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

// displayRouteMonitoring 显示路由监控消息详情（共享函数）
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

// formatIP 格式化 IP 地址（共享函数）
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

// formatPrefix 格式化前缀（共享函数）
func formatPrefix(pfx *api.Prefix) string {
	if pfx == nil || pfx.Address == nil {
		return "N/A"
	}
	return fmt.Sprintf("%s/%d", formatIP(pfx.Address), pfx.Length)
}
