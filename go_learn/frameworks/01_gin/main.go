// ============================================================
// frameworks/01_gin/main.go  Gin v1.9 入门示例
// ------------------------------------------------------------
// Gin 是 Go 最流行的 Web 框架，HTTP 路由+中间件生态完善，
// 对标 Spring MVC / Spring Boot Web。
//
// 关键功能：
//   1) 路由（GET/POST/PUT/DELETE + 路径参数）
//   2) JSON 绑定与响应
//   3) 查询参数 / 表单 / 表单绑定
//   4) 中间件（全局 / 分组 / 单路由）
//   5) 文件上传
//   6) 错误处理 + 统一响应
//
// 运行步骤：
//   cd frameworks/01_gin
//   go mod tidy    # 下载依赖（首次）
//   go run main.go
//   浏览器访问：
//     http://localhost:18080/ping
//     http://localhost:18080/user/1001?name=Alice
// ============================================================

package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 0. 统一响应结构体（类比 ResponseResult / R）
// ============================================================
type Resp struct {
	Code    int         `json:"code"`    // 0 成功，其他失败
	Message string      `json:"message"` // 信息
	Data    interface{} `json:"data"`    // 业务数据
	Ts      int64       `json:"ts"`      // 时间戳
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Resp{0, "success", data, time.Now().UnixMilli()})
}
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Resp{code, msg, nil, time.Now().UnixMilli()})
}
func FailErr(c *gin.Context, code int, err error) {
	Fail(c, code, err.Error())
}

// ============================================================
// 1. DTO：请求/响应模型 + binding 标签
// ============================================================

