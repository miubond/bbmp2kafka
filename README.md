<div align="center">

# bbmp2kafka

**BGP Monitoring Protocol (BMP) to Kafka Bridge**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.18+-00ADD8?logo=go)](https://golang.org/)
[![Platform](https://img.shields.io/badge/Platform-Linux-orange.svg)](https://www.linux.org/)

*高性能的 BMP 消息采集与 Kafka 流式转发系统*

[特性](#特性) • [快速开始](#快速开始) • [部署](#部署) • [配置](#配置) • [架构](#架构)

</div>

---

## 目录

- [简介](#简介)
- [特性](#特性)
- [快速开始](#快速开始)
  - [前置要求](#前置要求)
  - [编译](#编译)
  - [运行](#运行)
- [部署](#部署)
- [架构](#架构)
  - [工作流程](#工作流程)
  - [Kafka 主题架构](#kafka-主题架构)
- [组件说明](#组件说明)
  - [Producer](#producer)
  - [Consumer](#consumer)
- [配置](#配置)
  - [Producer 配置](#producer-配置)
  - [Consumer 配置](#consumer-配置)
- [输出示例](#输出示例)
- [运维指南](#运维指南)
- [项目结构](#项目结构)
- [性能优化](#性能优化)
- [故障排查](#故障排查)
- [License](#license)
- [参考资源](#参考资源)

---

## 简介

`bbmp2kafka` 是一个高性能的 BGP 监控协议 ([BMP RFC 7854](https://datatracker.ietf.org/doc/html/rfc7854)) 消息采集器，能够实时接收来自路由器的 BMP 消息流，并将其转发到 Apache Kafka 消息队列。

### 应用场景

- 🌐 **BGP 路由监控**: 实时追踪全网路由变化
- 📊 **网络可视化**: 为监控平台提供数据源
- 🔍 **故障分析**: 记录对等体状态变化和路由异常
- 📈 **流量工程**: 分析 AS 路径和路由策略效果

---

## 特性

### ✅ BMP 消息支持

| 消息类型 | 状态 | 功能描述 |
|---------|------|---------|
| **Route Monitoring** | ✅ 完全支持 | 路由更新、前缀、AS 路径、社区属性 |
| **Peer Up Notification** | ✅ 完全支持 | BGP 对等体建立事件 |
| **Peer Down Notification** | ✅ 完全支持 | BGP 对等体断开事件及原因 |
| **Statistics Report** | ⚠️ 部分支持 | 识别消息类型 |
| **Initiation Message** | ⚠️ 部分支持 | 识别消息类型 |
| **Termination Message** | ⚠️ 部分支持 | 识别消息类型 |

### 🚀 核心特性

- **高性能**: 基于 [bio-routing/bio-rd](https://github.com/bio-routing/bio-rd) 高效解析
- **Protocol Buffers**: 使用 Protobuf 序列化，减少网络传输
- **多主题路由**: 根据 BMP 消息类型智能路由到不同 Kafka 主题
- **健康检查**: 内置 HTTP 健康检查和 Prometheus metrics
- **流量控制**: Token Bucket 限流器防止消息风暴
- **灵活消费**: 提供多种 Consumer 满足不同消费场景

---

## 快速开始

### 前置要求

- **Go**: 1.18 或更高版本
- **Kafka**: 2.0+ 或兼容版本
- **操作系统**: Linux (生产环境)

### 编译

#### 在 Windows 开发环境编译

```powershell
# 编译 Producer (Linux x64)
$env:GOOS='linux'; $env:GOARCH='amd64'
go build -mod=vendor -o bbmp2kafka

# 编译 Consumers
cd bbmp2kafka_consumer/proto_consumer

go build -mod=vendor -o consumer_pre_policy consumer_pre_policy.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_post_policy consumer_post_policy.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_mirroring consumer_mirroring.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_all_topics consumer_all_topics.go bbmp.pb.go
```

#### 在 Linux 环境编译

```bash
# 编译 Producer
make build

# 或手动编译
go build -mod=vendor -o bbmp2kafka

# 编译 Consumers
cd bbmp2kafka_consumer/proto_consumer
go build -mod=vendor -o consumer_pre_policy consumer_pre_policy.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_post_policy consumer_post_policy.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_mirroring consumer_mirroring.go common_functions.go bbmp.pb.go
go build -mod=vendor -o consumer_all_topics consumer_all_topics.go bbmp.pb.go
```

### 运行

#### 启动 Producer

```bash
./bbmp2kafka \
  -bmp.listen.addr=:5000 \
  -kafka.cluster=localhost:9092 \
  -kafka.topic=bmp \
  -health.listen.addr=:8080
```

#### 启动 Consumer

```bash
# 方式1: 消费所有主题（推荐用于开发和测试）
./consumer_all_topics

# 方式2: 按需消费单个主题（推荐用于生产环境）
./consumer_pre_policy      # 只消费 pre-policy 路由
./consumer_post_policy     # 只消费 post-policy 路由
./consumer_mirroring       # 只消费 mirroring 路由
```

---

## 部署

### 使用 SCP 部署

```bash
# 1. 上传二进制文件
scp bbmp2kafka user@server:/opt/bbmp2kafka/
scp bbmp2kafka_consumer/proto_consumer/consumer_* user@server:/opt/bbmp2kafka/

# 2. SSH 登录服务器
ssh user@server

# 3. 设置权限
cd /opt/bbmp2kafka
chmod +x bbmp2kafka consumer_*

# 4. 创建服务目录
mkdir -p /var/log/bbmp2kafka
```

### Systemd 服务配置

创建 `/etc/systemd/system/bbmp2kafka.service`:

```ini
[Unit]
Description=BMP to Kafka Producer
After=network.target kafka.service

[Service]
Type=simple
User=bbmp
WorkingDirectory=/opt/bbmp2kafka
ExecStart=/opt/bbmp2kafka/bbmp2kafka \
  -bmp.listen.addr=:5000 \
  -kafka.cluster=localhost:9092 \
  -kafka.topic=bmp \
  -health.listen.addr=:8080
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
```

启动服务：

```bash
sudo systemctl daemon-reload
sudo systemctl enable bbmp2kafka
sudo systemctl start bbmp2kafka
sudo systemctl status bbmp2kafka
```

### Docker 部署 (可选)

```dockerfile
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY bbmp2kafka /app/
EXPOSE 5000 8080
CMD ["./bbmp2kafka", "-bmp.listen.addr=:5000", "-kafka.cluster=kafka:9092", "-kafka.topic=bmp"]
```

---

## 架构

### 工作流程

```
BGP Router (BMP Session)
         │
         │ BMP Messages
         ↓
   ┌─────────────┐
   │  bbmp2kafka │ (Producer)
   │   :5000     │
   └─────────────┘
         │
         │ Protobuf Messages
         ↓
   ┌─────────────┐
   │    Kafka    │
   │  :9092      │
   └─────────────┘
         │
         ├─→ bmp.pre-policy
         ├─→ bmp.post-policy
         └─→ bmp.mirroring
         │
         ↓
   ┌─────────────┐
   │  Consumers  │
   └─────────────┘
```

### Kafka 主题架构

Producer 根据 BMP 消息类型自动路由到不同的 Kafka 主题：

| Kafka 主题 | BMP 消息类型 | 说明 |
|-----------|-------------|------|
| `bmp.pre-policy` | Route Monitoring (Pre-Policy Adj-RIB-In) | 策略应用前的路由信息 |
| `bmp.post-policy` | Route Monitoring (Post-Policy Adj-RIB-In) | 策略应用后的路由信息 |
| `bmp.mirroring` | Route Monitoring (Local RIB) | 本地路由表镜像 |

**消息格式**: 所有消息使用 Protocol Buffers 序列化 (定义见 `protos/bbmp/bbmp.proto`)

---

## 组件说明

### Producer

**二进制**: `bbmp2kafka` (~24 MB)

**功能**:
- 监听 TCP 端口接收 BMP 连接
- 解析 BMP 消息（Route Monitoring、Peer Up/Down 等）
- 序列化为 Protobuf 格式
- 发送到 Kafka 的对应主题
- 提供健康检查接口 (`/health`)
- 导出 Prometheus 指标 (`/metrics`)

**关键模块**:
- `main.go`: 主程序入口和命令行参数处理
- `adjRIBIn.go`: 路由监控消息处理
- `bmpEventHandler.go`: Peer 事件处理
- `tokenBucket.go`: 流量控制

### Consumer

| 程序 | 消费主题 | Consumer Group ID | 适用场景 |
|------|---------|-------------------|---------|
| `consumer_all_topics` | pre-policy<br>post-policy<br>mirroring | bgp-proto-consumer-group-v2 | 开发测试、完整监控 |
| `consumer_pre_policy` | pre-policy | bgp-pre-policy-consumer-group | 策略前路由分析 |
| `consumer_post_policy` | post-policy | bgp-post-policy-consumer-group | 策略后路由分析 |
| `consumer_mirroring` | mirroring | bgp-mirroring-consumer-group | 本地路由表监控 |

**功能**:
- 从 Kafka 消费 Protobuf 消息
- 反序列化并格式化输出
- 支持断点续传（Consumer Group）
- 彩色日志输出

---

## 配置

### Producer 配置

| 参数 | 类型 | 默认值 | 说明 |
|------|-----|-------|------|
| `-bmp.listen.addr` | string | `:5000` | BMP 监听地址和端口 |
| `-kafka.cluster` | string | *必需* | Kafka broker 地址（逗号分隔） |
| `-kafka.topic` | string | *必需* | Kafka 主题前缀 |
| `-health.listen.addr` | string | `:8080` | 健康检查和 Metrics 端口 |

**示例**:

```bash
./bbmp2kafka \
  -bmp.listen.addr=0.0.0.0:5000 \
  -kafka.cluster=kafka1:9092,kafka2:9092,kafka3:9092 \
  -kafka.topic=bmp \
  -health.listen.addr=:8080
```

### Consumer 配置

在源代码中修改（位于各 `consumer_*.go` 文件）:

```go
// Kafka broker 地址
brokers := []string{"localhost:9092"}

// 消费的主题
topics := []string{"bmp.pre-policy"}

// Consumer Group ID
groupID := "bgp-pre-policy-consumer-group"

// 起始偏移量
config.Consumer.Offsets.Initial = sarama.OffsetNewest  // 或 OffsetOldest
```

---

## 输出示例

### 路由监控消息

```
📨 [主题: bmp.post-policy | 分区: 10 | 偏移: 12345]
⏰ 时间戳: 2025-10-24 18:30:45
  📋 解析结果:
    消息类型: RouteMonitoringMessage
    ✅ 路由监控消息
    🛣️  路由信息:
      路由器 IP: 168.76.252.251
      邻居 BGP IP: 45.159.58.253
      本地 AS: 65001
      远程 AS: 65002
      公告: true
      前缀: 192.168.4.0/24
      🛤️  BGP 路径详情:
        下一跳: 10.0.0.1
        AS 路径: [65002, 65003, 65004]
        社区属性: [65001:100, 65001:200]
        本地优先级: 100
        MED: 50
```

### Peer Up 事件

```
📨 [主题: bmp.pre-policy | 分区: 0 | 偏移: 456]
⏰ 时间戳: 2025-10-24 18:31:00
  📋 解析结果:
    消息类型: PeerUpNotification
    ✅ BGP 对等体上线
    🟢 对等体事件:
      路由器 IP: 168.76.252.251
      对等体 IP: 45.159.58.253
      对等体 AS: 65002
      状态: up
      时间: 2025-10-24 18:31:00
```

### Peer Down 事件

```
📨 [主题: bmp.pre-policy | 分区: 0 | 偏移: 457]
⏰ 时间戳: 2025-10-24 18:35:12
  📋 解析结果:
    消息类型: PeerDownNotification
    ⚠️  BGP 对等体下线
    🔴 对等体事件:
      路由器 IP: 168.76.252.251
      对等体 IP: 45.159.58.253
      对等体 AS: 65002
      状态: down
      原因: 连接重置
      时间: 2025-10-24 18:35:12
```

---

## 运维指南

### 健康检查

```bash
# 检查 Producer 健康状态
curl http://localhost:8080/health

# 查看 Prometheus 指标
curl http://localhost:8080/metrics
```

### 进程管理

```bash
# 查看进程状态
ps aux | grep bbmp2kafka
ps aux | grep consumer_

# 优雅停止
pkill -SIGTERM bbmp2kafka
pkill -SIGTERM consumer_all_topics

# 强制停止
pkill -SIGKILL bbmp2kafka
```

### 日志管理

```bash
# 实时查看日志
tail -f /var/log/bbmp2kafka/producer.log
tail -f /var/log/bbmp2kafka/consumer.log

# 查看最近错误
journalctl -u bbmp2kafka -n 100 --no-pager

# 日志轮转配置 /etc/logrotate.d/bbmp2kafka
/var/log/bbmp2kafka/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
}
```

### 监控指标

Producer 提供以下 Prometheus 指标：

- `bmp_messages_received_total`: 接收的 BMP 消息总数
- `bmp_messages_parsed_total`: 成功解析的消息数
- `bmp_messages_failed_total`: 解析失败的消息数
- `kafka_messages_sent_total`: 发送到 Kafka 的消息数
- `kafka_errors_total`: Kafka 发送错误数

---

## 项目结构

```
bbmp2kafka/
├── main.go                          # Producer 主程序
├── adjRIBIn.go                      # 路由监控消息处理
├── bmpEventHandler.go               # Peer 事件处理器
├── tokenBucket.go                   # Token Bucket 限流器
├── Makefile                         # 构建配置
├── go.mod                           # Go 模块依赖
├── go.sum                           # 依赖校验和
├── LICENSE                          # Apache 2.0 许可证
├── README.md                        # 本文档
├── protos/
│   └── bbmp/
│       ├── bbmp.proto               # Protobuf 消息定义
│       └── bbmp.pb.go               # 生成的 Go 代码
├── bbmp2kafka_consumer/
│   └── proto_consumer/
│       ├── consumer_all_topics.go   # 全主题消费者
│       ├── consumer_pre_policy.go   # Pre-policy 消费者
│       ├── consumer_post_policy.go  # Post-policy 消费者
│       ├── consumer_mirroring.go    # Mirroring 消费者
│       ├── common_functions.go      # 通用函数库
│       ├── bbmp.pb.go               # Protobuf 定义（副本）
│       ├── go.mod                   # 消费者模块依赖
│       └── vendor/                  # 依赖包
├── vendor/                          # Producer 依赖包
└── example_data/
    └── bmp.raw                      # BMP 消息样本数据
```

---

## 性能优化

### Producer 调优

1. **增加 Kafka 批处理大小**（修改 `main.go`）:
   ```go
   config.Producer.Flush.Messages = 100
   config.Producer.Flush.Frequency = 100 * time.Millisecond
   ```

2. **调整 Go runtime**:
   ```bash
   GOMAXPROCS=4 ./bbmp2kafka ...
   ```

3. **启用 Kafka 压缩**:
   ```go
   config.Producer.Compression = sarama.CompressionSnappy
   ```

### Consumer 调优

1. **增加 Consumer 并发数**（修改消费者代码）:
   ```go
   config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
   config.ChannelBufferSize = 256
   ```

2. **调整提交间隔**:
   ```go
   config.Consumer.Offsets.AutoCommit.Interval = 5 * time.Second
   ```

---

## 故障排查

### 常见问题

#### 1. Producer 无法连接 Kafka

**症状**: 日志显示 "kafka: client has run out of available brokers"

**解决方案**:
```bash
# 检查 Kafka 服务状态
systemctl status kafka

# 检查网络连通性
telnet kafka-server 9092

# 检查防火墙
sudo iptables -L -n | grep 9092
```

#### 2. Consumer 无法消费消息

**症状**: Consumer 启动但没有输出

**解决方案**:
```bash
# 检查 Consumer Group 状态
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group bgp-proto-consumer-group-v2

# 重置 Offset（谨慎使用）
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --group bgp-proto-consumer-group-v2 --reset-offsets --to-earliest --execute --all-topics
```

#### 3. BMP 连接失败

**症状**: 路由器无法建立 BMP 会话

**解决方案**:
```bash
# 检查端口监听
netstat -tlnp | grep 5000

# 检查防火墙
sudo ufw allow 5000/tcp

# 查看连接日志
journalctl -u bbmp2kafka | grep -i error
```

### 调试模式

启用详细日志：

```bash
# Producer
./bbmp2kafka -log.level=debug ...

# Consumer（修改代码中的日志级别）
sarama.Logger = log.New(os.Stdout, "[sarama] ", log.LstdFlags)
```

---

## License

本项目基于 **Apache License 2.0** 开源。

```
Copyright (c) 2022 Cloudflare, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
```

详见 [LICENSE](LICENSE) 文件。

---

## 参考资源

### 协议与标准

- 📜 [RFC 7854 - BGP Monitoring Protocol (BMP)](https://datatracker.ietf.org/doc/html/rfc7854)
- 📜 [RFC 4271 - Border Gateway Protocol 4 (BGP-4)](https://datatracker.ietf.org/doc/html/rfc4271)

### 依赖项目

- [bio-routing/bio-rd](https://github.com/bio-routing/bio-rd) - BGP/BMP 协议实现
- [IBM/sarama](https://github.com/IBM/sarama) - Kafka Go 客户端
- [Protocol Buffers](https://developers.google.com/protocol-buffers) - 数据序列化

### 相关工具

- [Apache Kafka](https://kafka.apache.org/) - 分布式消息队列
- [Prometheus](https://prometheus.io/) - 监控和告警系统
- [Grafana](https://grafana.com/) - 可视化平台

---

<div align="center">

**Made with ❤️ for Network Engineers**

[⬆ 返回顶部](#bbmp2kafka)

</div>
