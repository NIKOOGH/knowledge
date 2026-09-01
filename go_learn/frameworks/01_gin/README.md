# 01_gin —— Gin v1.9 Web 框架入门

> 对标 Java Spring MVC / Spring Boot Web。

## 运行

```powershell
$env:GOROOT = "d:\文件\test\tools\go"
$env:Path   = "d:\文件\test\tools\go\bin;$env:Path"
$env:GOPROXY = "https://goproxy.cn,direct"

cd d:\文件\test\go_learn\frameworks\01_gin
go mod tidy       # 拉依赖（首次）
go run main.go
```

## 接口列表（全部用 curl / 浏览器 / Postman 实测）

### 1. 健康检查

```bash
curl http://localhost:18080/ping
```

### 2. 带 query

```bash
curl "http://localhost:18080/hello?name=Tom"
```

### 3. 路径参数（查用户）

```bash
curl http://localhost:18080/user/1001
# 不存在 → code 404
curl http://localhost:18080/user/9999
```

### 4. 需要认证的分组（`/api/v1/*` 都要求 `X-Token: ADMIN`）

#### 4.1 新增用户

```bash
curl -X POST http://localhost:18080/api/v1/user \
  -H "Content-Type: application/json" \
  -H "X-Token: ADMIN" \
  -d '{"name":"Carol","age":28,"email":"c@x.com"}'
```

#### 4.2 删除用户

```bash
curl -X DELETE http://localhost:18080/api/v1/user/1001 \
  -H "X-Token: ADMIN"
```

#### 4.3 登录（form）

```bash
curl -X POST http://localhost:18080/api/v1/login \
  -H "X-Token: ADMIN" \
  -d 'username=admin&password=123456'
```

#### 4.4 上传文件

```bash
curl -X POST http://localhost:18080/api/v1/upload \
  -H "X-Token: ADMIN" \
  -F "file=@./README.md"
```

### 5. 无 X-Token 应该 401

```bash
curl -X POST http://localhost:18080/api/v1/user \
  -H "Content-Type: application/json" \
  -d '{"name":"Hacker","age":20,"email":"h@x.com"}'
```

### 6. 参数校验失败

```bash
curl -X POST http://localhost:18080/api/v1/user \
  -H "Content-Type: application/json" -H "X-Token: ADMIN" \
  -d '{"name":"a","age":-1,"email":"not-email"}'
```

## 核心知识点对应代码位置

| 功能 | 代码位置 |
|------|---------|
| 路由 + 路径参数 + 查询参数 | `main()` 里的 r.GET 片段 |
| JSON 绑定与验证 | `UserCreateRequest` + `c.ShouldBindJSON` |
| Form 绑定 | `c.ShouldBind(&req)`（无 JSON） |
| 分组路由（类比 `@RequestMapping("/api/v1")`） | `v1 := r.Group("/api/v1")` |
| 全局中间件（日志 / panic 捕获） | `r.Use(AccessLog(), gin.Recovery())` |
| 分组中间件（认证） | `v1.Use(AuthMiddleware())` |
| 上下文传值 | `c.Set / c.Get` |
| 统一响应 | `OK/Fail/FailErr 三函数` |
| NoRoute 404 | `r.NoRoute(...)` |
| 文件上传 | `c.FormFile / c.SaveUploadedFile` |
