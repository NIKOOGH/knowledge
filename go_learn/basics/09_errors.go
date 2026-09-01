// ============================================================
// 09_errors.go  错误处理
// ------------------------------------------------------------
// 核心知识点：
//   1) errors.New / fmt.Errorf 生成错误
//   2) if err != nil 检查模式
//   3) 自定义错误类型
//   4) errors.Is / errors.As 包装错误检查（Go 1.13+）
//   5) panic 与 recover（真正的"异常"，很少直接用）
//   运行：go run 09_errors.go
// ============================================================

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"
)

// ============================================================
// 1. 基本错误：errors.New + if err != nil
// ============================================================

// Login 通过用户名密码登录，成功返回 token 字符串，失败返回 error
func Login(username, password string) (string, error) {
	if username == "" {
		// errors.New 创建一个简单错误
		return "", errors.New("用户名不能为空")
	}
	if len(password) < 6 {
		// fmt.Errorf 用格式化字符串创建错误
		return "", fmt.Errorf("密码太短（%d < 6）", len(password))
	}
	if username == "admin" && password == "123456" {
		return "token-" + fmt.Sprint(time.Now().Unix()), nil
	}
	return "", errors.New("用户名或密码错误")
}

// 典型调用模式：
func loginDemo() {
	token, err := Login("", "123456")
	if err != nil {
		fmt.Println("登录失败：", err)
	} else {
		fmt.Println("登录成功，token =", token)
	}
	token, err = Login("admin", "123456")
	if err != nil {
		fmt.Println("登录失败：", err)
	} else {
		fmt.Println("登录成功，token =", token)
	}
}

// ============================================================
// 2. 包装错误（error wrapping）：fmt.Errorf + %w
// ============================================================
// 业务错误可以保留原始 cause，再增加上下文

// 先定义一个包级 sentinel error（全局哨兵错误，方便外部 Is 判断）
var ErrNotFound = errors.New("record not found")

// QueryDB 模拟查数据库，底层返回 ErrNotFound
func QueryDB(id int) (string, error) {
	if id <= 0 {
		return "", ErrNotFound
	}
	return "record_" + fmt.Sprintf("%d", id), nil
}

// QueryWithUser 封装：在底层错误上加上下文，%w 让包装错误可追溯
func QueryWithUser(id int, user string) (string, error) {
	record, err := QueryDB(id)
	if err != nil {
		// %w 保留原始 error（可被 errors.Is/As 解包），%v 只是转成字符串
		return "", fmt.Errorf("user=%s query id=%d failed: %w", user, id, err)
	}
	return record, nil
}

// 2.1 errors.Is：判断是否是某个 sentinel error
func isCheckDemo() {
	_, err := QueryWithUser(-1, "Alice")
	fmt.Println("err =", err)
	if errors.Is(err, ErrNotFound) {
		fmt.Println("确实是 ErrNotFound → 可以走 404 分支")
	} else {
		fmt.Println("不是 ErrNotFound → 可能是系统错误")
	}
}

// ============================================================
// 3. 自定义错误类型（实现 error 接口，只有一个方法 Error() string）
// ============================================================

// BizError 业务错误，包含错误码、消息、时间
type BizError struct {
	Code    int
	Message string
	When    time.Time
}

// 实现 error 接口唯一方法：Error() string
func (e *BizError) Error() string {
	return fmt.Sprintf("[%s] bizError %d: %s",
		e.When.Format("15:04:05"), e.Code, e.Message)
}

// NewBizError 构造自定义错误（返回指针类型）
func NewBizError(code int, msg string) *BizError {
	return &BizError{Code: code, Message: msg, When: time.Now()}
}

// doWork 可能返回 BizError 或普通 error
func doWork(userId int) error {
	if userId < 1 {
		return NewBizError(1001, "非法用户 ID")
	}
	if userId > 999 {
		return errors.New("服务内部错误")
	}
	return nil
}

// 3.1 errors.As：把 error 解包到自定义类型
func asCheckDemo() {
	err := doWork(-1)
	if err != nil {
		var biz *BizError
		if errors.As(err, &biz) {
			fmt.Printf("业务错误 code=%d msg=%s time=%v\n",
				biz.Code, biz.Message, biz.When)
		} else {
			fmt.Println("非业务错误，走系统错误流程：", err)
		}
	}

	err2 := doWork(9999)
	var biz2 *BizError
	if errors.As(err2, &biz2) {
		fmt.Println("这次也是 BizError，不应走到这里")
	} else {
		fmt.Println("这次走系统错误：", err2)
	}
}

// ============================================================
// 4. panic 与 recover（一般不用，只在不可恢复场景用）
// ============================================================
// Java 的 try-catch-throw 在 Go 里不要拿来当正常流程用。
// 正确理解：
//   panic() — 扔出异常（会向上一层层冒，沿途 defer 都执行完，还没人 recover 就崩）
//   recover() — 只能在 defer 函数里调用，捕获 panic 的值

func safeDivide(a, b int) (result int, err error) {
	// defer + 闭包捕获 panic
	defer func() {
		if r := recover(); r != nil {
			// recover 返回的 interface{} 是 panic() 传的那个参数
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	if b == 0 {
		panic("division by zero")
	}
	return a / b, nil
}

func panicRecoverDemo() {
	r, err := safeDivide(10, 0)
	if err != nil {
		fmt.Println("safeDivide 出错：", err, "返回值 r=", r) // r=0，err 非空
	}
	r, err = safeDivide(10, 3)
	fmt.Println("10/3 =", r, "err=", err)
}

// 生产使用建议：
//   - 应用入口（main / HTTP Handler 中间件）放 recover，防止单次请求 panic 把整个进程搞挂
//   - 业务代码不要 panic，正常用 error 返回
//   - 典型框架（Gin/Go-Zero）都内置了 recover 中间件

// ============================================================
// 5. 常见库错误判断小例子：os.IsNotExist / errors.Is
// ============================================================
func fileCheckDemo() {
	_, err := os.Open("/this/file/does/not/exist")
	if err != nil {
		// 旧写法（但不推荐对新版包装错误）
		if os.IsNotExist(err) {
			fmt.Println("文件确实不存在（os.IsNotExist）")
		}
		// 新写法：errors.Is(err, fs.ErrNotExist) 兼容 %w 包装
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Println("文件确实不存在（errors.Is fs.ErrNotExist）")
		}
	}
}

// ============================================================
// 主函数
// ============================================================
func main() {
	fmt.Println("===== 1. errors.New / if err != nil =====")
	loginDemo()

	fmt.Println("\n===== 2. 包装错误 + errors.Is =====")
	isCheckDemo()

	fmt.Println("\n===== 3. 自定义错误 + errors.As =====")
	asCheckDemo()

	fmt.Println("\n===== 4. panic + recover =====")
	panicRecoverDemo()

	fmt.Println("\n===== 5. 系统库错误判断 =====")
	fileCheckDemo()
}
