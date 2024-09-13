package test

import (
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/logger"
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/settings"
	"fmt"
	"go.uber.org/zap"
	"testing"
)

func TestUserSignup(t *testing.T) {
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

	params := models.ParamUserSignUp{
		Username:   "chan123",
		Password:   "chan123",
		RePassword: "chan123",
		Email:      "chenzhefan2001@163.com",
	}
	err := logic.SignUp(&params)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func TestUserSignin(t *testing.T) {
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

	params := models.User{
		Username: "chan123",
		Password: "chan123",
	}
	err := logic.SignInWithPassword(&params)
	if err != nil {
		fmt.Println(err)
		return
	}
}
