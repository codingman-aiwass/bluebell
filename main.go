package main

// @title bluebell forum project
// @version 1.0
// @description 一个简洁但功能完备的交流论坛
// @termsOfService http://swagger.io/terms/

// @contact.name John
// @contact.url http://www.swagger.io/support
// @contact.emails support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @Host localhost
// @BasePath :8080/
import (
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/logger"
	"bluebell/pkg/snowflake"
	"bluebell/routes"
	"bluebell/settings"
	"context"
	"fmt"
	"go.uber.org/zap"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	//1. 加载配置文件
	if err := settings.Init(); err != nil {
		fmt.Printf("load settings failed, err:%v\n", err)
		return
	}
	fmt.Println("load settings success")
	//2. 初始化日志
	if err := logger.Init(settings.GlobalSettings.LogCfg, settings.GlobalSettings.AppCfg.Mode); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	defer zap.L().Sync()
	zap.L().Debug("logger init success...")
	//3.初始化mysql
	if err := mysql_repo.InitDB(settings.GlobalSettings.MysqlCfg); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	defer mysql_repo.Close()
	// 初始化雪花算法
	if err := snowflake.Init(settings.GlobalSettings.AppCfg.StartTime,
		settings.GlobalSettings.AppCfg.MachineID); err != nil {
		fmt.Printf("init snowflake failed, err:%v\n", err)
	}
	//4.初始化redis
	if err := redis_repo.Init(settings.GlobalSettings.RedisCfg); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	defer redis_repo.CLose()

	//5.注册路由
	r := routes.SetupRouter(settings.GlobalSettings.AppCfg.Mode)

	//6.启动服务（优雅关机
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.GlobalSettings.AppCfg.Port),
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal(fmt.Sprintf("listen: %s\n", err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutdown Server ...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal(fmt.Sprintf("Server Shutdown:%v\n", err))
	}
	zap.L().Info("Server exiting")
}
