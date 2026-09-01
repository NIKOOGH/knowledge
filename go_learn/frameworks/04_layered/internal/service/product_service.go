// ============================================================
// frameworks/04_layered/internal/service/product_service.go  Service 层（业务逻辑）
// ------------------------------------------------------------
// 职责：编排 Repository 调用、写业务校验、加缓存 / 事务。
//       不碰 HTTP，不直接拼 JSON，方便单元测试。
//
// 分层类比 Java：
//   Controller（HTTP）→ Service（业务）→ Repository（DAO）→ DB
// ============================================================
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"go-learn-layered/internal/model"
	"go-learn-layered/internal/repository"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// ============================================================
// 1. 定义 Service 接口（方便 mock 测试 / 替换实现）
// ============================================================
type IProductService interface {
	Create(ctx context.Context, req *model.CreateProductRequest) (*model.ProductVO, error)
	GetByID(ctx context.Context, id int64) (*model.ProductVO, error)
	Update(ctx context.Context, id int64, req *model.UpdateProductRequest) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, keyword string, page, size int) (*model.PageResult, error)
	DeductStock(ctx context.Context, id int64, n int) error
}

// ============================================================
// 2. 实现（显式依赖：Repo + Redis）
//    Go 不用 @Autowired，通过构造函数传参（手动 DI）
// ============================================================
type productService struct {
	repo repository.IProductRepo
	db   *gorm.DB          // 用来开事务
	rdb  *redis.Client     // Redis 客户端（nil 表示禁用缓存）
}

func NewProductService(repo repository.IProductRepo, db *gorm.DB, rdb *redis.Client) IProductService {
	return &productService{repo: repo, db: db, rdb: rdb}
}

// ============================================================
// 3. 业务方法
// ============================================================

// 3.1 新增
func (s *productService) Create(ctx context.Context, req *model.CreateProductRequest) (*model.ProductVO, error) {
	// 业务校验：价格/库存已由 binding 限制，这里可以做更细的校验
	if req.Stock > 1_000_000 {
		return nil, errors.New("stock 不能超过 100 万（业务规则）")
	}
	p := &model.Product{
		Name:       req.Name,
		Price:      req.Price,
		Stock:      req.Stock,
		CategoryID: req.CategoryID,
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create product failed: %w", err)
	}
	// 写缓存（如果 Redis 可用）
	s.setProductCache(ctx, p)
	return toVO(p), nil
}

// 3.2 查详情：缓存优先，缓存 miss 再查 DB
func (s *productService) GetByID(ctx context.Context, id int64) (*model.ProductVO, error) {
	// ---- 查 Redis ----
	if p := s.getProductCache(ctx, id); p != nil {
		return toVO(p), nil
	}
	// ---- 查 DB ----
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("产品不存在")
		}
		return nil, err
	}
	// ---- 回填缓存 ----
	s.setProductCache(ctx, p)
	return toVO(p), nil
}

// 3.3 更新（改价/改库存）：改完要失效缓存
func (s *productService) Update(ctx context.Context, id int64, req *model.UpdateProductRequest) error {
	if req == nil {
		return errors.New("update request is nil")
	}
	if err := s.repo.Update(ctx, id, req.Price, req.Stock); err != nil {
		return err
	}
	// 失效缓存
	s.delProductCache(ctx, id)
	return nil
}

// 3.4 删除：同样失效缓存
func (s *productService) Delete(ctx context.Context, id int64) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.delProductCache(ctx, id)
	return nil
}

// 3.5 搜索：不走缓存（条件太多）
func (s *productService) Search(ctx context.Context, keyword string, page, size int) (*model.PageResult, error) {
	list, total, err := s.repo.Search(ctx, keyword, page, size)
	if err != nil {
		return nil, err
	}
	voList := make([]*model.ProductVO, 0, len(list))
	for _, p := range list {
		voList = append(voList, toVO(p))
	}
	return &model.PageResult{Total: total, Page: page, Size: size, List: voList}, nil
}

// 3.6 扣库存：事务保护（只是单个 update，但预留事务用法示例）
func (s *productService) DeductStock(ctx context.Context, id int64, n int) error {
	if n <= 0 {
		return errors.New("deduct amount must > 0")
	}
	rows, err := s.repo.DeductStock(ctx, id, n)
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("库存不足或商品不存在")
	}
	// 扣完库存：失效缓存（DB 变了）
	s.delProductCache(ctx, id)
	return nil
}

// ============================================================
// 4. 内部辅助：Redis 缓存读写 / VO 转换
// ============================================================

// 缓存 key 前缀
const cachePrefix = "layered:product:"
const cacheTTL = 5 * time.Minute

func cacheKey(id int64) string {
	return fmt.Sprintf("%s%d", cachePrefix, id)
}

func (s *productService) setProductCache(ctx context.Context, p *model.Product) {
	if s.rdb == nil {
		return
	}
	bs, err := json.Marshal(p)
	if err != nil {
		return
	}
	_ = s.rdb.Set(ctx, cacheKey(p.ID), bs, cacheTTL).Err()
}

func (s *productService) getProductCache(ctx context.Context, id int64) *model.Product {
	if s.rdb == nil {
		return nil
	}
	bs, err := s.rdb.Get(ctx, cacheKey(id)).Bytes()
	if err != nil || len(bs) == 0 {
		return nil
	}
	var p model.Product
	if err := json.Unmarshal(bs, &p); err != nil {
		return nil
	}
	return &p
}

func (s *productService) delProductCache(ctx context.Context, id int64) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, cacheKey(id)).Err()
}

// 实体 → VO（做字段转换，如分 → 元）
func toVO(p *model.Product) *model.ProductVO {
	return &model.ProductVO{
		ID:    p.ID,
		Name:  p.Name,
		Price: p.Price,
		Stock: p.Stock,
		// 演示：存 DB 的价格以"分"为单位，返回展示时除以 100
		PriceY: fmt.Sprintf("%.2f 元", float64(p.Price)/100.0),
	}
}
