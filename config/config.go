package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 全局配置，由环境变量注入，避免密钥进入代码仓库
type Config struct {
	App AppConfig
	DB  DBConfig
	JWT JWTConfig
}

type AppConfig struct {
	Name string
	Port string
	Mode string // debug / release
	// AutoMigrate 是否在启动时用结构体同步表结构。
	// 默认关闭：正式交付以 SQL 脚本建表，自动迁移只作为开发期开关。
	AutoMigrate bool
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Charset  string
	MaxOpen  int
	MaxIdle  int
}

// DSN 生成 MySQL 连接串，必须带 parseTime 与 loc，否则时间字段会偏移
func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.DBName, d.Charset)
}

type JWTConfig struct {
	Secret string
	Expire time.Duration
	Issuer string
}

// Load 读取环境变量并给出适合本地开发的默认值
func Load() *Config {
	return &Config{
		App: AppConfig{
			Name: str("APP_NAME", "工业软件企业后台管理系统"),
			Port: str("APP_PORT", "8080"),
			Mode: str("APP_MODE", "debug"),
			// 默认 false，需要时显式设置 AUTO_MIGRATE=true
			AutoMigrate: boolean("AUTO_MIGRATE", false),
		},
		DB: DBConfig{
			Host:     str("DB_HOST", "127.0.0.1"),
			Port:     num("DB_PORT", 3306),
			User:     str("DB_USER", "root"),
			Password: str("DB_PASSWORD", ""),
			DBName:   str("DB_NAME", "industrial_admin"),
			Charset:  str("DB_CHARSET", "utf8mb4"),
			MaxOpen:  num("DB_MAX_OPEN", 20),
			MaxIdle:  num("DB_MAX_IDLE", 5),
		},
		JWT: JWTConfig{
			Secret: str("JWT_SECRET", "dev-only-secret-change-me"),
			Expire: time.Duration(num("JWT_EXPIRE_HOURS", 2)) * time.Hour,
			Issuer: str("JWT_ISSUER", "industrial_admin"),
		},
	}
}

func str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func num(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func boolean(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
