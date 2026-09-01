// ============================================================
// frameworks/04_layered/cmd/server/main.go  启动入口（Wire 依赖装配）
// ------------------------------------------------------------
// 职责：组装 DB / Redis / Repo / Service / Handler，
//       启动 Gin 服务器。
// 类比 Java 的 SpringApplication.run + @Configuration @Bean
// ============================================================
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"go-learn-layered/internal/config"
	"go-learn-layered/internal/handler"
	"go-learn-layered/internal/model"
	"go-learn-layered/internal/repository"
	"go-learn-layered/internal/service"

	"github.com/gin-gonic/gin"
	goredis "github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================
// 0. 认证中间件（简化：X-Token=ADMIN）—— 生产版本里可以抽成独立文件
// ============================================================
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Token") != "ADMIN" {
			c.AbortWithStatusJSON(http.StatusUnauthorized,
				model.Resp{Code: 401, Message: "未登录或 Token 无效", Ts: time.Now().UnixMilli()})
			return
		}
		c.Next()
	}
}

// ============================================================
// 1. 启动 DB（SQLite 内存库，演示无外部依赖）
// ============================================================
func startDB() *gorm.DB {
	cfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)}
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared&_fk=1"), cfg)
	if err != nil {
		log.Fatalf("打开 DB 失败：%v", err)
	}
	if err := db.AutoMigrate(&model.Product{}); err != nil {
		log.Fatalf("AutoMigrate 失败：%v", err)
	}
	// 预置示例数据
	seeds := []*model.Product{
		{ID: 1, Name: "iPhone 15", Price: 699900, Stock: 100, CategoryID: 1},
		{ID: 2, Name: "小米 14", Price: 399900, Stock: 200, CategoryID: 1},
		{ID: 3, Name: "MacBook Pro", Price: 1499900, Stock: 50, CategoryID: 2},
		{ID: 4, Name: "AirPods Pro", Price: 159900, Stock: 500, CategoryID: 3},
	}
	for _, p := range seeds {
		_ = db.Create(p).Error
	}
	log.Println("[bootstrap] DB 初始化完成（SQLite 内存库）")
	return db
}

// ============================================================
// 2. 启动 Redis（可空，用 config.Redis.Enabled 控制）
// ============================================================
func startRedis(cfg config.RedisConfig) *goredis.Client {
	if !cfg.Enabled {
		log.Println("[bootstrap] Redis 未启用（config.Redis.Enabled=false），走降级模式（无缓存）")
		return nil
	}
	r := goredis.NewClient(&goredis.Options{Addr: cfg.Addr, Password: cfg.Password, DB: cfg.DB})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		log.Printf("[bootstrap] Redis 不可达，降级为无缓存模式：%v", err)
		return nil
	}
	log.Println("[bootstrap] Redis 连接成功")
	return r
}

// ============================================================
// 3. 组装依赖 + 启动
// ============================================================
func main() {
	cfg := config.Load()
	log.Printf("[bootstrap] 配置加载完成：server.addr=%s redis.enabled=%v", cfg.Server.Addr, cfg.Redis.Enabled)

	// ---- 基础设施 ----
	db := startDB()
	rdb := startRedis(cfg.Redis)

	// ---- Repository（DAO 层）----
	productRepo := repository.NewProductRepo(db)

	// ---- Service（业务层，DI 注入 Repo + DB + Redis）----
	productSvc := service.NewProductService(productRepo, db, rdb)

	// ---- Handler（Controller 层）----
	productHandler := handler.NewProductHandler(productSvc)

	// ---- Gin 路由 ----
	gin.SetMode(cfg.Server.Mode)
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery())

	// 健康检查
	engine.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong @ "+time.Now().Format("15:04:05"))
	})

	// 注册业务路由（写接口会走 authMiddleware）
	productHandler.Register(engine, authMiddleware())

	// ---- 启动服务 ----
	srv := &http.Server{
		Addr:    cfg.Server.Addr,
		Handler: engine,
		// 合理的超时配置
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("[bootstrap] 启动在 %s ...\n", cfg.Server.Addr)
	log.Println("  GET  /api/v1/product/:id     → 查产品详情（公开，带缓存）")
	log.Println("  GET  /api/v1/products?k=&page=&size= → 搜索")
	log.Println("  POST /api/v1/product        → 新增（需要 X-Token: ADMIN）")
	log.Println("  PUT  /api/v1/product/:id    → 更新")
	log.Println("  DEL  /api/v1/product/:id    → 删除")
	log.Println("  POST /api/v1/product/:id/deduct?n=N  → 扣库存")

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("启动失败：%v", err)
	}
}
