// ============================================================
// 11_packages.go  包与模块（import / go.mod / 可见性）
// ------------------------------------------------------------
// 核心知识点：
//   1) GOPATH vs Go Modules（1.11+ 默认 go mod）
//   2) 新建一个模块：go mod init 模块路径
//   3) import：标准库 / 相对路径 / 绝对模块路径
//   4) 可见性规则：首字母大写 public，小写 private
//   5) init() 函数
//   6) 点导入 / 别名导入 / 空白导入
//
// 说明：本文件作为独立可运行示例不依赖任何非标准库文件，
//      实际创建新模块的命令和输出在注释里展示。
//   运行：go run 11_packages.go
// ============================================================

package main

import (
	f "fmt"
	"sort"
	"time"
)

// ============================================================
// import 的三种常见用法（上方 import 即实际演示）
// ============================================================
// 本文件使用：
//   import f "fmt"        —— 别名导入：把 fmt 简写成 f
//   import "sort"         —— 标准导入
//   import "time"         —— 标准导入
//
// 另外两种常用方式（在你的其他项目中按需 import）：
//   1) 空白导入（_）：只跑包的 init()，不使用其中的标识符
//      最常见用途：注册数据库驱动
//      import _ "github.com/go-sql-driver/mysql"
//   2) 点导入（.）：直接写包内的名字，省掉前缀（不推荐：名字容易冲突）
//      import . "fmt"   // 之后可以直接 Println(...)
// ============================================================

// ============================================================
// init() 函数：包被导入时自动执行（不需要显式调用）
// ============================================================
// 同一个包可以有多个 init()，按定义顺序依次执行
// 多个包的 init 顺序按 import 依赖树的 DFS 顺序

var InitSeq []string

func init() {
	InitSeq = append(InitSeq, "main.init(1)")
}
func init() {
	InitSeq = append(InitSeq, "main.init(2)")
}

// ============================================================
// 一、创建新模块的完整流程（★ 必学）
// ============================================================
// 假设要做一个 myapp 项目，用 Gin：
//
// $ mkdir myapp && cd myapp
// $ go mod init github.com/yourname/myapp
//     -> 生成 go.mod：
//       module github.com/yourname/myapp
//       go 1.22
//
// $ go get -u github.com/gin-gonic/gin
//     -> 自动修改 go.mod，新增 require 段
//     -> 生成 go.sum（依赖哈希校验）
//
// $ vim main.go
//     package main
//     import "github.com/gin-gonic/gin"
//     func main() {
//         r := gin.Default()
//         r.GET("/ping", func(c *gin.Context) {
//             c.JSON(200, gin.H{"msg": "pong"})
//         })
//         r.Run(":18080")
//     }
//
// $ go mod tidy      // 自动补齐缺失依赖、删除未用依赖（很常用）
// $ go run main.go
//
// 常见命令：
//   go mod init <模块名>        初始化 go.mod
//   go mod tidy                 整理依赖
//   go mod download             把依赖下载到本地缓存（$GOPATH/pkg/mod）
//   go mod vendor               把依赖拷贝到项目 vendor 目录（可离线构建）
//   go build -o myapp .         编译输出可执行文件 myapp
//
// GOPROXY（国内必备）：
//   Windows PowerShell：
//     $env:GOPROXY = "https://goproxy.cn,direct"
//   或全局：
//     go env -w GOPROXY=https://goproxy.cn,direct
//   其它镜像：https://goproxy.io / https://mirrors.aliyun.com/goproxy/
//
// 模块版本（语义化）：
//   go get github.com/gin-gonic/gin@v1.9.1   指定版本
//   go get github.com/gin-gonic/gin@latest    最新稳定版
//   go get github.com/gin-gonic/gin@master    分支/commit
// ============================================================

// ============================================================
// 二、可见性规则（一句话：首字母大写 public，小写 private）
// ============================================================
// Go 没有 public / private / protected 关键字
//   type Foo struct {}        // Foo 可被其他包引用
//   type bar struct {}        // bar 只能当前包用
//   func Hello() {}           // Hello 对外可见
//   func hello() {}           // hello 仅当前包
//   type S struct { A int; b int }  // A 对外，b 仅当前包
//
// 所有导出的东西（大写开头）都该写注释：Go 静态检查工具会报错

// ============================================================
// 三、目录结构约定（Go 社区约定，不是强制）
// ============================================================
// 典型中大型项目（Java 开发者最容易理解的分层版）：
//
// myapp/
// ├── cmd/                 可执行入口（一个入口一个子目录）
// │   └── myapp/
// │       └── main.go      package main
// ├── internal/            内部代码（其他项目不能 import，_import_ 语法限制）
// │   ├── controller/       路由 / HTTP Handler
// │   ├── service/          业务逻辑
// │   ├── repository/       DB / 缓存访问
// │   ├── model/            结构体 / DTO
// │   └── config/           配置加载
// ├── pkg/                  对外可复用的库（其他项目可以 import）
// ├── api/                  proto / OpenAPI / graphql schema
// ├── configs/              配置文件（yaml/toml）
// ├── scripts/              CI/CD、部署脚本
// ├── go.mod
// ├── go.sum
// ├── Makefile             可选：make build / make run
// └── README.md
//
// 小程序简单项目就一个 main.go + go.mod，也完全没问题。
// ============================================================

// ============================================================
// 四、实际代码小 demo：验证 import 和可见性
// ============================================================

// 导出类型：大写首字母，对外可见
type Person struct {
	Name string // 导出字段
	age  int    // 未导出字段（其他包访问不到）
}

// 导出方法：导出类型上的导出方法，对外可见
func (p *Person) Birthday() {
	p.age++
}

// 未导出函数：仅同包内可用
func calcAge(year int) int {
	return time.Now().Year() - year
}

func main() {
	f.Println("init 函数执行顺序：", InitSeq)

	nums := []int{3, 1, 4, 1, 5, 9, 2, 6}
	sort.Ints(nums)          // 标准库 sort 包（import）
	f.Println("排序结果：", nums) // 别名 f 替代 fmt

	p := Person{Name: "Alice"}
	p.Birthday()
	// p.age 可以访问（同包）
	f.Printf("Person: %+v age=%d\n", p, p.age)

	// 计算年龄（调用未导出函数，同包 OK）
	yob := 1990
	age := calcAge(yob)
	f.Printf("出生年份 %d 的人现在约 %d 岁\n", yob, age)

	f.Println("\n=== 运行结束 ===")
	f.Println("接下来可以把 frameworks 目录下的 Gin/GORM/go-redis 依次跑一遍：")
	f.Println("  cd frameworks/01_gin && go mod tidy && go run main.go")
}
