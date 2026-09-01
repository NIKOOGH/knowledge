# Mini-RPC：手写轻量级 RPC 框架

对标 Dubbo / gRPC 的简化版，展示微服务通信底层原理。

## 架构总览

```
┌─────────────────────── Consumer（消费者） ───────────────────────────┐
│                                                                     │
│  业务代码 → 动态代理(RpcClientProxy) → 负载均衡 → 熔断器 → Netty客户端 │
│                                                        │            │
└────────────────────────────────────────────────────────┼────────────┘
                                                         │ 自定义二进制协议(TCP)
┌────────────────────────────────────────────────────────┼────────────┐
│  Netty服务端 → 协议解码 → 反射调用服务 → 编码返回响应      ▼            │
│                                              Nacos 注册中心           │
└─────────────────────────────────────────────────────────────────────┘
```

## 核心模块

| 模块 | 包 | 关键点 |
|------|-----|--------|
| 自定义协议 | `protocol` | 魔数+长度字段解决粘包/拆包 |
| 序列化 | `serialization` | 序列化抽象 + JSON实现 |
| 注册中心 | `registry` | Nacos注册/发现/订阅变更 |
| 负载均衡 | `loadbalance` | 随机 / 平滑加权轮询 / 一致性哈希 |
| 熔断降级 | `circuitbreaker` | 三态熔断器(闭→开→半开) + 滑动窗口 |
| 传输层 | `transport` | Netty服务端/客户端、异步转同步 |

## 一、自定义协议设计（核心）

### 报文格式（二进制）

```
+----------+--------+------------+--------------+-----------+--------+----------+
|  Magic   | Version| MsgType    | SerializeType| RequestId | Length |   Body   |
|  4 bytes | 1 byte | 1 byte     | 1 byte       | 8 bytes   |4 bytes | N bytes  |
|  0xCABE  |  0x01  | REQ/RESP...| JSON/KRYO... | 唯一标识   |body长度 | 序列化后  |
+----------+--------+------------+--------------+-----------+--------+----------+
```

- **魔数(Magic)**：快速识别非法连接/流量，避免解码错乱
- **Length字段**：解决 TCP 粘包/拆包的根本手段——按帧读取
- **RequestId**：客户端异步并发发送时，将响应正确配对回请求（异步转同步的关键）

### 粘包/拆包原理

TCP 是字节流协议，没有消息边界。两条消息可能被合并到一个缓冲区（粘包），
一条消息也可能被拆成多次到达（拆包）。解决方案是**在报文中携带长度字段**：

- 服务端使用 Netty 内置的 `LengthFieldBasedFrameDecoder` 按帧切分；
- 本框架同时**手写了 `ProtocolDecoder`**，展示"读到足够字节才解码"的积累式解码过程。

## 二、调用流程

```
1. Consumer 调用 helloService.sayHi("test")
2. JDK动态代理拦截方法调用 → 封装 RpcRequest{interfaceName, method, args, requestId}
3. 从本地服务列表缓存(Nacos订阅)中拉取该服务的实例列表
4. 熔断器检查：OPEN → 直接降级失败；CLOSED/HALF_OPEN → 放行
5. 负载均衡选择一个节点，通过长连接 Channel 发送（必要时建连）
6. 发送前将 RpcFuture 存入 RequestHolder，请求线程 future.get() 挂起等待
7. Provider 解码请求 → 反射定位并执行真实服务 → 回写 RpcResponse
8. Consumer 收到响应后按 requestId 取出 RpcFuture.complete()，业务线程唤醒拿到结果
```

## 三、负载均衡算法

1. **随机(Random)**：`list.get(ThreadLocalRandom.current().nextInt(size))`
2. **轮询(RoundRobin)**：平滑加权轮询（Nginx 同款算法），避免瞬时权重倾斜
3. **一致性哈希(ConsistentHash)**：虚拟节点哈希环（默认150个虚节点），节点上下线仅影响相邻区段

## 四、熔断器（简易 Sentinel）

三态模型：

```
          失败率超阈值                冷却时间到
CLOSED ───────────────▶ OPEN ──────▶ HALF_OPEN ─┐
   ▲                                            │放行一次探测请求
   └──────────────── 成功 ◀─────────────────────┘
```

- **滑动窗口统计**最近N次调用的失败率（环形数组，无锁CAS写入）
- OPEN 后所有请求快速失败（走降级逻辑），不打下游
- HALF_OPEN 放行一个探测请求：成功→恢复 CLOSED；失败→重新 OPEN

## 五、快速开始

```bash
# 1. 启动 Nacos (localhost:8848)

# 2. 启动服务提供者
mvn compile exec:java -Dexec.mainClass=com.mini.rpc.example.ProviderDemo

# 3. 启动服务消费者
mvn compile exec:java -Dexec.mainClass=com.mini.rpc.example.ConsumerDemo
```

## 六、项目结构

```
src/main/java/com/mini/rpc/
├── protocol/          # 协议层（编解码）
├── serialization/     # 序列化层
├── registry/          # 注册中心（Nacos）
├── loadbalance/       # 负载均衡策略
├── circuitbreaker/    # 熔断器
├── core/              # RpcRequest/RpcResponse/RpcFuture
├── transport/         # Netty 客户端 & 服务端
├── proxy/             # 动态代理
├── config/            # 配置
└── example/           # 使用示例
```
