// ============================================================
// 06_slices_maps.go  切片（slice）与 Map
// ------------------------------------------------------------
// 核心知识点：
//   1) 数组 [N]T（定长，少用）
//   2) 切片 []T（变长，最常用）：make / len / cap / append
//   3) 切片扩容原理 / 切片共享底层数组陷阱
//   4) Map：make / 取值 / 存在性判断 / 删除 / 遍历
//   运行：go run 06_slices_maps.go
// ============================================================

package main

import "fmt"

// ============================================================
// 一、数组（定长，值类型，几乎不用）
// ============================================================
func arrayDemo() {
	// 长度是类型的一部分：[3]int 和 [4]int 是两个不同类型
	var arr [3]int = [3]int{10, 20, 30}
	arr[0] = 100
	fmt.Println("数组 len=", len(arr), "内容=", arr)

	// 切片常常以数组作为底层，但我们平时只关心切片
}

// ============================================================
// 二、切片 []T（变长，最核心容器）
// ============================================================
func sliceCreate() {
	// 2.1 字面量初始化
	s := []int{1, 2, 3}
	fmt.Println("字面量切片 s=", s, "len=", len(s), "cap=", cap(s))

	// 2.2 用 make 创建：make([]T, len[, cap])
	s2 := make([]int, 5)       // len=5, cap=5，每个元素都是零值 0
	s3 := make([]int, 0, 100)  // len=0, cap=100，预分配容量避免多次扩容
	fmt.Printf("s2 len=%d cap=%d  s3 len=%d cap=%d\n", len(s2), cap(s2), len(s3), cap(s3))

	// 2.3 nil 切片 vs 空切片
	var sNil []int          // nil；长度 = 0，容量 = 0，没底层数组
	sEmpty := []int{}       // 空切片；长度 = 0，容量 = 0，有底层数组（空）
	sMake := make([]int, 0) // 空切片，和 sEmpty 等价
	fmt.Println("nil 切片 len=0? ", len(sNil) == 0, " 是nil吗？", sNil == nil)
	fmt.Println("空切片  len=0? ", len(sEmpty) == 0, " 是nil吗？", sEmpty == nil)
	fmt.Println("make空    len=0? ", len(sMake) == 0, " 是nil吗？", sMake == nil)
	// 实战经验：函数返回 slice，错误时直接 return nil, err
}

// 2.4 append：追加元素（自动扩容）
func sliceAppend() {
	var s []int            // nil 切片可以直接 append，无需初始化
	s = append(s, 1)       // [1]
	s = append(s, 2, 3, 4) // [1 2 3 4]
	fmt.Println("追加后 s=", s)

	// 追加另一个切片：用 ... 打散
	t := []int{5, 6, 7}
	s = append(s, t...)
	fmt.Println("追加 t 后 s=", s)
}

// 2.5 切片截取：与原切片共享底层数组（最常见陷阱）
func sliceCut() {
	arr := []int{0, 1, 2, 3, 4, 5}
	sub := arr[1:4] // [1 2 3]   start inclusive, end exclusive，cap 共享原数组剩余
	fmt.Printf("arr=%v  sub=%v  len(sub)=%d cap(sub)=%d\n", arr, sub, len(sub), cap(sub))

	// ★ 修改 sub 的元素，会直接改到底层数组，arr 也会变
	sub[0] = 99
	fmt.Println("改 sub[0]=99 后 arr =", arr) // [0 99 2 3 4 5]

	// 切片扩容后会新建数组，不再共享原数组
	sub = append(sub, 9999) // sub cap 还够，还是改原数组 → arr[4] 被覆盖
	fmt.Println("append 后的 sub=", sub, " arr=", arr)

	// 正确做法：需要独立副本时，用 copy() 新建
	a := []int{1, 2, 3}
	b := make([]int, len(a))
	copy(b, a)
	b[0] = 999
	fmt.Println("copy 后 a=", a, "b=", b) // a 不变
}

// 2.6 切片常用技巧
func sliceTips() {
	// 删除下标 i 位置的元素
	s := []int{1, 2, 3, 4, 5}
	i := 2 // 要删元素 "3"
	s = append(s[:i], s[i+1:]...) // 顺序敏感，会让后面元素移动，大数据慎用
	fmt.Println("删除 index=2 后 s =", s) // [1 2 4 5]

	// 弹出最后一个（模拟栈 pop）
	var last int
	last, s = s[len(s)-1], s[:len(s)-1]
	fmt.Println("pop last=", last, "剩余 s=", s)

	// 过滤：创建新 slice 保留需要的元素
	nums := []int{1, 2, 3, 4, 5, 6}
	even := make([]int, 0, len(nums))
	for _, v := range nums {
		if v%2 == 0 {
			even = append(even, v)
		}
	}
	fmt.Println("偶数 even =", even)
}

// ============================================================
// 三、Map（字典，map[K]V）
// ============================================================
func mapBasic() {
	// 3.1 创建
	// 方式 1：字面量
	m1 := map[string]int{
		"apple":  5,
		"banana": 10,
	}
	fmt.Println("m1 =", m1)

	// 方式 2：make
	m2 := make(map[string]string, 16) // 16 是初始容量建议值（非强制）
	m2["name"] = "Tom"
	m2["city"] = "Beijing"
	fmt.Println("m2 =", m2)

	// 3.2 取值
	fmt.Println("m1[apple] =", m1["apple"])

	// ★ 3.3 存在性判断（双赋值）—— 非常重要！
	// 因为 key 不存在时，取到的是 value 类型的零值（int 零值是 0），
	// 所以靠 m[k]==0 判断不可靠，必须用 ok 返回值
	v, ok := m1["orange"]
	fmt.Printf("m1[orange] v=%d ok=%v（不存在，零值=0，ok=false）\n", v, ok)

	if price, exists := m1["banana"]; exists {
		fmt.Println("香蕉的价格是", price)
	} else {
		fmt.Println("香蕉下架了")
	}

	// 3.4 删除
	delete(m1, "apple")
	fmt.Println("删除 apple 后 m1 =", m1)

	// 3.5 遍历：for-range 返回 (key, value)
	// 注意：map 的遍历顺序是随机的，不保证按插入顺序
	for k, v := range m2 {
		fmt.Printf("  %s → %s\n", k, v)
	}

	// 3.6 nil map：可以读（返回零值），不能写（会 panic）
	var mNil map[int]int
	// mNil[1] = 2   // 编译能过，运行 panic
	fmt.Println("读 nil map：mNil[1] =", mNil[1]) // 零值
}

// Map 常见场景：计数器
func mapCounter() {
	words := []string{"a", "b", "a", "c", "b", "a"}
	counts := make(map[string]int)
	for _, w := range words {
		// counts[w] 不存在时默认为零值 0，直接加就对
		counts[w]++
	}
	fmt.Println("词频统计 =", counts)
}

// ============================================================
// 主函数
// ============================================================
func main() {
	fmt.Println("===== 一、数组 =====")
	arrayDemo()

	fmt.Println("\n===== 二、切片 创建 =====")
	sliceCreate()
	fmt.Println("\n===== 二、切片 append =====")
	sliceAppend()
	fmt.Println("\n===== 二、切片截取 共享数组 =====")
	sliceCut()
	fmt.Println("\n===== 二、切片技巧 =====")
	sliceTips()

	fmt.Println("\n===== 三、Map 基础 =====")
	mapBasic()
	fmt.Println("\n===== 三、Map 计数器 =====")
	mapCounter()
}
