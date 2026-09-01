# 02_gorm —— GORM v1.25 ORM 入门

> 对标 Java MyBatis / Spring Data JPA。

## 运行

```powershell
$env:GOROOT = "d:\文件\test\tools\go"
$env:Path   = "d:\文件\test\tools\go\bin;$env:Path"
$env:GOPROXY = "https://goproxy.cn,direct"

cd d:\文件\test\go_learn\frameworks\02_gorm
go mod tidy
go run main.go
```

## 预期输出

```
===> AutoMigrate 完成
===> 预置数据完成
===> 查 ID=1：iPhone 15   price 6999
===> 查 ID=9999：gorm.ErrRecordNotFound（符合预期）
===> Search（耳机，第 1 页/页大小 2）total=2 list=2 条
===> Search 关键词'苹果' total=1
===> ID=1 更新后 price = 7299
===> ID=1 扣 5 件：受影响 1 行（应当 1）
===> ID=1 扣 100000 件：受影响 0 行（应当 0）
===> 事务前：ID=1 stock=95, ID=2 stock=200
===> 事务后：ID=1 stock=85, ID=2 stock=210
===> 删除 ID=5 成功
```

## 核心知识点

| 功能 | 代码位置 |
|------|---------|
| Model 定义 + 结构体标签（`gorm:"column:...`） | `type Product struct` |
| 自定义表名 | `func (Product) TableName()` |
| 连接数据库（sqlite 内存库） | `openDB()` |
| 切 MySQL | 代码注释中的 `dsn := "root..."`，取消注释即可 |
| 自动建表 / 加列 | `db.AutoMigrate(...)` |
| 新增（带钩子自动填时间） | `repo.Create` + `BeforeCreate` |
| 按主键查 | `First(&p, id)` + `gorm.ErrRecordNotFound` |
| 条件查询 + 分页 | `Where + Order + Limit + Offset + Count` |
| 指定字段更新 | `Model(...).Update("price", x)` |
| 并发安全扣库存 | `WHERE stock >= N` 写法 |
| 事务 | `db.Transaction(func(tx *gorm.DB) error {...})` |
| 钩子：BeforeCreate/BeforeUpdate | 类型方法上的钩子 |

## 常用操作速查（对照本示例修改即可）

```go
// ---- IN 查询 ----
db.Where("id IN ?", []int64{1, 2, 3}).Find(&list)

// ---- BETWEEN ----
db.Where("price BETWEEN ? AND ?", 1000, 5000).Find(&list)

// ---- 纯原生 SQL ----
db.Raw("SELECT id, name FROM t_product WHERE id = ?", 1).Scan(&p)

// ---- 更新多字段 struct（零值不更新！）----
db.Model(&p).Updates(Product{Name: "NewName", Price: 0})

// ---- 更新多字段 map（零值也会更新）----
db.Model(&Product{}).Where("id=?", 1).Updates(map[string]interface{}{
    "name": "xxx", "price": 0,
})

// ---- 聚合 ----
var count int64
db.Model(&Product{}).Where("category_id = ?", 1).Count(&count)
var avgPrice float64
db.Model(&Product{}).Select("AVG(price)").Scan(&avgPrice)

// ---- 分组 ----
type Row struct { CatID int64; Cnt int64; Avg float64 }
db.Model(&Product{}).Select("category_id as cat_id, count(*) cnt, avg(price) avg").
    Group("category_id").Scan(&rows)

// ---- N+1 问题解决方案（Preload 预加载）----
// 见官方文档：https://gorm.io/zh_CN/docs/preload.html
```
