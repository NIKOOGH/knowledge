// ============================================================
// frameworks/04_layered/internal/handler/product_handler.go  Controller / Handler 层
// ------------------------------------------------------------
// 职责：接收 HTTP、参数校验、调用 Service、统一响应。
//       类比 Spring Boot Controller：@RestController + @RequestMapping。
// ============================================================
package handler

import (
	"net/http"
	"strconv"
	"time"

	"go-learn-layered/internal/model"
	"go-learn-layered/internal/service"

	"github.com/gin-gonic/gin"
)

// ============================================================
// 响应辅助（所有 Handler 都用这 3 个函数写回响应）
// ============================================================
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, model.Resp{Code: 0, Message: "success", Data: data, Ts: time.Now().UnixMilli()})
}
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, model.Resp{Code: code, Message: msg, Data: nil, Ts: time.Now().UnixMilli()})
}

// ============================================================
// ProductHandler：产品相关 HTTP 入口（通过依赖注入拿到 Service）
// ============================================================
type ProductHandler struct {
	svc service.IProductService
}

func NewProductHandler(svc service.IProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// pathID 解析：把 :id 路径参数转成 int64
func pathID(c *gin.Context) (int64, bool) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}

// Register 注册路由到 Gin Router（分层职责：Handler 管自己的 URL）
func (h *ProductHandler) Register(engine *gin.Engine, auth gin.HandlerFunc) {
	// 读写分开：查询接口公开，写接口要求鉴权
	api := engine.Group("/api/v1")
	{
		// 公开接口
		api.GET("/product/:id", h.Get)
		api.GET("/products", h.Search)

		// 写接口：走认证中间件（简化版：auth 中间件只是个函数，由 main 传进来）
		write := api.Group("/")
		if auth != nil {
			write.Use(auth)
		}
		write.POST("/product", h.Create)
		write.PUT("/product/:id", h.Update)
		write.DELETE("/product/:id", h.Delete)
		write.POST("/product/:id/deduct", h.Deduct)
	}
}

// ============================================================
// 具体 Handler
// ============================================================

// Create 新增
func (h *ProductHandler) Create(c *gin.Context) {
	var req model.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, err.Error())
		return
	}
	vo, err := h.svc.Create(c.Request.Context(), &req)
	if err != nil {
		Fail(c, 500, err.Error())
		return
	}
	OK(c, vo)
}

// Get 查单条
func (h *ProductHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		Fail(c, 400, "id 必须是整数")
		return
	}
	vo, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		Fail(c, 404, err.Error())
		return
	}
	OK(c, vo)
}

// Update 更新
func (h *ProductHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		Fail(c, 400, "id 必须是整数")
		return
	}
	var req model.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Fail(c, 400, err.Error())
		return
	}
	if err := h.svc.Update(c.Request.Context(), id, &req); err != nil {
		Fail(c, 500, err.Error())
		return
	}
	OK(c, gin.H{"id": id})
}

// Delete 删除
func (h *ProductHandler) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		Fail(c, 400, "id 必须是整数")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Fail(c, 500, err.Error())
		return
	}
	OK(c, gin.H{"deleted_id": id})
}

// Search 查询列表
func (h *ProductHandler) Search(c *gin.Context) {
	keyword := c.DefaultQuery("k", "")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	pageResult, err := h.svc.Search(c.Request.Context(), keyword, page, size)
	if err != nil {
		Fail(c, 500, err.Error())
		return
	}
	OK(c, pageResult)
}

// Deduct 扣库存接口（演示事务+缓存失效）
func (h *ProductHandler) Deduct(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		Fail(c, 400, "id 必须是整数")
		return
	}
	n, err := strconv.Atoi(c.DefaultQuery("n", "1"))
	if err != nil || n <= 0 {
		Fail(c, 400, "n 必须是正整数")
		return
	}
	if err := h.svc.DeductStock(c.Request.Context(), id, n); err != nil {
		Fail(c, 500, err.Error())
		return
	}
	OK(c, gin.H{"deduct_id": id, "n": n})
}
