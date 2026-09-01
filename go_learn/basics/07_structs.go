// ============================================================
// 07_structs.go  结构体 + 方法（Go 的"类"替代方案）
// ------------------------------------------------------------
// 核心知识点：
//   1) 结构体定义 + 字段 + 字段标签（struct tag）
//   2) 值接收者 vs 指针接收者（最易混淆）
//   3) 结构体嵌入 = Go 式的"继承"（组合优于继承）
//   4) 构造函数约定（没有 constructor，用 NewXxx 工厂函数）
//   5) 结构体相等性比较
//   运行：go run 07_structs.go
// ============================================================

package main

import (
	"encoding/json"
	"fmt"
)

// ============================================================
// 1. 结构体定义
// ============================================================

// Person 定义一个"人"结构体
type Person struct {
	Name   string // 字段名大写 = public，包外可访问
	Age    int
	Email  string
	gender string // 小写 = private，仅本包可访问
}

// 带 JSON 标签（最常用的 struct tag，序列化/反序列化控制）
type Product struct {
	// json:"name"     → JSON 的 key 名
	// omitempty       → 该字段为零值（空/0/false/nil）时不输出 JSON
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Price int    `json:"price,omitempty"`
	Desc  string `json:"description,omitempty"`
}

// ============================================================
// 2. 构造函数约定：NewXxx 返回 *T（Go 没有 constructor 关键字）
// ============================================================

// NewPerson 构造函数（约定：错误或返回值 / 工厂模式）
// 这里没有错误就直接返回 *Person
func NewPerson(name string, age int, email string) *Person {
	// &Person{...} 在 Go 里是合法的：编译器会把结构体逃逸到堆上，返回指针不会 dangling
	return &Person{
		Name:  name,
		Age:   age,
		Email: email,
	}
}

// ============================================================
// 3. 方法（为类型绑定函数，Go 替代 class 的关键）
// ============================================================
// 语法：func (接收者 类型) 方法名(参数) 返回值 { ... }
// 接收者类似 Java 的 this，但必须显式命名（通常用类型首字母缩写）

// 3.1 值接收者：不会修改原对象，适用于只读操作
func (p Person) Greet() string {
	return "Hello, I'm " + p.Name + ", age " + fmt.Sprintf("%d", p.Age)
}

// 3.2 指针接收者：可以修改原对象，也避免大结构体拷贝（常用）
func (p *Person) Birthday() {
	p.Age++ // 修改指针指向的原对象
}

// ★ 最佳实践：方法的接收者要么全是值，要么全是指针，不要混搭（影响接口实现）
// 一般来说：修改原对象 / 结构体很大 → 用指针接收者

// ============================================================
// 4. 结构体嵌入（Go 的"继承" = 组合）
// ============================================================
// Java: class Employee extends Person { String company; }
// Go:   在 Employee 里嵌一个 Person（不需要写名字），叫嵌入字段 / 匿名字段

type Employee struct {
	Person         // <-- 嵌入字段（名字=类型名 Person）
	Company string
	Level   int
}

// Employee 可以直接访问 Person 的字段，也自动继承了 Person 的方法
// 相当于 Java 里 Employee.this.Name，但 Go 用的是嵌入语法

// 可以给 Employee 覆盖同名方法
func (e *Employee) Greet() string {
	// 通过 e.Person.Greet() 调父类方法
	return e.Person.Greet() + "，我在 " + e.Company + " 工作"
}

// ============================================================
// 主函数
// ============================================================
func main() {
	fmt.Println("===== 1. 结构体创建 =====")
	// 1.1 字段顺序初始化（不推荐：字段调整就会错）
	p1 := Person{"Alice", 25, "a@x.com", "F"}
	fmt.Println("p1 =", p1)

	// 1.2 键值对初始化（推荐）
	p2 := Person{Name: "Bob", Age: 30}
	fmt.Println("p2 =", p2) // Email="" gender=""

	// 1.3 通过构造函数
	p3 := NewPerson("Carol", 28, "c@x.com")
	fmt.Printf("p3 = %+v\n", *p3) // %+v 输出字段名

	fmt.Println("\n===== 2. 方法调用 =====")
	fmt.Println(p3.Greet())
	p3.Birthday() // 指针接收者，修改的是 p3 本身
	fmt.Println("过完生日：", p3.Greet()) // 28 → 29

	fmt.Println("\n===== 3. 结构体嵌入（继承）=====")
	// 3.1 创建嵌入结构体
	e := Employee{
		Person: Person{
			Name: "Dave",
			Age:  33,
		},
		Company: "ACME",
		Level:   5,
	}
	// 3.2 直接访问嵌入字段：可以写 e.Name，也可以写 e.Person.Name
	fmt.Println("e.Name =", e.Name, " / e.Person.Name =", e.Person.Name)
	fmt.Println("e.Company =", e.Company)
	// 3.3 调用方法（覆盖的 Greet）
	fmt.Println(e.Greet())

	fmt.Println("\n===== 4. Struct Tag + JSON =====")
	prod := Product{
		ID:    1001,
		Name:  "iPhone",
		Price: 0,   // 零值 → omitempty 不输出
		Desc:  "",  // 空字符串 → omitempty 不输出
	}
	jsonBytes, _ := json.MarshalIndent(prod, "", "  ")
	fmt.Println("序列化 JSON：")
	fmt.Println(string(jsonBytes))

	fmt.Println("\n===== 5. 结构体相等性比较 =====")
	// 所有字段都可比较 → 结构体可比较
	a := Person{Name: "Tom", Age: 18}
	b := Person{Name: "Tom", Age: 18}
	c := Person{Name: "Jerry", Age: 18}
	fmt.Println("a==b？", a == b) // true
	fmt.Println("a==c？", a == c) // false
	// 含有 slice/map 的结构体 == 不可用，会编译报错
}
