# Go 语言快速上手学习手册（go_learn）

> 面向零基础 / 转语言开发者：**所有基础语法 + 三大常用框架**全覆盖。
> 每个文件独立可运行，注释详尽。读完 → 改完 → 跑通，即可具备 Go 生产力。
>
> 运行环境：Go 1.22+（`d:\文件\test\tools\go` 已内置 1.22.5）
> 推荐 IDE：Trae / VS Code（Go 扩展）
> 文档位置：`d:\文件\test\go_learn\`

---

## 目录

### 一、基础语法（basics/）

每个文件 `go run <文件名>` 即可运行。

| 文件 | 主题 | 知识点 |
|------|------|--------|
| [01_variables.go](basics/01_variables.go) | 变量与常量 | var / := / 类型推断 / 常量 / iota |
| [02_types.go](basics/02_types.go) | 基本类型 | int/float/bool/string/rune/byte/类型转换 |
| [03_control_flow.go](basics/03_control_flow.go) | 控制流 | if/else / for / switch / range |
| [04_functions.go](basics/04_functions.go) | 函数 | 多返回值 / 命名返回值 / 可变参数 / 匿名函数 / defer |
| [05_pointers.go](basics/05_pointers.go) | 指针 | *T / & / nil / 值传递 vs 引用传递 |
| [06_slices_maps.go](basics/06_slices_maps.go) | 切片与 Map | make / append / 扩容 / 遍历 / 常用技巧 |
| [07_structs.go](basics/07_structs.go) | 结构体 | 定义 / 字段标签 / 方法 / 组合（继承替代） |
| [08_interfaces.go](basics/08_interfaces.go) | 接口 | 隐式实现 / 多态 / 空接口 / type assertion |
| [09_errors.go](basics/09_errors.go) | 错误处理 | errors.New / fmt.Errorf / 自定义错误 / panic&recover |
| [10_goroutines_channels.go](basics/10_goroutines_channels.go) | 并发 | go 关键字 / channel / select / sync.WaitGroup / Mutex |
| [11_packages.go](basics/11_packages.go) | 包与模块 | GOPATH / go.mod / import / 可见性规则 |

### 二、常用框架（frameworks/）

每个子目录都是独立模块，包含 `go.mod`、`main.go` 和使用说明。

| 目录 | 框架 | 对标 Java 生态 |
|------|------|--------------|
| [01_gin](frameworks/01_gin) | Gin v1.9 (Web 框架) | Spring MVC / Spring Boot Web |
| [02_gorm](frameworks/02_gorm) | GORM v1.25 (ORM) | MyBatis / Spring Data JPA |
| [03_goredis](frameworks/03_goredis) | go-redis v9 (Redis 客户端) | Jedis / Lettuce / RedisTemplate |
| [04_layered](frameworks/04_layered) | 分层工程示例（Controller→Service→Repository） | Spring Boot 标准三层架构 |

### 三、与 Java 的心智映射

| Java 概念 | Go 等价物 | 说明 |
|-----------|-----------|------|
| class | struct + 方法（接收者） | Go 无 class，用结构体+方法+组合实现面向对象 |
| extends 继承 | 结构体嵌入（组合） | Go 用组合代替继承，更灵活 |
| interface 接口 | interface | Go 接口隐式实现，无需 `implements` 关键字 |
| try-catch | error 返回值 + panic/recover | 通常用 error 处理可预期错误，panic 只用于不可恢复 |
| Thread + Runnable | go func() + Channel |  goroutine 比线程轻量，Channel 是通信原语 |
| HashMap\<K,V\> | map[K]V | 原生类型，引用语义，需 make 初始化 |
| ArrayList\<T\> | []T（slice 切片） | 自动扩容，比 Java 更易用 |
| Stream / Optional | 手写 for 循环 | Go 无泛型高阶函数（旧版），1.18+ 有泛型，但仍偏过程式 |
| Lombok @Data | 手写 getter/setter 或 IDE 生成 | 但通常直接访问 public 字段 |
| Spring Boot IoC 容器 | 显式构造函数传参（手动 DI） | 没有强制的 DI 框架，简单项目直接 new；大型项目可选 wire/di 工具 |
| MyBatis Mapper | GORM + Repository 模式 | GORM 用方法链写查询，功能类似 Criteria API |

---

## 快速开始

### 0. 配置环境变量（Windows）

```powershell
$env:GOROOT = "d:\文件\test\tools\go"
$env:Path   = "d:\文件\test\tools\go\bin;$env:Path"
$env:GO111MODULE = "on"
# 推荐国内代理（加速依赖下载）
$env:GOPROXY = "https://goproxy.cn,direct"
go version  # 验证：go version go1.22.5 windows/amd64
```

### 1. 运行基础语法示例

```powershell
cd d:\文件\test\go_learn\basics
go run 01_variables.go
go run 02_types.go
# ... 依次运行
```

### 2. 运行框架示例（以 Gin 为例）

```powershell
cd d:\文件\test\go_learn\frameworks\01_gin
go mod tidy     # 下载依赖
go run main.go
# 浏览器访问 http://localhost:18080/ping
```

### 3. 学习路径建议

```
第 1 天：01~04   变量 → 类型 → 控制流 → 函数
第 2 天：05~06   指针 → 切片&Map（核心数据结构，多练）
第 3 天：07~09   结构体 → 接口 → 错误处理（面向对象核心）
第 4 天：10~11   并发编程 → 包与模块（Go 最引以为傲的特性）
第 5 天：01_gin   Web 框架：写个 CRUD 接口
第 6 天：02_gorm  ORM：把接口数据持久化
第 7 天：03_goredis + 04_layered  缓存 + 完整分层工程
```

---

## 重要理念：Go 和 Java 不一样

1. **简单优先**：Go 刻意砍掉了继承/重载/泛型复杂用例，让"随便哪个工程师写的代码都差不多"。
2. **显式优先**：没有隐式类型转换、没有构造函数重载、没有 DI 容器隐式注入。代码一眼看透。
3. **错误即值**：`if err != nil` 是家常便饭，不是语法啰嗦，是强制你思考每条路径。
4. **组合优于继承**：用嵌入小结构体代替大继承树。
5. **并发一等公民**：`go f()` 开一个协程，Channel 做通信，比线程池/Callable/Future 轻量得多。

---

## 延伸阅读

- 官方中文教程：https://go.dev/tour/list
- 圣经《Effective Go》中文：https://go.dev/doc/effective_go
- 高性能 Go：https://dave.cheney.net/high-performance-go-workshop
- 本 `go_learn` 目录内所有代码，改参数、加断点、跑通即可掌握。
