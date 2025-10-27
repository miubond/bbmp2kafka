# bbmp2kafka - BMP to Kafka 消息转发器

## 🎯 功能概览

将 BGP 监控协议 (BMP, RFC7854) 消息转发到 Kafka，支持完整的路由监控和对等体事件。

### ✅ 支持的 BMP 消息

| 消息类型 | Producer | Consumer | 显示内容 |
|---------|----------|----------|---------|
| **Route Monitoring** | ✅ | ✅ | 完整路由信息（前缀、AS路径、社区属性）|
| **Peer Up** | ✅ | ✅ | 对等体上线（IP、AS号、时间）|
| **Peer Down** | ✅ | ✅ | 对等体下线（IP、AS号、时间、原因）|
| Initiation | ⚠️ | ⚠️ | 仅识别类型 |
| Termination | ⚠️ | ⚠️ | 仅识别类型 |

---

## 🚀 快速开始

### 1. 编译（Windows 开发环境）

```bash
# 编译 Producer
GOOS=linux GOARCH=amd64 go build -mod=vendor -o bbmp2kafka

# 编译 Consumers
cd bbmp2kafka_consumer/proto_consumer
GOOS=linux GOARCH=amd64 go build -o consumer_pre_policy consumer_pre_policy.go common_functions.go bbmp.pb.go
GOOS=linux GOARCH=amd64 go build -o consumer_post_policy consumer_post_policy.go common_functions.go bbmp.pb.go
GOOS=linux GOARCH=amd64 go build -o consumer_mirroring consumer_mirroring.go common_functions.go bbmp.pb.go
GOOS=linux GOARCH=amd64 go build -o consumer_all_topics consumer_all_topics.go bbmp.pb.go
```

### 2. 部署到 Linux

```bash
# 上传文件
scp bbmp2kafka root@server:~/bbmp2kafka/
scp bbmp2kafka_consumer/proto_consumer/consumer_* root@server:~/bbmp2kafka/

# 设置权限
ssh root@server
cd ~/bbmp2kafka
chmod +x bbmp2kafka consumer_*
```

### 3. 运行

```bash
# 启动 Producer
./bbmp2kafka \
  -bmp.listen.addr=:5000 \
  -kafka.cluster=localhost:9092 \
  -kafka.topic=bmp.events \
  -health.listen.addr=:8080

# 启动 Consumer（选择一个）
./consumer_all_topics          # 消费所有3个主题
./consumer_post_policy         # 只消费 bmp.post-policy
./consumer_pre_policy          # 只消费 bmp.pre-policy
./consumer_mirroring           # 只消费 bmp.mirroring
```

---

## 📦 程序说明

### Producer

**bbmp2kafka** (~23 MB)
- 监听 BMP 连接（默认端口 5000）
- 接收所有 BMP 消息
- 转发路由监控和 Peer 事件到 Kafka
- 提供健康检查和 Prometheus metrics (端口 8080)

### Consumers（4个版本）

| 程序 | 消费的 Topic | Group ID |
|------|-------------|----------|
| `consumer_pre_policy` | bmp.pre-policy | bgp-pre-policy-consumer-group |
| `consumer_post_policy` | bmp.post-policy | bgp-post-policy-consumer-group |
| `consumer_mirroring` | bmp.mirroring | bgp-mirroring-consumer-group |
| `consumer_all_topics` | 所有3个 | bgp-proto-consumer-group-v2 |

---

## 📊 输出示例

### 路由监控消息

```
📨 [主题: bmp.post-policy | 分区: 10 | 偏移: 12345]
⏰ 时间戳: 2025-10-24 18:30:45
  📋 解析结果:
    消息类型: RouteMonitoringMessage
    ✅ 路由监控消息
    🛣️ 路由信息:
      路由器 IP: 168.76.252.251
      邻居 BGP IP: 45.159.58.253
      本地 AS: 18013
      远程 AS: 18013
      公告: true
      前缀: 192.168.4.0/24
      🛤️ BGP 路径详情:
        下一跳: 10.0.0.1
        AS 路径: [1000]
        社区属性: [65321:4637]
```

### Peer Up 事件

```
📨 [主题: bmp.events | 分区: 0 | 偏移: 456]
⏰ 时间戳: 2025-10-24 18:31:00
  📋 解析结果:
    消息类型: PeerUpNotification
    ✅ BGP 对等体上线
    🟢 对等体事件:
      路由器 IP: 168.76.252.251
      对等体 IP: 45.159.58.253
      对等体 AS: 18013
      状态: up
      时间: 2025-10-24 18:31:00
```

### Peer Down 事件

```
📨 [主题: bmp.events | 分区: 0 | 偏移: 457]
⏰ 时间戳: 2025-10-24 18:35:12
  📋 解析结果:
    消息类型: PeerDownNotification
    ⚠️  BGP 对等体下线
    🔴 对等体事件:
      路由器 IP: 168.76.252.251
      对等体 IP: 45.159.58.253
      对等体 AS: 18013
      状态: down
      时间: 2025-10-24 18:35:12
```

---

## 🔧 常用命令

```bash
# 查看进程状态
ps aux | grep bbmp2kafka

# 查看日志
tail -f consumer.log

# 健康检查
curl http://localhost:8080/health

# 停止服务
pkill -SIGINT bbmp2kafka
pkill -SIGINT consumer_
```

---

## 📖 详细文档

查看 `bbmp2kafka_consumer/proto_consumer/` 目录中的文档：
- **QUICK_START.md** - 快速开始指南
- **README.md** - 原始项目说明

---

## 🏗️ 项目结构

```
bbmp2kafka/
├── bbmp2kafka                    # Producer (Linux binary)
├── main.go                       # Producer 主程序
├── adjRIBIn.go                  # 路由处理
├── bmpEventHandler.go           # Peer 事件处理
├── tokenBucket.go               # 限流器
├── protos/bbmp/                 # Protobuf 定义
└── bbmp2kafka_consumer/
    └── proto_consumer/
        ├── consumer_pre_policy   # Pre-policy Consumer
        ├── consumer_post_policy  # Post-policy Consumer
        ├── consumer_mirroring    # Mirroring Consumer
        ├── consumer_all_topics   # All Topics Consumer
        └── common_functions.go   # 共享函数
```

---

## ⚙️ 配置

### Producer 参数

| 参数 | 默认值 | 说明 |
|------|-------|------|
| `-bmp.listen.addr` | :5000 | BMP 监听地址 |
| `-kafka.cluster` | (必需) | Kafka 服务器地址 |
| `-kafka.topic` | (必需) | Kafka 主题名称 |
| `-health.listen.addr` | :8080 | 健康检查端口 |

### Consumer 配置

在代码中修改：
- `brokers` - Kafka 服务器地址
- `topics` - 订阅的主题
- `groupID` - Consumer Group ID
- `OffsetOldest/OffsetNewest` - 消费起点

---

## 📝 License

Apache 2.0 License - Copyright (c) 2022 Cloudflare, Inc.

---

## 🔗 相关资源

- BMP RFC: https://datatracker.ietf.org/doc/html/rfc7854
- Bio-Routing: https://github.com/bio-routing/bio-rd
- Kafka: https://kafka.apache.org/

---

**编译成功，ready to deploy!** 🚀

