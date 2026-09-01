// ============================================================
// 04_functions.go  函数
// ------------------------------------------------------------
// 核心知识点：
//   1) 基本函数定义与参数传递
//   2) 多返回值（Go 最常用的 (T, error) 模式）
//   3) 命名返回值
//   4) 可变参数
//   5) 匿名函数 / 闭包 / 函数作为参数
//   6) defer 延迟执行
//   运行：go run 04_functions.go
// ============================================================

package main

import (
	"errors"
	"fmt"
)

// ------------------------------------------------------------
// 1. 基本函数
// ------------------------------------------------------------
// 类型写在名字后面：func 函数名(参数) 返回值 { ... }
func add(a int, b int) int {
	return a + b
}

// 同类型参数可以合并类型声明
func multiply(a, b int) int {
	return a * b
}

// ------------------------------------------------------------
// 2. 多返回值（★ 最常用 (T, error) 模式）
// ------------------------------------------------------------
// Go 没有异常机制，用 error 返回值表示失败；成功返回 nil
func divide(a, b float64) (float64, error) {
	if b == 0 {
		// errors.New 创建一个简单错误
		return 0, errors.New("除数不能为零")
	}
	return a / b, nil
}

// 任意多返回值，比如取两个数的商和余数
func divmod(a, b int) (int, int) {
	return a / b, a % b
}

// ------------------------------------------------------------
// 3. 命名返回值
// ------------------------------------------------------------
// 返回值变量先声明好，函数里直接写 return（不带参数）→ 自动把变量带回去
func rectangle(width, height int) (area int, perimeter int) {
	area = width * height
	perimeter = 2 * (width + height)
	return // 等价于 return area, perimeter（"裸返回"）
}

// 实际项目中，简短函数用命名返回值增加可读性；长函数不建议裸返回（难找返回值）

// ------------------------------------------------------------
// 4. 可变参数：最后一个参数前加 ...
// ------------------------------------------------------------
func sumAll(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

// 把切片打散传入：nums...
func passSlice() {
	slice := []int{1, 2, 3, 4}
	total := sumAll(slice...) // 等价 sumAll(1,2,3,4)
	fmt.Println("切片打散求和：", total)
}

// ------------------------------------------------------------
// 5. 匿名函数 + 闭包
// ------------------------------------------------------------
func closures() {
	// 5.1 把匿名函数赋值给变量
	fn := func(a, b int) int { return a + b }
	fmt.Println("匿名函数调用：", fn(1, 2))

	// 5.2 立即执行函数（IIFE）
	res := func(a, b int) int { return a - b }(10, 3)
	fmt.Println("立即执行函数：", res)

	// 5.3 闭包：内部函数引用外部变量
	//    这个 counter() 每次被调用，内部 cnt 会持续加 1
	counter := func() func() int {
		cnt := 0
		return func() int {
			cnt++
			return cnt
		}
	}()
	fmt.Println("闭包计数器：", counter(), counter(), counter()) // 1 2 3
}

// ------------------------------------------------------------
// 6. 函数作为参数（高阶函数 / 回调）
// ------------------------------------------------------------
// 类型别名 + 函数参数：见 08_interfaces.go 的组合式写法
type CalcFunc func(int, int) int

func calc(a, b int, fn CalcFunc) int {
	return fn(a, b)
}

// ------------------------------------------------------------
// 7. defer：延迟执行
// ------------------------------------------------------------
// 多个 defer 按 "后进先出" 顺序执行（类似栈）
// 典型用途：释放资源（关闭文件、解锁、关连接）
func deferDemo() {
	defer fmt.Println("defer A — 第 1 个注册，最后执行")
	defer fmt.Println("defer B — 第 2 个注册，倒数第 2 执行")
	defer fmt.Println("defer C — 第 3 个注册，最先执行")
	fmt.Println("正常逻辑执行中...")
	// return / panic 前，所有 defer 都会执行完
}

// 实际例子：关闭文件
//   f, err := os.Open("a.txt")
//   if err != nil { return err }
//   defer f.Close()    ← 函数结束时一定会自动关闭，不会漏写

// defer 参数是立刻求值的，不是调用时求值（陷阱点）
func deferArgEval() {
	i := 0
	defer fmt.Println("defer 里的 i =", i) // i=0 被捕获了
	i++
	fmt.Println("最后 i =", i) // 1
}

// ------------------------------------------------------------
// 主函数：调用上面所有例子
// ------------------------------------------------------------
func main() {
	fmt.Println("===== 1. 基本函数 =====")
	fmt.Println("add =", add(1, 2))
	fmt.Println("multiply =", multiply(2, 3, 4))

	fmt.Println("\n===== 2. 多返回值 =====")
	q, err := divide(10, 3)
	if err != nil {
		fmt.Println("错误：", err)
	} else {
		fmt.Println("10/3 =", q)
	}
	_, err = divide(1, 0) // 忽略商，只看错误
	fmt.Println("1/0 错误：", err)

	q2, r2 := divmod(10, 3)
	fmt.Printf("10/3 = 商 %d 余 %d\n", q2, r2)

	fmt.Println("\n===== 3. 命名返回值 =====")
	area, perimeter := rectangle(3, 4)
	fmt.Printf("3x4 矩形：面积=%d 周长=%d\n", area, perimeter)

	fmt.Println("\n===== 4. 可变参数 =====")
	fmt.Println("sumAll =", sumAll(1, 2, 3, 4, 5))
	passSlice()

	fmt.Println("\n===== 5. 匿名函数 / 闭包 =====")
	closures()

	fmt.Println("\n===== 6. 函数作为参数 =====")
	fmt.Println("calc(add) =", calc(10, 5, add))
	fmt.Println("calc(multiply) =", calc(10, 5, multiply))
	fmt.Println("calc(lambda) =", calc(10, 5, func(a, b int) int { return a - b }))

	fmt.Println("\n===== 7. defer =====")
	deferDemo()
	deferArgEval()
}
