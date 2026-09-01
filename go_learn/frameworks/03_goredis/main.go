// ============================================================
// frameworks/03_goredis/main.go  go-redis v9 入门示例
// ------------------------------------------------------------
// go-redis 是 Go 生态最常用的 Redis 客户端（v9 对应 Redis 6/7），
// 对标 Java Jedis / Lettuce / Spring RedisTemplate。
//
// 关键示例：
//   1) 连接 Redis（单机 / 哨兵 / Cluster）
//   2) 通用命令（Set / Get / Del / Exists / Expire）
//   3) 数据结构：String / List / Set / ZSet / Hash
//   4) Pipeline：一次发送多条命令减少 RTT
//   5) 事务（TxPipeline / WATCH）
//   6) Lua 脚本：原子操作（秒杀常用）
//   7) 发布订阅 Pub/Sub
//
// 运行步骤（无需真实 Redis 也能体验代码结构）：
//   cd frameworks/03_goredis
//   go mod tidy
//   go run main.go
//
//   如果本机没有 Redis，程序会打印连接失败并给出示例命令（不会 panic）。
//   启动 Redis 最简单方法：
//     Windows：docker run -d --name redis -p 6379:6379 redis:7
//     Linux：同上
// ============================================================

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// 全局 ctx：实际项目应该按请求 / 任务传
var bg = context.Background()

// ============================================================
// 1. 连接
// ============================================================

// NewSimpleClient 单机 Redis 客户端（最常用）
func NewSimpleClient() *redis.Client {
	// 如果有密码：Password: "xxx"；如果是 DB：DB: 1
	return redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:6379",
		Password:     "",
		DB:           0,
		PoolSize:     50,   // 连接池大小（默认 10 * CPU 核数）
		MinIdleConns: 10,   // 最小空闲连接
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	})
}

// 哨兵模式（高可用）—— 注释版，改配置即可用
// func NewSentinelClient() *redis.Client {
//     return redis.NewFailoverClient(&redis.FailoverOptions{
//         MasterName:    "mymaster",
//         SentinelAddrs: []string{"10.0.0.1:26379", "10.0.0.2:26379", "10.0.0.3:26379"},
//     })
// }

// 集群模式（6 节点）—— 注释版
// func NewClusterClient() *redis.ClusterClient {
//     return redis.NewClusterClient(&redis.ClusterOptions{
//         Addrs: []string{":7001", ":7002", ":7003", ":7004", ":7005", ":7006"},
//     })
// }

// ============================================================
// 2. String（最基础）
// ============================================================
func demoString(r *redis.Client) {
	key := "demo:string:counter"

	// Set + EX（秒级过期）
	err := r.Set(bg, key, 0, 30*time.Minute).Err()
	if err != nil {
		log.Fatal("Set 失败：", err)
	}

	// INCR（原子自增，做计数器 / 限流 / 订单号）
	for i := 0; i < 5; i++ {
		newVal, err := r.Incr(bg, key).Result()
		if err != nil {
			log.Fatal(err)
		}
		_ = newVal
	}
	count, _ := r.Get(bg, key).Int()
	fmt.Printf("[String] 计数器 自增后 = %d（预期 5）\n", count)

	// MGet 批量取（减少网络往返）
	keys := []string{"demo:string:a", "demo:string:b", "demo:string:c"}
	_ = r.MSet(bg, "demo:string:a", 1, "demo:string:b", 2, "demo:string:c", 3)
	vals, _ := r.MGet(bg, keys...).Result()
	fmt.Printf("[String] MGet a,b,c = %v\n", vals)

	// SETNX：不存在才设置（分布式锁基础）
	ok, _ := r.SetNX(bg, "lock:demo", "1", 10*time.Second).Result()
	fmt.Printf("[String] SETNX lock:demo = %v（第一次 true，再次 false）\n", ok)
}

