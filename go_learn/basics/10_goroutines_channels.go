// ============================================================
// 10_goroutines_channels.go  并发编程
// ------------------------------------------------------------
// 核心知识点：
//   1) goroutine 轻量级协程（go func()）
//   2) Channel：chan T / make(chan T)
//   3) 有缓冲 / 无缓冲通道
//   4) select 多路复用
//   5) sync.WaitGroup：等一组 goroutine 结束
//   6) sync.Mutex / RWMutex：临界区
//   运行：go run 10_goroutines_channels.go
// ============================================================

package main

import (
	"fmt"
	"sync"
	"time"
)

// ============================================================
// 1. goroutine 启动：go 关键字
// ============================================================
func hello() {
	fmt.Println("Hello from goroutine")
}

func helloGoroutine() {
	// 启动一个子协程，不会阻塞主协程
	go hello()

	// 匿名函数也可以 go 启动
	go func(msg string) {
		fmt.Println("匿名函数：", msg)
	}("hello world")

	// 给点时间让 goroutine 跑完（仅演示，正式代码用 WaitGroup）
	time.Sleep(50 * time.Millisecond)
}

// ============================================================
// 2. Channel 通道：协程间通信（"通过通信共享内存"）
// ============================================================

// 2.1 无缓冲通道：发送 <- 接收 必须配对，缺一阻塞
func unbufferedChan() {
	ch := make(chan int) // 无缓冲通道：chan int
	// 等价 make(chan int, 0)

	go func() {
		fmt.Println("子协程：准备发送 100")
		ch <- 100 // <-- 如果没人读，就一直卡在这里
		fmt.Println("子协程：100 发送完成")
	}()

	time.Sleep(20 * time.Millisecond) // 让子协程先开始，观察阻塞现象
	fmt.Println("主协程：准备读取")
	val := <-ch
	fmt.Println("主协程：读到 val =", val)
	time.Sleep(10 * time.Millisecond)
}

// 2.2 有缓冲通道：容量 > 0，写满才阻塞
func bufferedChan() {
	ch := make(chan string, 2) // 缓冲大小 = 2

	ch <- "a"
	ch <- "b"
	fmt.Println("写入两个元素后，len(ch) =", len(ch), "/ cap =", cap(ch))
	// ch <- "c"  // 写第三个 → 阻塞

	fmt.Println("读：", <-ch)     // "a"
	fmt.Println("再读：", <-ch)   // "b"
}

// 2.3 关闭通道 + for-range
func closeAndRange() {
	ch := make(chan int, 5)
	go func() {
		for i := 1; i <= 5; i++ {
			ch <- i
		}
		close(ch) // 关闭：发送端负责 close
		fmt.Println("子协程：已 close(ch)")
	}()

	// for-range 会自动在通道关闭时退出（最优雅的读取方法）
	fmt.Print("for-range 读到：")
	for v := range ch {
		fmt.Print(v, " ")
	}
	fmt.Println()

	// 已关闭通道再读：返回零值 + ok=false
	x, ok := <-ch
	fmt.Printf("再读已关通道 x=%d ok=%v\n", x, ok)
}

// 2.4 chan<- 只写通道  <-chan 只读通道（用于函数参数约束写/读权限）
type OnlyWrite chan<- int
type OnlyRead <-chan int

// ============================================================
// 3. select：同时监听多个通道（类似 Unix select/epoll）
// ============================================================
func selectDemo() {
	ch1 := make(chan string)
	ch2 := make(chan string)

	go func() {
		time.Sleep(50 * time.Millisecond)
		ch1 <- "来自 ch1 的消息"
	}()
	go func() {
		time.Sleep(20 * time.Millisecond)
		ch2 <- "来自 ch2 的消息（更快）"
	}()

	// 最多等 200ms，谁先到就先处理谁
	timeout := time.After(200 * time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case m := <-ch1:
			fmt.Println("select 收到 ch1：", m)
		case m := <-ch2:
			fmt.Println("select 收到 ch2：", m)
		case <-timeout:
			fmt.Println("select 超时了")
			return
		}
	}

	// select default：非阻塞读写
	select {
	case v := <-ch1:
		fmt.Println("有数据：", v)
	default:
		fmt.Println("ch1 没数据，我先走了（default 分支）")
	}
}

// ============================================================
// 4. sync.WaitGroup：等一组 goroutine 全部结束
// ============================================================
func waitGroupDemo() {
	var wg sync.WaitGroup

	for i := 1; i <= 3; i++ {
		// 启动一个 goroutine 前 Add(1)
		wg.Add(1)
		go func(taskID int) {
			// 关键：wg.Done() 一定要执行到 → defer 保证
			defer wg.Done() // Done 等价 Add(-1)
			fmt.Printf("任务 %d：开始工作\n", taskID)
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("任务 %d：完成\n", taskID)
		}(i) // 把 i 作为参数传入，防止循环变量闭包陷阱
	}

	fmt.Println("主协程：等待所有任务...")
	wg.Wait() // 阻塞，直到 wg 计数器归 0
	fmt.Println("所有任务完成！")
}

// ============================================================
// 5. sync.Mutex：临界区锁
// ============================================================
type SafeCounter struct {
	mu    sync.Mutex
	value int
}

func (c *SafeCounter) Inc() {
	c.mu.Lock()         // 加锁
	defer c.mu.Unlock() // 离开函数自动解锁
	c.value++
}

func (c *SafeCounter) Get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

func mutexDemo() {
	var wg sync.WaitGroup
	counter := &SafeCounter{}

	N := 1000
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			counter.Inc()
		}()
	}
	wg.Wait()
	fmt.Printf("并发 %d 次累加：counter = %d（应该刚好 %d）\n", N, counter.Get(), N)
	// 不加锁的话会远小于 1000（竞态条件）
}

// RWMutex：多读单写（读多读少场景使用）
func rwMutexDemo() {
	var rw sync.RWMutex
	var value int

	write := func() {
		rw.Lock() // 写锁：独占
		defer rw.Unlock()
		value++
		time.Sleep(1 * time.Millisecond)
		fmt.Println("写：value =", value)
	}
	read := func(id int) {
		rw.RLock() // 读锁：共享，多个 goroutine 同时读不会卡
		defer rw.RUnlock()
		time.Sleep(1 * time.Millisecond)
		fmt.Printf("读%d：value = %d\n", id, value)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); write() }()
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); read(i) }(i)
	}
	wg.Wait()
}

// ============================================================
// 主函数
// ============================================================
func main() {
	fmt.Println("===== 1. goroutine 启动 =====")
	helloGoroutine()

	fmt.Println("\n===== 2.1 无缓冲通道 =====")
	unbufferedChan()

	fmt.Println("\n===== 2.2 有缓冲通道 =====")
	bufferedChan()

	fmt.Println("\n===== 2.3 close + for-range =====")
	closeAndRange()

	fmt.Println("\n===== 3. select 多路复用 =====")
	selectDemo()

	fmt.Println("\n===== 4. sync.WaitGroup =====")
	waitGroupDemo()

	fmt.Println("\n===== 5.1 sync.Mutex =====")
	mutexDemo()

	fmt.Println("\n===== 5.2 sync.RWMutex =====")
	rwMutexDemo()
}
