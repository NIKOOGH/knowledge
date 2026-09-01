// ============================================================
// 02_types.go  基本数据类型
// ------------------------------------------------------------
// 核心知识点：
//   1) 数值类型（int/float/uint）
//   2) bool 与 string
//   3) rune / byte
//   4) 类型转换（无隐式转换，必须显式 T(v)）
//   5) 别名类型与 type 关键字
//   运行：go run 02_types.go
// ============================================================

package main

import (
	"fmt"
	"unsafe"
)

func main() {
	// ------------------------------------------------------------
	// 1. 整数类型
	// ------------------------------------------------------------
	// 有符号：int8  int16  int32  int64  int（平台相关，32/64 位）
	// 无符号：uint8 uint16 uint32 uint64 uint / uintptr（存指针）
	var i8 int8 = -128
	var i64 int64 = 9223372036854775807
	var u8 uint8 = 255
	var natural int = 42

	fmt.Printf("整数：int8=%d int64=%d uint8=%d int=%d\n", i8, i64, u8, natural)
	fmt.Printf("平台 int 占字节数：%d (64位机器 = 8，32位机器 = 4)\n", unsafe.Sizeof(natural))

	// 常见用法：绝大多数情况直接写 int 即可，性能和 int64 几乎无差
	// 除非明确要做位运算或跨平台兼容，才写 int32/int64

	// ------------------------------------------------------------
	// 2. 浮点数
	// ------------------------------------------------------------
	// float32 约 7 位有效数字；float64 约 15 位（默认 float64）
	var f32 float32 = 3.14
	f64 := 2.718281828459045
	fmt.Printf("浮点：f32=%.3f f64=%.10f\n", f32, f64)

	// ------------------------------------------------------------
	// 3. bool
	// ------------------------------------------------------------
	var yes bool = true
	no := false
	fmt.Println("布尔：yes=", yes, "no=", no)

	// ! && || 逻辑运算，与 Java 相同
	fmt.Println("!no =", !no, "yes && no =", yes && no, "yes || no =", yes || no)

	// ------------------------------------------------------------
	// 4. 字符串 string
	// ------------------------------------------------------------
	// Go 的字符串底层是 UTF-8 字节数组，不可变（immutable）
	greeting := "Hello, 你好"
	fmt.Printf("字符串=%q 长度=%d 字节\n", greeting, len(greeting))
	// 注意 len 返回字节数，不是字符数 → "你好" 每个汉字 3 字节，总长 = 7+6=13

	// 字符串拼接
	who := "World"
	full := greeting + " " + who + "!"
	fmt.Println("拼接：", full)

	// 字符串按索引取的是 byte（uint8），不是字符
	fmt.Printf("greeting[0]=%d 对应字符=%c\n", greeting[0], greeting[0])

	// 遍历字符（for-range 方式会自动按 rune 解码 UTF-8）
	fmt.Print("逐字符遍历：")
	for idx, ch := range greeting {
		fmt.Printf("[%d]%c ", idx, ch)
	}
	fmt.Println()

	// 多行原始字符串：反引号 ` ... `，不转义、保留换行
	raw := `
SELECT id, name
FROM user
WHERE id = 123
`
	fmt.Println("原始字符串：", raw)

	// ------------------------------------------------------------
	// 5. byte 与 rune
	// ------------------------------------------------------------
	// byte = uint8 的别名，存 ASCII / 单字节字符
	var b byte = 'A'
	fmt.Printf("byte=%d 字符=%c\n", b, b)

	// rune = int32 的别名，存 Unicode 码点（一个汉字就是一个 rune）
	var r rune = '你'
	fmt.Printf("rune=%d 字符=%c\n", r, r)

	// ------------------------------------------------------------
	// 6. 类型转换（★ 必考点：Go 没有隐式类型转换）
	// ------------------------------------------------------------
	// Java 允许 int → long / byte → int 自动提升；Go 一律手动 T(v)
	var x int = 10
	var y float64 = float64(x)   // int → float64，必须显式
	var z int8 = int8(x)         // int → int8，必须显式
	fmt.Println("转换：x=", x, "y=", y, "z=", z)

	// 例子：算术运算必须类型一致
	// var bad = x + y  // 编译错误：int + float64 不匹配
	good := float64(x) + y
	fmt.Println("类型一致运算：good=", good)

	// 字符串 ↔ 字节切片（[]byte）常见转换：
	msg := "hello"
	msgBytes := []byte(msg)        // string → []byte
	back := string(msgBytes)       // []byte → string
	fmt.Printf("字符串→字节=%v，字节→字符串=%s\n", msgBytes, back)

	// ------------------------------------------------------------
	// 7. type 自定义类型（不是别名，是新类型）
	// ------------------------------------------------------------
	type UserID int64        // 把 UserID 定义为新类型，底层是 int64
	type Money int64         // 避免把 UserID 与 Money 乱传

	var uid UserID = 1001
	var price Money = 9999
	// var sum Money = price + int64(uid)  // 编译错误：类型不匹配
	var sum int64 = int64(price) + int64(uid) // 必须显式转
	fmt.Println("自定义类型：uid=", uid, "price=", price, "sum=", sum)
}
