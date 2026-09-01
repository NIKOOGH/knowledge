// ============================================================
// 08_interfaces.go  接口（多态 / 抽象）
// ------------------------------------------------------------
// 核心知识点：
//   1) 接口定义：只写方法签名
//   2) 隐式实现：不需要 implements 关键字
//   3) 多态：接口变量可以存任意实现类型的值
//   4) 空接口 interface{} = any
//   5) 类型断言：.(T)  /  type switch
//   6) 接口组合（Go 常见模式）
//   运行：go run 08_interfaces.go
// ============================================================

package main

import (
	"fmt"
	"strings"
)

// ============================================================
// 1. 接口定义：只写方法签名，无实现
// ============================================================

// Speaker 能"说话"的对象就实现了这个接口
type Speaker interface {
	Speak() string // 方法签名：无参，返回 string
}

// Shouter 能"喊"的对象
type Shouter interface {
	Shout(volume int) string
}

// ============================================================
// 2. 隐式实现（无 implements）
// ============================================================

// Dog 只要有 Speak() string 方法，就自动实现了 Speaker 接口
// 不需要写 type Dog struct implements Speaker ...
type Dog struct{ Name string }

func (d Dog) Speak() string {
	return "Woof! I'm " + d.Name
}

type Cat struct{ Name string }

func (c Cat) Speak() string {
	return "Meow! I'm " + c.Name
}

// Person 同时实现 Speaker + Shouter 两个接口
type Person struct{ Name string }

func (p Person) Speak() string {
	return "Hello, I'm " + p.Name
}
func (p Person) Shout(volume int) string {
	s := p.Speak()
	if volume > 5 {
		s = strings.ToUpper(s) + "!!!"
	}
	return s
}

// ============================================================
// 3. 多态：同一接口变量，不同实现，行为不同
// ============================================================
func letThemSpeak(speakers []Speaker) {
	for _, s := range speakers {
		fmt.Println("  " + s.Speak())
	}
}

// ============================================================
// 4. 接口组合（Go 常见写法，类比 Java 接口多继承）
// ============================================================
// SpeakerAndShouter 拥有两个接口的方法（组合而不是继承）
type SpeakerAndShouter interface {
	Speaker
	Shouter
}

// Person 自动实现 SpeakerAndShouter，因为它同时有 Speak() + Shout()

// ============================================================
// 5. 空接口 interface{}（新写法可用 any 关键字，Go 1.18+）
// ============================================================
// 空接口没有任何方法 → 所有类型都隐式实现了它 → 类似 Java Object
func printAnything(v interface{}) {
	fmt.Printf("类型=%T  值=%v\n", v, v)
}

// ============================================================
// 6. 类型断言：从接口变量取回具体类型
// ============================================================
func typeAssertion(sp Speaker) {
	// 6.1 直接断言（失败会 panic）
	// p := sp.(Person)  // 如果 sp 底层是 Dog → panic

	// 6.2 安全断言（ok 判断，最常用）
	p, ok := sp.(Person)
	if ok {
		fmt.Println("这是 Person：Name =", p.Name)
		return
	}
	if d, ok := sp.(Dog); ok {
		fmt.Println("这是 Dog：Name =", d.Name)
		return
	}
	fmt.Println("未知实现类型")
}

// 6.3 type switch：多分支判断具体类型
func describe(v any) { // any 等价 interface{}
	switch x := v.(type) {
	case int:
		fmt.Println("整数：", x)
	case string:
		fmt.Println("字符串，长度", len(x), ":", x)
	case Dog:
		fmt.Println("狗：", x.Name)
	case Speaker: // 甚至可以基于接口匹配
		fmt.Println("是个 Speaker，说:", x.Speak())
	default:
		fmt.Printf("未知类型：%T=%v\n", v, v)
	}
}

// ============================================================
// 主函数
// ============================================================
func main() {
	fmt.Println("===== 1. 隐式实现 + 多态 =====")
	speakers := []Speaker{
		Dog{Name: "旺财"},
		Cat{Name: "咪咪"},
		Person{Name: "小明"},
	}
	letThemSpeak(speakers)

	fmt.Println("\n===== 2. 接口组合 =====")
	var ss SpeakerAndShouter = Person{Name: "小红"}
	fmt.Println("Speak =", ss.Speak())
	fmt.Println("Shout =", ss.Shout(10))

	fmt.Println("\n===== 3. 空接口（any）=====")
	printAnything(42)
	printAnything("hello")
	printAnything(Dog{Name: "汪汪"})

	fmt.Println("\n===== 4. 类型断言 =====")
	typeAssertion(Person{Name: "张三"})
	typeAssertion(Dog{Name: "大黄"})

	fmt.Println("\n===== 5. type switch =====")
	describe(123)
	describe("你好")
	describe(Dog{Name: "毛球"})
	describe(true)
	describe(Cat{Name: "咪咪"}) // 走 Speaker 分支
}
