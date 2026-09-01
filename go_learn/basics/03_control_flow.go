// ============================================================
// 03_control_flow.go  控制流（if / for / switch / range）
// ------------------------------------------------------------
// 核心知识点：
//   1) if 语句（可带初始化语句）
//   2) for 的四种形态（C 风格 / while / 死循环 / range）
//   3) switch 的灵活用法（无需 break / case 多值 / 表达式 switch）
//   4) continue / break / label 跳出多层循环
//   运行：go run 03_control_flow.go
// ============================================================

package main

import "fmt"

func main() {
	// ------------------------------------------------------------
	// 1. if / else
	// ------------------------------------------------------------
	score := 85

	if score >= 90 {
		fmt.Println("优秀")
	} else if score >= 75 {
		fmt.Println("良好") // 会走到这里
	} else {
		fmt.Println("加油")
	}

	// 1.1 if 可带初始化语句（很常用，避免变量泄露）
	// 语法：if 初始化; 条件 { ... }
	if name := "Alice"; len(name) > 3 {
		fmt.Println("名字较长：", name)
	}
	// 这里访问不到 name（作用域仅在 if/else 内）

	// 典型用法：调用返回 (T, error) 的函数
	if err := someFunction(); err != nil {
		fmt.Println("有错误：", err)
	} else {
		fmt.Println("执行成功")
	}

	// ------------------------------------------------------------
	// 2. for：Go 只有 for 一个循环关键字，没有 while / do-while
	// ------------------------------------------------------------

	// 2.1 C 风格三段式
	for i := 0; i < 3; i++ {
		fmt.Println("三段式 i =", i)
	}

	// 2.2 等价 while 的写法：只留条件
	count := 0
	for count < 3 {
		fmt.Println("while 风格 count =", count)
		count++
	}

	// 2.3 死循环：什么都不写（等价 while(true)）
	loopCount := 0
	for {
		loopCount++
		if loopCount >= 2 {
			break
		}
		fmt.Println("死循环 style，第", loopCount, "次")
	}

	// 2.4 for-range 遍历（数组/切片/Map/字符串/通道都能用）
	arr := []int{10, 20, 30} // 切片类型，见 06_slices_maps.go
	for idx, val := range arr {
		fmt.Printf("切片：索引%d = %d\n", idx, val)
	}

	// 忽略索引
	for _, val := range arr {
		fmt.Printf("切片（忽略索引）：%d\n", val)
	}

	// 字符串 range 会按 rune（Unicode 字符）切，见 02_types.go
	s := "Hi中文"
	for i, ch := range s {
		fmt.Printf("[%d]%c ", i, ch)
	}
	fmt.Println()

	// ------------------------------------------------------------
	// 3. switch：非常灵活
	// ------------------------------------------------------------

	// 3.1 普通 switch（默认每个 case 有 break，不需要写 break）
	day := 2
	switch day {
	case 1:
		fmt.Println("周一")
	case 2:
		fmt.Println("周二") // 命中
	case 3, 4, 5: // case 可多值，逗号分隔
		fmt.Println("工作日")
	default:
		fmt.Println("周末")
	}

	// 3.2 初始化语句 + switch 变量
	switch level := scoreToLevel(score); level {
	case "A":
		fmt.Println("A 级")
	case "B":
		fmt.Println("B 级")
	default:
		fmt.Println("等级：", level)
	}

	// 3.3 表达式 switch（类似 if-else if 链）
	temp := 35
	switch {
	case temp < 0:
		fmt.Println("结冰")
	case temp >= 0 && temp < 20:
		fmt.Println("凉爽")
	case temp >= 30:
		fmt.Println("很热") // 命中
	default:
		fmt.Println("舒适")
	}

	// 3.4 fallthrough：显式穿透到下一个 case（几乎不用，Go 默认不穿透）
	switch 2 {
	case 1:
		fmt.Println("case 1")
	case 2:
		fmt.Println("case 2")
		fallthrough // 会继续执行 case 3 的代码块（不再判断 case 条件）
	case 3:
		fmt.Println("case 3（由 fallthrough 进入）")
	}

	// ------------------------------------------------------------
	// 4. label 跳出多层循环（Java 中也有类似 label）
	// ------------------------------------------------------------
OuterLoop:
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			fmt.Printf("(%d,%d) ", i, j)
			if i == 1 && j == 1 {
				break OuterLoop // 直接跳出最外层循环
			}
		}
	}
	fmt.Println("跳出完成")
}

// ------------------------------------------------------------
// 一些辅助函数，帮助示例
// ------------------------------------------------------------

// 返回 nil 表示无错误（实际项目看 09_errors.go）
func someFunction() error {
	return nil
}

func scoreToLevel(score int) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 75:
		return "B"
	default:
		return "C"
	}
}
