# 03_goredis —— go-redis v9 Redis 客户端入门

> 对标 Java Jedis / Lettuce / Spring RedisTemplate。

## 运行

```powershell
$env:GOROOT = "d:\文件\test\tools\go"
$env:Path   = "d:\文件\test\tools\go\bin;$env:Path"
$env:GOPROXY = "https://goproxy.cn,direct"

cd d:\文件\test\go_learn\frameworks\03_goredis
go mod tidy
go run main.go
```

**无 Redis 也能运行**：程序启动后先 Ping 一次，不可达时会打印 Docker 一键启动命令并退出（不会 panic）。

## 启动 Redis（若已启动可跳过）

```bash
# Docker 方式（最快）
docker run -d --name redis -p 6379:6379 redis:7

# 或 Windows 原生：下载 https://github.com/tporadowski/redis/releases
```

## 预期输出

```
===== 2. String =====
[String] 计数器 自增后 = 5（预期 5）
[String] MGet a,b,c = [1 2 3]
[String] SETNX lock:demo = true（第一次 true，再次 false）

===== 3. Hash =====
[Hash] age = 20
[Hash] HGetAll 全部字段 = map[age:20 email:a@x.com name:Alice]

===== 4. List =====
[List] LPUSH 后长度 = 3
[List] BRPop 取出 = [demo:list:queue task1]
[List] BRPop 取出 = [demo:list:queue task2]
[List] BRPop 取出 = [demo:list:queue task3]
[List] 队列消费完

===== 5. Set + ZSet =====
[Set] SCard = 3（3 个元素，'Go' 去重）
[Set] SIsMember(Go) = true
[ZSet] 排行榜 Top3：
    1: Alice score=100
    2: Carol score=95
    3: Bob score=80

===== 6. Pipeline =====
[Pipeline] 批量写入 100 条，第 42 条值 = 42（预期 42）

===== 7. Lua 秒杀 =====
[Lua 秒杀] 20 人抢 10 件，成功 10 人，剩余库存 = 0（预期 = 0）

===== 8. Pub/Sub =====
[Pub/Sub] 订阅等待 3 条消息：
    收到：news-A
    收到：news-B
    收到：news-C
```

## 核心知识点

| 功能 | 代码位置 |
|------|---------|
| 单机连接 / 连接池 | `NewSimpleClient()` |
| 哨兵 / 集群连接 | 注释掉的 `NewSentinelClient` / `NewClusterClient` |
| String：SET + EX / INCR / MGet / SETNX（分布式锁） | `demoString` |
| Hash：HSet / HGet / HGetAll | `demoHash` |
| List：LPush / BRPop（阻塞消费队列） | `demoList` |
| Set 去重 / ZSet 排行榜 TopN | `demoSetAndZSet` |
| Pipeline 批量命令（减少 RTT） | `demoPipeline` |
| Lua 脚本原子操作（20 人抢 10 件库存） | `demoLuaSeckill`（秒杀常用） |
| Pub/Sub 发布订阅 | `demoPubSub` |

## 常用对照（Redis 命令 → go-redis API）

| Redis 命令 | go-redis 方法 |
|------------|--------------|
| `SET k v EX 30` | `r.Set(ctx, "k", "v", 30*time.Second)` |
| `GET k` | `r.Get(ctx, "k").Result()` |
| `SETNX k v EX 10` | `r.SetNX(ctx, "k", "v", 10*time.Second)` |
| `EXPIRE k 60` | `r.Expire(ctx, "k", 60*time.Second)` |
| `HGETALL u:1` | `r.HGetAll(ctx, "u:1").Result()` → `map[string]string` |
| `LPUSH q a b c` | `r.LPush(ctx, "q", "a", "b", "c")` |
| `BRPOP q 0` | `r.BRPop(ctx, 0, "q").Result()`（阻塞直到有数据） |
| `ZREVRANGE rank 0 9 WITHSCORES` | `r.ZRevRangeWithScores(ctx, "rank", 0, 9)` |
| `SCARD s` | `r.SCard(ctx, "s")` |
| `DEL k1 k2` | `r.Del(ctx, "k1", "k2")` |
| `EXISTS k` | `r.Exists(ctx, "k").Result()` |
| `SCRIPT LOAD + EVALSHA` | `redis.NewScript("...")` → `.Run(ctx, r, keys, args...)`（自动用 eval 缓存） |

## 生产小技巧

1. **全局 Context**：超时 / 取消 / 链路追踪，所有 Redis 调用都传 ctx（HTTP Handler 用 `c.Request.Context()`）
2. **超时**：Options 的 ReadTimeout/WriteTimeout 不要设太大（2-3 秒就够）
3. **错误处理**：判断空值不要用 `err == "redis: nil"`，用 `errors.Is(err, redis.Nil)`
4. **分布式锁**：不要用 SETNX 自己实现，推荐官方 `github.com/go-redsync/redsync/v4`
5. **批量查**：MGet > 多次 Get；Pipeline > 多次 RTT
