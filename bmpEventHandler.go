//
// Copyright (c) 2022 Cloudflare, Inc.
//
// Licensed under Apache 2.0 license found in the LICENSE file
// or at http://www.apache.org/licenses/LICENSE-2.0
//

package main

import (
	"sync/atomic"
	"time"

	"github.com/cloudflare/bbmp2kafka/protos/bbmp"

	"github.com/IBM/sarama"
	"google.golang.org/protobuf/proto"

	bnet "github.com/bio-routing/bio-rd/net"

	log "github.com/sirupsen/logrus"
)

// BMPEventHandler 全局 BMP 事件处理器
type BMPEventHandler struct {
	producer    sarama.SyncProducer
	kafkaTopic  string
	tokenBucket *tokenBucket
}

// NewBMPEventHandler 创建新的 BMP 事件处理器
func NewBMPEventHandler(producer sarama.SyncProducer, kafkaTopic string, tokenBucket *tokenBucket) *BMPEventHandler {
	return &BMPEventHandler{
		producer:    producer,
		kafkaTopic:  kafkaTopic,
		tokenBucket: tokenBucket,
	}
}

// OnPeerUp 实现 BMPEventCallback 接口 - 当 Peer 上线时调用
func (h *BMPEventHandler) OnPeerUp(routerIP, localIP, peerIP bnet.IP, localAS, peerAS uint32) {
	log.WithFields(log.Fields{
		"router_ip": routerIP.String(),
		"peer_ip":   peerIP.String(),
		"peer_as":   peerAS,
	}).Info("Sending Peer Up notification to Kafka")

	// 使用现有的 BBMPUnicastMonitoringMessage 结构发送 Peer 事件
	// MessageType 设置为 PeerUpNotification
	msg := &bbmp.BBMPMessage{
		MessageType: bbmp.BBMPMessage_PeerUpNotification,
		BbmpUnicastMonitoringMessage: &bbmp.BBMPUnicastMonitoringMessage{
			RouterIp:      routerIP.ToProto(),
			LocalBpgIp:    localIP.ToProto(),
			NeighborBgpIp: peerIP.ToProto(),
			LocalAs:       localAS,
			RemoteAs:      peerAS,
			Announcement:  true, // 标记为 peer up
			Timestamp:     uint32(time.Now().Unix()),
			Pfx:           nil, // Peer 事件没有前缀
			BgpPath:       nil, // Peer 事件没有路径
		},
	}

	h.sendMessage(msg)
	messagesProcessed.Inc()
}

// OnPeerDown 实现 BMPEventCallback 接口 - 当 Peer 下线时调用
func (h *BMPEventHandler) OnPeerDown(routerIP, localIP, peerIP bnet.IP, localAS, peerAS uint32, reason string) {
	log.WithFields(log.Fields{
		"router_ip": routerIP.String(),
		"peer_ip":   peerIP.String(),
		"peer_as":   peerAS,
		"reason":    reason,
	}).Info("Sending Peer Down notification to Kafka")

	msg := &bbmp.BBMPMessage{
		MessageType: bbmp.BBMPMessage_PeerDownNotification,
		BbmpUnicastMonitoringMessage: &bbmp.BBMPUnicastMonitoringMessage{
			RouterIp:      routerIP.ToProto(),
			LocalBpgIp:    localIP.ToProto(),
			NeighborBgpIp: peerIP.ToProto(),
			LocalAs:       localAS,
			RemoteAs:      peerAS,
			Announcement:  false, // 标记为 peer down
			Timestamp:     uint32(time.Now().Unix()),
			Pfx:           nil, // Peer 事件没有前缀
			BgpPath:       nil, // Peer 事件没有路径
		},
	}

	h.sendMessage(msg)
	messagesProcessed.Inc()
}

// sendMessage 发送消息到 Kafka
func (h *BMPEventHandler) sendMessage(msg *bbmp.BBMPMessage) {
	msgBytes, err := proto.Marshal(msg)
	if err != nil {
		messagesMarshalFailed.Inc()
		if h.tokenBucket.getToken() {
			log.Errorf("failed to marshal BMP event message: %v", err)
		}
		return
	}

	_, _, err = h.producer.SendMessage(&sarama.ProducerMessage{
		Topic: h.kafkaTopic,
		Value: sarama.ByteEncoder(msgBytes),
	})
	if err != nil {
		atomic.StoreInt32(&healthy, 0)
		messagesSendFailed.Inc()
		if h.tokenBucket.getToken() {
			log.Errorf("could not send BMP event message: %v", err)
		}
		return
	}

	atomic.StoreInt32(&healthy, 1)
}

