// ============================================================
// frameworks/04_layered/internal/repository/product_repo.go  Repository 层（DAO）
// ------------------------------------------------------------
// 职责：只关心"怎么读写数据库"，不写业务逻辑。
//       类比 Java Spring Data JPA / MyBatis Mapper。
//       如果有缓存，这一层也负责（见 CacheDecorator 模式：ProductRepoCache）
// ============================================================
package repository

import (
	"context"
	"errors"
	"fmt"

	"go-learn-layered/internal/model"

	"gorm.io/gorm"
)

// IProductRepo 定义"产品 Repository"应该有的行为（接口）
type IProductRepo interface {
	Create(ctx context.Context, p *model.Product) error
	GetByID(ctx context.Context, id int64) (*model.Product, error)
	Update(ctx context.Context, id int64, price int, stock int) error
	Delete(ctx context.Context, id int64) error
	Search(ctx context.Context, keyword string, page, size int) ([]*model.Product, int64, error)
	DeductStock(ctx context.Context, id int64, n int) (int64, error)
}

// productRepo 是 GORM 实现
type productRepo struct {
	db *gorm.DB
}

// NewProductRepo 构造（依赖注入：把 *gorm.DB 传进来）
func NewProductRepo(db *gorm.DB) IProductRepo {
	return &productRepo{db: db}
}

func (r *productRepo) Create(ctx context.Context, p *model.Product) error {
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *productRepo) GetByID(ctx context.Context, id int64) (*model.Product, error) {
	var p model.Product
	err := r.db.WithContext(ctx).First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *productRepo) Update(ctx context.Context, id int64, price int, stock int) error {
	updates := map[string]interface{}{}
	if price >= 0 {
		updates["price"] = price
	}
	if stock >= 0 {
		updates["stock"] = stock
	}
	if len(updates) == 0 {
		return nil
	}
	res := r.db.WithContext(ctx).Model(&model.Product{}).Where("id = ?", id).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("update no rows affected")
	}
	return nil
}

func (r *productRepo) Delete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&model.Product{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("delete no rows affected")
	}
	return nil
}

func (r *productRepo) Search(ctx context.Context, keyword string, page, size int) ([]*model.Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	tx := r.db.WithContext(ctx).Model(&model.Product{})
	if keyword != "" {
		tx = tx.Where("name LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*model.Product
	err := tx.Order("id DESC").Limit(size).Offset((page - 1) * size).Find(&list).Error
	return list, total, err
}

func (r *productRepo) DeductStock(ctx context.Context, id int64, n int) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Product{}).
		Where("id = ? AND stock >= ?", id, n).
		Update("stock", gorm.Expr("stock - ?", n))
	if res.Error != nil {
		return 0, fmt.Errorf("deduct stock db err: %w", res.Error)
	}
	return res.RowsAffected, nil
}