// UserCreateRequest 新增用户请求（binding 标签用于参数校验）
type UserCreateRequest struct {
	// form 用于 form-data，json 用于 JSON，binding:"required" 表示必填
	Name  string `json:"name" form:"name" binding:"required,min=2,max=20"`
	Age   int    `json:"age" form:"age" binding:"gte=0,lte=150"`
	Email string `json:"email" form:"email" binding:"required,email"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

// UserVO 用户响应对象（只返回需要的字段）
type UserVO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
}

// ============================================================
// 2. 业务层：一个假的 UserService（后面 GORM 示例会替换成真实 DB）
// ============================================================
type UserService struct {
	store map[int64]*UserVO
	inc   int64
}

func NewUserService() *UserService {
	m := make(map[int64]*UserVO)
	// 预置几个示例用户
	m[1001] = &UserVO{ID: 1001, Name: "Alice", Age: 20, Email: "a@x.com"}
	m[1002] = &UserVO{ID: 1002, Name: "Bob", Age: 30, Email: "b@x.com"}
	return &UserService{store: m, inc: 2000}
}

var (
	errUserNotFound = errors.New("用户不存在")
	errDupEmail     = errors.New("邮箱已被注册")
)

func (s *UserService) Get(id int64) (*UserVO, error) {
	u, ok := s.store[id]
	if !ok {
		return nil, errUserNotFound
	}
	return u, nil
}

func (s *UserService) Create(req *UserCreateRequest) (*UserVO, error) {
	for _, u := range s.store {
		if strings.EqualFold(u.Email, req.Email) {
			return nil, errDupEmail
		}
	}
	s.inc++
	u := &UserVO{ID: s.inc, Name: req.Name, Age: req.Age, Email: req.Email}
	s.store[u.ID] = u
	return u, nil
}

func (s *UserService) Delete(id int64) error {
	if _, ok := s.store[id]; !ok {
		return errUserNotFound
	}
	delete(s.store, id)
	return nil
}

// ============================================================
// 3. 中间件示例：认证中间件（看请求头 X-Token）
// ============================================================
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 简单示例：X-Token=ADMIN 放行，否则 401
		token := c.GetHeader("X-Token")
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, Resp{401, "请先登录", nil, time.Now().UnixMilli()})
			return
		}
		// 把 token 存进上下文，后续 handler 可拿
		c.Set("token", token)
		c.Next() // 继续执行后面的 handler
	}
}

// 访问日志中间件（gin.Default() 自带，这里演示自定义）
func AccessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start)
		log.Printf("[ACCESS] %s %s -> %d (cost %s)",
			c.Request.Method, c.Request.URL, c.Writer.Status(), cost)
	}
}

// ============================================================
// 4. 启动与路由注册
// ============================================================
func main() {
	// 生产模式（Gin 不输出 debug 日志）
	gin.SetMode(gin.ReleaseMode)
	// r := gin.Default()   // Default() = gin.New() + Logger + Recovery 中间件
	r := gin.New()
	r.Use(AccessLog())              // 日志
	r.Use(gin.Recovery())           // panic 捕获，防止整个进程挂（重要）

	// 服务实例
	userSvc := NewUserService()

	// ============================================================
	// 基础路由
	// ============================================================
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong @ "+time.Now().Format("15:04:05"))
	})

	// 带查询参数：/hello?name=Tom
	r.GET("/hello", func(c *gin.Context) {
		name := c.DefaultQuery("name", "Guest") // 没有就给默认值
		c.JSON(http.StatusOK, gin.H{"greet": "Hello, " + name})
	})

	// 带路径参数：/user/:id
	r.GET("/user/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		var id int64
		// 用 fmt.Sscanf 比 ParseInt 简写
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			Fail(c, 400, "id 必须是整数")
			return
		}
		u, err := userSvc.Get(id)
		if err != nil {
			Fail(c, 404, err.Error())
			return
		}
		OK(c, u)
	})

	// ============================================================
	// 分组路由 + 中间件（类比 Spring 的 @RequestMapping + 切面）
	// ============================================================
	// /api/v1 分组全部要求 X-Token 认证
	v1 := r.Group("/api/v1")
	v1.Use(AuthMiddleware())
	{
		// POST /api/v1/user  —— JSON 体绑定
		v1.POST("/user", func(c *gin.Context) {
			var req UserCreateRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				FailErr(c, 400, err)
				return
			}
			u, err := userSvc.Create(&req)
			if err != nil {
				Fail(c, 409, err.Error())
				return
			}
			OK(c, u)
		})

		// DELETE /api/v1/user/:id
		v1.DELETE("/user/:id", func(c *gin.Context) {
			var id int64
			if _, err := fmt.Sscanf(c.Param("id"), "%d", &id); err != nil {
				Fail(c, 400, "id 必须是整数")
				return
			}
			if err := userSvc.Delete(id); err != nil {
				Fail(c, 404, err.Error())
				return
			}
			OK(c, gin.H{"deleted_id": id})
		})

		// POST /api/v1/login —— form 表单绑定
		v1.POST("/login", func(c *gin.Context) {
			var req LoginRequest
			if err := c.ShouldBind(&req); err != nil {
				FailErr(c, 400, err)
				return
			}
			if req.Username == "admin" && req.Password == "123456" {
				OK(c, gin.H{"token": "mock-token-" + fmt.Sprint(time.Now().Unix())})
				return
			}
			Fail(c, 401, "用户名或密码错误")
		})

		// POST /api/v1/upload —— 文件上传
		v1.POST("/upload", func(c *gin.Context) {
			// 单文件
			fh, err := c.FormFile("file")
			if err != nil {
				FailErr(c, 400, err)
				return
			}
			dir := "./uploads"
			_ = os.MkdirAll(dir, 0755)
			saveTo := fmt.Sprintf("%s/%d_%s", dir, time.Now().Unix(), fh.Filename)
			if err := c.SaveUploadedFile(fh, saveTo); err != nil {
				FailErr(c, 500, err)
				return
			}
			OK(c, gin.H{"filename": fh.Filename, "saved_to": saveTo, "size": fh.Size})
		})
	}

	// ============================================================
	// 404 NoRoute
	// ============================================================
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, Resp{404, "Not Found: " + c.Request.URL.Path, nil, time.Now().UnixMilli()})
	})

	// ============================================================
	// 启动
	// ============================================================
	addr := ":18080"
	log.Printf("[Gin] 启动在 %s ...\n", addr)
	log.Println("GET  /ping                 → 健康检查")
	log.Println("GET  /hello?name=Tom       → 带 query")
	log.Println("GET  /user/1001            → 路径参数")
	log.Println("POST /api/v1/user          → 新增（需 X-Token: ADMIN 请求头）")
	if err := r.Run(addr); err != nil {
		log.Fatalf("启动失败：%v", err)
	}
}