// ============================================================
// 3. Hash（字段结构；类似 Java Map<String,String>）
// ============================================================
func demoHash(r *redis.Client) {
	key := "demo:hash:user:1001"
	r.Del(bg, key)

	r.HSet(bg, key, "name", "Alice")
	r.HSet(bg, key, map[string]interface{}{
		"age":   "20",
		"email": "a@x.com",
	})

	age, _ := r.HGet(bg, key, "age").Int()
	fmt.Printf("[Hash] age = %d\n", age)

	fields, _ := r.HGetAll(bg, key).Result()
	fmt.Printf("[Hash] HGetAll 全部字段 = %v\n", fields)
}

// ============================================================
// 4. List（队列 / 栈，lpush/rpop 实现 FIFO 队列）
// ============================================================
func demoList(r *redis.Client) {
	key := "demo:list:queue"
	r.Del(bg, key)

	// 入队：LPUSH
	r.LPush(bg, key, "task1", "task2", "task3") // 头插
	length, _ := r.LLen(bg, key).Result()
	fmt.Printf("[List] LPUSH 后长度 = %d\n", length)

	// 出队：BRPOP 阻塞式 pop（做异步消费者 / 任务队列最常用）
	// 加 1 秒超时，返回 key 与 value
	for {
		vals, err := r.BRPop(bg, 1*time.Second, key).Result()
		if errors.Is(err, redis.Nil) || len(vals) == 0 {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("[List] BRPop 取出 = %v（vals[0]=key vals[1]=value）\n", vals)
	}
	fmt.Println("[List] 队列消费完")
}

// ============================================================
// 5. Set（去重集合）+ ZSet（有序集合，排行榜）
// ============================================================
func demoSetAndZSet(r *redis.Client) {
	sKey := "demo:set:tag"
	r.Del(bg, sKey)

	// Set：SADD + SISMEMBER
	r.SAdd(bg, sKey, "Go", "Java", "Go", "Python")
	cnt, _ := r.SCard(bg, sKey).Result()
	fmt.Printf("[Set] SCard = %d（3 个元素，'Go' 去重）\n", cnt)
	exists, _ := r.SIsMember(bg, sKey, "Go").Result()
	fmt.Printf("[Set] SIsMember(Go) = %v\n", exists)

	// ZSet：ZADD + ZREVRANGE（排行榜：按分数从高到低取前 N）
	zKey := "demo:zset:rank"
	r.Del(bg, zKey)
	r.ZAdd(bg, zKey,
		redis.Z{Score: 100, Member: "Alice"},
		redis.Z{Score: 80, Member: "Bob"},
		redis.Z{Score: 95, Member: "Carol"},
		redis.Z{Score: 60, Member: "Dave"},
	)
	top3, _ := r.ZRevRangeWithScores(bg, zKey, 0, 2).Result()
	fmt.Println("[ZSet] 排行榜 Top3：")
	for i, it := range top3 {
		fmt.Printf("    %d: %s score=%.0f\n", i+1, it.Member, it.Score)
	}
}

// ============================================================
// 6. Pipeline（批量发送，减少 RTT）
// ============================================================
func demoPipeline(r *redis.Client) {
	keyPrefix := "demo:pipe:i"
	pipe := r.Pipeline()
	for i := 0; i < 100; i++ {
		pipe.Set(bg, fmt.Sprintf("%s:%d", keyPrefix, i), i, time.Minute)
		pipe.Expire(bg, fmt.Sprintf("%s:%d", keyPrefix, i), time.Hour)
	}
	// 一次性发送所有命令
	_, err := pipe.Exec(bg)
	if err != nil {
		log.Fatal("Pipeline Exec 失败：", err)
	}
	v, _ := r.Get(bg, fmt.Sprintf("%s:%d", keyPrefix, 42)).Int()
	fmt.Printf("[Pipeline] 批量写入 100 条，第 42 条值 = %d（预期 42）\n", v)
}

// ============================================================
// 7. Lua 脚本：扣库存（秒杀场景，原子操作）
// ============================================================
// Lua 脚本返回：0 成功；1 库存不足
var stockScript = redis.NewScript(`
local stockKey  = KEYS[1]
local deductKey = KEYS[2]
local userId    = ARGV[1]
local n         = tonumber(ARGV[2])

-- 幂等：同一个用户重复提交直接成功
local already = redis.call("SISMEMBER", deductKey, userId)
if already == 1 then return 0 end

local stock = tonumber(redis.call("GET", stockKey) or "0")
if stock < n then return 1 end

redis.call("DECRBY", stockKey, n)
redis.call("SADD", deductKey, userId)
return 0
`)

func demoLuaSeckill(r *redis.Client) {
	stockKey := "demo:lua:stock"
	boughtKey := "demo:lua:bought"

	// 初始化库存 10 件
	r.Set(bg, stockKey, 10, 0)
	r.Del(bg, boughtKey)

	// 模拟 20 个用户抢
	success := 0
	for uid := 1; uid <= 20; uid++ {
		ret, err := stockScript.Run(bg, r,
			[]string{stockKey, boughtKey}, // KEYS[]
			fmt.Sprintf("u%d", uid), 1).Int64() // ARGV[]
		if err != nil {
			log.Fatal("脚本执行失败：", err)
		}
		if ret == 0 {
			success++
		}
	}
	left, _ := r.Get(bg, stockKey).Int()
	fmt.Printf("[Lua 秒杀] 20 人抢 10 件，成功 %d 人，剩余库存 = %d（预期 = 0）\n", success, left)
}

// ============================================================
// 8. Pub/Sub 发布订阅（消息广播 / 实时通知）
// ============================================================
func demoPubSub(r *redis.Client) {
	ch := "demo:channel:news"

	// 订阅端（通常是另一个进程，这里用 goroutine 模拟）
	pubsub := r.Subscribe(bg, ch)
	defer pubsub.Close()

	go func() {
		time.Sleep(100 * time.Millisecond)
		msgs := []string{"news-A", "news-B", "news-C"}
		for _, m := range msgs {
			r.Publish(bg, ch, m)
		}
	}()

	chMsg := pubsub.Channel()
	fmt.Println("[Pub/Sub] 订阅等待 3 条消息：")
	count := 0
	for msg := range chMsg {
		fmt.Printf("    收到：%s\n", msg.Payload)
		count++
		if count >= 3 {
			break
		}
	}
}

// ============================================================
// main
// ============================================================
func main() {
	r := NewSimpleClient()
	defer r.Close()

	// 连通性检查：先 Ping 一下
	ctx, cancel := context.WithTimeout(bg, 2*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		fmt.Println("[提示] 无法连接 Redis（预计 127.0.0.1:6379）")
		fmt.Println("       启动 Redis 最简单方法（需要 Docker）：")
		fmt.Println("          docker run -d --name redis -p 6379:6379 redis:7")
		fmt.Println("")
		fmt.Println("       无 Redis 时，代码结构 + 注释已覆盖全部用法。")
		fmt.Println("       以下示例会跳过真实执行。")
		return
	}

	// 清理上次残留 key（演示用）
	r.Del(bg, "demo:string:counter", "demo:string:a", "demo:string:b", "demo:string:c", "lock:demo")
	r.Del(bg, "demo:hash:user:1001")
	r.Del(bg, "demo:list:queue")
	r.Del(bg, "demo:set:tag", "demo:zset:rank")
	r.Del(bg, "demo:pipe:i*")
	r.Del(bg, "demo:lua:stock", "demo:lua:bought")

	fmt.Println("===== 2. String ===== ")
	demoString(r)

	fmt.Println("\n===== 3. Hash ===== ")
	demoHash(r)

	fmt.Println("\n===== 4. List ===== ")
	demoList(r)

	fmt.Println("\n===== 5. Set + ZSet ===== ")
	demoSetAndZSet(r)

	fmt.Println("\n===== 6. Pipeline ===== ")
	demoPipeline(r)

	fmt.Println("\n===== 7. Lua 秒杀 ===== ")
	demoLuaSeckill(r)

	fmt.Println("\n===== 8. Pub/Sub ===== ")
	demoPubSub(r)

	fmt.Println("\n===> 所有 go-redis 示例完成 ✅")
}
