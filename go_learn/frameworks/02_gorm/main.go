// ============================================================
// frameworks/02_gorm/main.go  GORM v1.25 入门示例
// ------------------------------------------------------------
// GORM 是 Go 最流行的 ORM，功能类似 MyBatis + Spring Data JPA，
// 自带自动建表、CRUD、条件查询、关联、分页等常用能力。
//
// 关键示例：
//   1) Model 定义（struct tag）
//   2) 连接数据库（默认用 SQLite 风格的 file:mem 模式，无需安装）
//   3) 自动迁移（AutoMigrate）
//   4) 增删改查
//   5) 条件查询（Where、Select、Order、Limit、Offset 即分页）
//   6) 事务（Transaction）
//   7) 钩子（BeforeCreate / BeforeUpdate）
//
// 运行步骤：
//   cd frameworks/02_gorm
//   go mod tidy
//   go run main.go
//
//   无需准备数据库：默认用 SQLite 内存库（go-sqlite3），
//   若要切 MySQL，取消下方 DSN 注释即可。
// ============================================================

package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/driver/sqlite"
	// 想切 MySQL：
	// "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================
// 1. Model 定义（类比 @Entity + @Column）
// ============================================================
// 默认约定：
//   1) 结构体名 = User → DB 表名 = users（蛇形复数，可改）
//   2) 字段名 = CreatedAt → 列名 = created_at
//   3) gorm.Model 自带 ID/CreateTime/UpdateTime/DeleteTime（软删）
//      也可以不嵌入，自己写字段

// gorm.Model 源码（供参考）：
//   type Model struct {
//       ID        uint           `gorm:"primaryKey"`
//       CreatedAt time.Time
//       UpdatedAt time.Time
//       DeletedAt gorm.DeletedAt `gorm:"index"`
//   }

