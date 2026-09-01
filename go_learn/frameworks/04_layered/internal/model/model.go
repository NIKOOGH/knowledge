// ============================================================
// frameworks/04_layered/internal/model/model.go  模型层（DTO / 实体）
// ============================================================
package model

import "time"

// Product ORM 实体（对应 t_product 表）
type Product struct {
	ID         int64     `json:"id"          gorm:"column:id;primaryKey"`
	Name       string    `json:"name"        gorm:"column:name;type:varchar(128);index"`
	Price      int       `json:"price"       gorm:"column:price"`
	Stock      int       `json:"stock"       gorm:"column:stock"`
	CategoryID int64     `json:"category_id" gorm:"column:category_id"`
	UpdatedAt  time.Time `json:"updated_at"  gorm:"column:updated_at"`
	CreatedAt  time.Time `json:"created_at"  gorm:"column:created_at"`
}

func (Product) TableName() string { return "t_product" }

// ProductVO 视图对象 / Response VO（去掉敏感字段 + 增加展示字段）
type ProductVO struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Price   int    `json:"price"`
	Stock   int    `json:"stock"`
	PriceY  string `json:"price_yuan"` // 分转元，展示用
}

// CreateProductRequest 新增产品请求（binding 标签）
type CreateProductRequest struct {
	Name       string `json:"name" binding:"required,min=1,max=128"`
	Price      int    `json:"price" binding:"gte=0"`
	Stock      int    `json:"stock" binding:"gte=0"`
	CategoryID int64  `json:"category_id" binding:"gte=0"`
}

// UpdateProductRequest 改价/改库存请求
type UpdateProductRequest struct {
	Price int `json:"price" binding:"gte=0"`
	Stock int `json:"stock" binding:"gte=0"`
}

// Resp 统一响应（与 Gin 示例保持一致）
type Resp struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Ts      int64       `json:"ts"`
}

// PageResult 分页响应
type PageResult struct {
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
	List  interface{} `json:"list"`
}
