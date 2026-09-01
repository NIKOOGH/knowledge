# 04_layered —— Controller → Service → Repository 分层工程示例

> 对标 Java Spring Boot 标准三层架构：`Controller → Service → Mapper`
> Go 等价：`Handler/Controller → Service → Repository`

## 目录结构（规范版）

```
04_layered/
├── cmd/
│   └── server/main.go     启动入口 + 依赖装配（SpringApplication.run + @Configuration）
├── internal/
│   ├── config/config.go       配置加载
│   ├── model/model.go         实体/DTO/VO/统一响应
│   ├── repository/product_repo.go  DAO 层（IProductRepo 接口 + GORM 实现）
│   ├── service/product_service.go   业务层（依赖注入 + 缓存 + 事务）
│   └── handler/product_handler.go   Controller 层（HTTP + 参数校验 + 调用 Service）
├── go.mod
└── README.md
```

## 对照心智图（Java → Go）

| Java 概念 | Go 本项目对应 | 说明 |
|----------|--------------|------|
| SpringApplication.run | `cmd/server/main.go` | 启动入口，做依赖装配 |
| `@Configuration` `@Bean` | `startDB / startRedis / NewXxx` | 手动构造 Bean 并返回 |
| `@Autowired` / `@Resource` | 构造函数传参 | Go 没 DI 容器，显式依赖注入（更清晰） |
| `@RestController` / `@RequestMapping` | `handler/ProductHandler` + `Register()` | Handler 负责路由注册与 HTTP 协议 |
| `@RequestBody` + `@Valid` | `c.ShouldBindJSON(&req)` + `binding:"xxx"` 标签 | Gin + validator 自动校验 |
| `@Service` + 业务逻辑 | `service/product_service.go` | Service 层，编排 Repo，写事务/缓存 |
| `MyBatis Mapper` / `Spring Data JPA` | `repository/product_repo.go` | Repo 层，只写 SQL/GORM，不写业务 |
| `@Cacheable` / `@CacheEvict` | Service 层手写 `set/get/delProductCache` | 显式写 Redis 读写，更可控 |
| `@Transactional` | 见 Service 层注释：用 `db.Transaction` 包裹 | 事务写在 Service，Go 传 Context 携带事务连接 |
| `OncePerRequestFilter` / 拦截器 | `gin.HandlerFunc` + `Use(...)` | 中间件 |
| `ResponseEntity<T>` | `model.Resp{code,msg,data,ts}` | 统一响应结构体 |
| `@PageableDefault` + `Page<T>` | `PageResult{total,page,size,list}` | 手写分页响应 |

## 运行

```powershell
$env:GOROOT = "d:\文件\test\tools\go"
$env:Path   = "d:\文件\test\tools\go\bin;$env:Path"
$env:GOPROXY = "https://goproxy.cn,direct"

cd d:\文件\test\go_learn\frameworks\04_layered
go mod tidy
go run ./cmd/server
```

## 接口示例（测试分层能力 + 缓存 + 扣库存）

### 1. 健康检查

```
curl http://localhost:18088/ping
```

### 2. 查产品（公开，带 Redis 缓存，若 Redis Enabled=false 则跳过）

```
curl http://localhost:18088/api/v1/product/1
# 响应中 price=699900 分（存储）+ price_yuan="6999.00 元"（展示）
```

### 3. 搜索（分页）

```
curl "http://localhost:18088/api/v1/products?k=iPhone&page=1&size=10"
```

### 4. 新增产品（需要 X-Token=ADMIN）

```bash
curl -X POST http://localhost:18088/api/v1/product \
  -H "Content-Type: application/json" \
  -H "X-Token: ADMIN" \
  -d '{"name":"小米 14 Ultra","price":649900,"stock":150,"category_id":1}'
```

### 5. 扣库存（经典"缓存失效"场景：Service 层 DeductStock 调用 repo.DeductStock 后会 delProductCache）

```bash
curl -X POST "http://localhost:18088/api/v1/product/1/deduct?n=2" \
  -H "X-Token: ADMIN"
# 再次查 /api/v1/product/1 可以看到 stock 减了 2，且下一次会重新命中缓存
```

### 6. 未带 Token 应该 401（鉴权中间件生效）

```bash
curl -X POST http://localhost:18088/api/v1/product \
  -H "Content-Type: application/json" \
  -d '{"name":"测试","price":1,"stock":1}'
# → code: 401
```

## 本项目体现的 Go 工程原则

1. **显式依赖注入（无 IoC 容器）**：所有依赖通过构造函数传入，调用关系一目了然。
   需要 IoC 可以引入 Google Wire / Uber dig，但中小项目手写最清晰。

2. **接口定义在使用者包**：IProductRepo 接口定义在 `repository` 包（但 Go 惯例是定义在使用方）。
   大规模团队可把接口挪到 `service` 包，Repo 去实现它。

3. **Context 贯穿始终**：所有 Repository/Service 方法首参数都是 `ctx`，
   配合 HTTP Handler 的 `c.Request.Context()`，天然支持超时、取消、trace 透传。

4. **业务逻辑不碰 HTTP**：Service 层参数是 struct，返回是 error，
   能不启动 Gin 也可以写单元测试（传个 mockRepo 即可）。

5. **错误不是异常**：每层用 `fmt.Errorf("xxx: %w", err)` 包装，
   最上层 Handler 统一处理（这里简化直接打印，正式项目可用 `errors.As` 判断错误码）。