// Product 商品表
type Product struct {
	ID          int64          `gorm:"column:id;primaryKey;autoIncrement"`
	Name        string         `gorm:"column:name;type:varchar(128);not null;index:idx_name"`
	Price       int            `gorm:"column:price;not null;default:0"`
	Stock       int            `gorm:"column:stock;not null;default:0"`
	CategoryID  int64          `gorm:"column:category_id;index"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	// 如果加这个就自带软删：
	// DeletedAt gorm.DeletedAt `gorm:"index"`
}

// 自定义表名（可选）—— 默认是 products
func (Product) TableName() string { return "t_product" }

// Category 分类表（做关联示例）
type Category struct {
	ID   int64  `gorm:"primaryKey;autoIncrement"`
	Name string `gorm:"type:varchar(64);not null"`
}

// ============================================================
// 2. 连接数据库
// ============================================================
func openDB() (*gorm.DB, error) {
	cfg := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 打印 SQL（学习阶段必开）
	}

	// ---- SQLite（无需安装，演示用）----
	// file::memory:?cache=shared  = 内存库，多 goroutine 共享
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), cfg)

	// ---- MySQL（生产用）----
	// username:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local
	// dsn := "root:root@tcp(127.0.0.1:3306)/go_learn?charset=utf8mb4&parseTime=True&loc=Local"
	// db, err := gorm.Open(mysql.Open(dsn), cfg)
	// --------------------------------------------------------

	if err != nil {
		return nil, fmt.Errorf("connect db failed: %w", err)
	}
	return db, nil
}

// ============================================================
// 3. CRUD 封装
// ============================================================
type ProductRepo struct {
	db *gorm.DB
}

func NewProductRepo(db *gorm.DB) *ProductRepo { return &ProductRepo{db: db} }

// Create 新增
func (r *ProductRepo) Create(p *Product) error {
	// DB.Create 自动填充 CreatedAt / UpdatedAt
	return r.db.Create(p).Error
}

// GetByID 按主键查（不存在时返回 gorm.ErrRecordNotFound）
func (r *ProductRepo) GetByID(id int64) (*Product, error) {
	var p Product
	err := r.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Search 条件查询：分页 + 排序 + 关键词 + 分类过滤
func (r *ProductRepo) Search(keyword string, catID int64, page, size int) ([]Product, int64, error) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	tx := r.db.Model(&Product{})
	if keyword != "" {
		tx = tx.Where("name LIKE ?", "%"+keyword+"%")
	}
	if catID > 0 {
		tx = tx.Where("category_id = ?", catID)
	}

	// 先 COUNT
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 再 LIMIT + OFFSET
	var list []Product
	err := tx.Order("id DESC").
		Limit(size).
		Offset((page - 1) * size).
		Find(&list).Error
	return list, total, err
}

// UpdatePrice 更新价格（指定字段，避免整行更新）
func (r *ProductRepo) UpdatePrice(id int64, price int) error {
	// 指定字段更新，推荐
	return r.db.Model(&Product{}).
		Where("id = ?", id).
		Update("price", price).Error
}

// Delete 删除（如果 Model 有 DeletedAt，这里会走软删）
func (r *ProductRepo) Delete(id int64) error {
	result := r.db.Delete(&Product{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("无记录被删除")
	}
	return nil
}

// DeductStock 扣库存（并发安全：用 gorm 写出 UPDATE ... WHERE stock >= N）
// 返回 受影响行数
func (r *ProductRepo) DeductStock(id int64, n int) (int64, error) {
	res := r.db.Model(&Product{}).
		Where("id = ? AND stock >= ?", id, n).
		Update("stock", gorm.Expr("stock - ?", n))
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

// ============================================================
// 4. 事务：TransferStock 把 N 件从 fromID 移到 toID（任何一步失败回滚）
// ============================================================
func (r *ProductRepo) TransferStock(fromID, toID int64, n int) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 扣 from
		rows := tx.Model(&Product{}).
			Where("id = ? AND stock >= ?", fromID, n).
			Update("stock", gorm.Expr("stock - ?", n)).RowsAffected
		if rows == 0 {
			return errors.New("源库存不足")
		}
		// 加 to
		rows = tx.Model(&Product{}).
			Where("id = ?", toID).
			Update("stock", gorm.Expr("stock + ?", n)).RowsAffected
		if rows == 0 {
			return errors.New("目标商品不存在")
		}
		// 返回 nil 就提交；返回非 nil 就回滚
		return nil
	})
}

// ============================================================
// 5. 钩子示例（BeforeCreate：创建前自动置时间戳 / 默认值）
// ============================================================
func (p *Product) BeforeCreate(tx *gorm.DB) error {
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = now
	}
	if p.Stock < 0 {
		return errors.New("stock 不能为负")
	}
	return nil
}

func (p *Product) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = time.Now()
	return nil
}

// ============================================================
// main
// ============================================================
func main() {
	db, err := openDB()
	if err != nil {
		log.Fatal(err)
	}

	// AutoMigrate：根据 struct 自动建表 / 加列（不会删列、不会改字段类型）
	err = db.AutoMigrate(&Product{}, &Category{})
	if err != nil {
		log.Fatalf("AutoMigrate 失败：%v", err)
	}
	fmt.Println("===> AutoMigrate 完成")

	repo := NewProductRepo(db)

	// ------------------------------------------------------------
	// 5.1 插入几条示例数据
	// ------------------------------------------------------------
	categories := []Category{
		{Name: "手机"}, {Name: "笔记本"}, {Name: "耳机"},
	}
	for i := range categories {
		db.Create(&categories[i])
	}
	products := []Product{
		{Name: "iPhone 15", Price: 6999, Stock: 100, CategoryID: 1},
		{Name: "小米 14", Price: 3999, Stock: 200, CategoryID: 1},
		{Name: "MacBook Pro", Price: 14999, Stock: 50, CategoryID: 2},
		{Name: "AirPods Pro", Price: 1599, Stock: 500, CategoryID: 3},
		{Name: "索尼 WH-1000XM5", Price: 2599, Stock: 300, CategoryID: 3},
	}
	for i := range products {
		if err := repo.Create(&products[i]); err != nil {
			log.Fatalf("插入失败：%v", err)
		}
	}
	fmt.Println("===> 预置数据完成，共 5 条商品 + 3 条分类")

	// ------------------------------------------------------------
	// 5.2 按主键查询
	// ------------------------------------------------------------
	p, err := repo.GetByID(1)
	if err != nil {
		log.Fatalf("GetByID 失败：%v", err)
	}
	fmt.Printf("===> 查 ID=1：%+v\n", *p)

	// 不存在的 ID
	_, err = repo.GetByID(9999)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("===> 查 ID=9999：gorm.ErrRecordNotFound（符合预期）")
	}

	// ------------------------------------------------------------
	// 5.3 条件查询 + 分页
	// ------------------------------------------------------------
	list, total, err := repo.Search("", 3, 1, 2) // 分类 3（耳机），第 1 页，页大小 2
	if err != nil {
		log.Fatalf("Search 失败：%v", err)
	}
	fmt.Printf("===> Search（耳机，第 1 页/页大小 2）total=%d list=%d 条\n", total, len(list))
	for _, it := range list {
		fmt.Printf("     - id=%d name=%s price=%d stock=%d\n", it.ID, it.Name, it.Price, it.Stock)
	}

	list, total, err = repo.Search("苹果", 0, 1, 10) // 模糊查询"苹果"
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("===> Search 关键词'苹果' total=%d\n", total)
	for _, it := range list {
		fmt.Printf("     - %s\n", it.Name)
	}

	// ------------------------------------------------------------
	// 5.4 更新价格 + 查回来
	// ------------------------------------------------------------
	if err := repo.UpdatePrice(1, 7299); err != nil {
		log.Fatal(err)
	}
	p, _ = repo.GetByID(1)
	fmt.Printf("===> ID=1 更新后 price = %d\n", p.Price)

	// ------------------------------------------------------------
	// 5.5 并发安全扣库存（经典 SQL：WHERE stock >= N）
	// ------------------------------------------------------------
	rows, err := repo.DeductStock(1, 5)
	fmt.Printf("===> ID=1 扣 5 件：受影响 %d 行（应当 1），库存剩 %d\n", rows, mustStock(repo, 1))

	rows, err = repo.DeductStock(1, 100000) // 扣不完
	fmt.Printf("===> ID=1 扣 100000 件：受影响 %d 行（应当 0）\n", rows)

	// ------------------------------------------------------------
	// 5.6 事务：库存转移
	// ------------------------------------------------------------
	fmt.Printf("===> 事务前：ID=1 stock=%d, ID=2 stock=%d\n", mustStock(repo, 1), mustStock(repo, 2))
	if err := repo.TransferStock(1, 2, 10); err != nil {
		log.Fatalf("事务失败：%v", err)
	}
	fmt.Printf("===> 事务后：ID=1 stock=%d, ID=2 stock=%d（1 少了 10，2 多了 10）\n", mustStock(repo, 1), mustStock(repo, 2))

	// ------------------------------------------------------------
	// 5.7 删除
	// ------------------------------------------------------------
	if err := repo.Delete(5); err != nil {
		log.Fatalf("删除失败：%v", err)
	}
	fmt.Println("===> 删除 ID=5（索尼耳机）成功")
	_, err = repo.GetByID(5)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		fmt.Println("===> 验证删除：ID=5 查不到")
	}

	fmt.Println("\n===> 所有示例完成，退出")
}

// mustStock 辅助函数（简化 main 的日志）
func mustStock(r *ProductRepo, id int64) int {
	p, err := r.GetByID(id)
	if err != nil {
		return -1
	}
	return p.Stock
}
