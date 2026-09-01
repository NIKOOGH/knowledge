// ============================================================
// frameworks/04_layered/internal/config/config.go  配置加载
// ------------------------------------------------------------
// 生产中会用 viper 读取 yaml；演示用简单 struct + 默认值
// ============================================================
package config

type Config struct {
	Server ServerConfig
	Redis  RedisConfig
}

type ServerConfig struct {
	Addr string
	Mode string // debug / release
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Enabled  bool // false 就不走缓存（演示降级）
}

// 默认配置（可通过环境变量覆盖：见 Load）
func Load() Config {
	return Config{
		Server: ServerConfig{Addr: ":18088", Mode: "release"},
		Redis: RedisConfig{Addr: "127.0.0.1:6379", DB: 0, Enabled: false},
	}
}
