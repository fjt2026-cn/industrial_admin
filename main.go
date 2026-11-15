package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"industrial_admin/config"
	"industrial_admin/db"
	"industrial_admin/routes"
)

// 入口：加载配置 → 初始化数据库 → 注册路由 → 启动 HTTP 服务 → 优雅退出
func main() {
	cfg := config.Load()

	if err := db.Init(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("关闭数据库连接失败: %v", err)
		}
	}()

	// 自动迁移由 AUTO_MIGRATE 开关控制，默认关闭；正式交付以 SQL 脚本建表
	if cfg.App.AutoMigrate {
		if err := db.AutoMigrate(); err != nil {
			log.Printf("自动迁移失败（可改用 SQL 脚本建表）: %v", err)
		}
	}

	engine := routes.New(cfg)

	srv := &http.Server{
		Addr:           ":" + cfg.App.Port,
		Handler:        engine,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		log.Printf("%s 已启动，监听 :%s（mode=%s）", cfg.App.Name, cfg.App.Port, cfg.App.Mode)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	// 信号监听：Ctrl+C 或 kill 时先关闭监听，再等待在途请求结束
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	fmt.Println("收到退出信号，正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("优雅关闭超时，强制退出: %v", err)
	}
	log.Println("服务已退出")
}
