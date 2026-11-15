package db

import (
	"context"
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"industrial_admin/config"
	"industrial_admin/model"
)

// DB 是全局数据库句柄。
// 约定：只在 db 包内赋值，其他包一律通过 db.DB 使用，避免出现多处各自 Open 连接。
var DB *gorm.DB

// newLogger 构造 GORM 日志器，控制 SQL 打印级别与慢查询阈值。
// debug 模式打印全部 SQL，方便联调；release 模式只打印慢查询与错误，避免日志刷屏。
func newLogger(cfg *config.Config) gormlogger.Interface {
	level := gormlogger.Warn
	if cfg.App.Mode == "debug" {
		level = gormlogger.Info
	}
	return gormlogger.New(
		// 输出目标为标准输出，运行日志由部署环境统一收集
		log.New(os.Stdout, "[GORM] ", log.LstdFlags),
		gormlogger.Config{
			SlowThreshold: 200 * time.Millisecond, // 超过 200ms 记为慢查询
			LogLevel:      level,
			// 查询不到记录属于正常业务分支（如账号不存在），不打错误日志
			IgnoreRecordNotFoundError: true,
			Colorful:                  false, // 日志落文件时不带颜色控制符
		},
	)
}

// Init 读取配置建立 MySQL 连接，设置连接池并做一次连通性校验。
// 调用方（main.go）在返回错误时必须终止启动，避免带病运行。
func Init(cfg *config.Config) error {
	conn, err := gorm.Open(mysql.Open(cfg.DB.DSN()), &gorm.Config{
		Logger: newLogger(cfg),
		// 单条写入默认不包事务，需要原子性时在 service 层显式使用 Transaction
		SkipDefaultTransaction: true,
		// 统一使用本机时区生成时间戳，避免与 MySQL 时区不一致导致偏差
		NowFunc: func() time.Time { return time.Now().Local() },
	})
	if err != nil {
		return err
	}

	// 取出底层 *sql.DB，用于设置连接池参数
	sqlDB, err := conn.DB()
	if err != nil {
		return err
	}

	sqlDB.SetMaxOpenConns(cfg.DB.MaxOpen)      // 最大打开连接数，防止高并发压垮数据库
	sqlDB.SetMaxIdleConns(cfg.DB.MaxIdle)      // 最大空闲连接数，过小会导致频繁建连
	sqlDB.SetConnMaxLifetime(time.Hour)        // 连接最长存活 1 小时，规避 MySQL 主动断连
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲超过 10 分钟即释放

	// Ping 必须带超时：数据库不可达时不能把启动流程挂死
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return err
	}

	DB = conn
	log.Printf("数据库连接成功：%s@%s:%d/%s", cfg.DB.User, cfg.DB.Host, cfg.DB.Port, cfg.DB.DBName)
	return nil
}

// AutoMigrate 把 model 结构体同步为表结构，仅开发阶段使用。
// 正式交付以 industrial_admin.sql 建表：自动迁移不会补外键、索引注释，也无法回滚。
func AutoMigrate() error {
	return DB.AutoMigrate(
		&model.Dept{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RolePermission{},
		&model.SysConfig{},
		&model.OperationLog{},
	)
}

// Close 释放连接池，由 main.go 在优雅退出时调用。
func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
