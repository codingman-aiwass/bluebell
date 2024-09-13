package mysql_repo

import (
	"bluebell/logger"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"bluebell/settings"
	_ "github.com/go-sql-driver/mysql" // 不要忘了导入数据库驱动
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var db *sqlx.DB

// Init and Close method for sqlx

//func Init(cfg *settings.MysqlConfig) (err error) {
//	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
//		cfg.User,
//		cfg.Password,
//		cfg.Host,
//		cfg.Port,
//		cfg.Database,
//	)
//	//dsn := "root:root@tcp(127.0.0.1:3306)/db1?charset=utf8mb4&parseTime=True"
//	db, err = sqlx.Connect("mysql", dsn)
//	if err != nil {
//		zap.L().Error("mysql connect error", zap.Error(err))
//		return
//	}
//	db.SetMaxOpenConns(cfg.MaxOpenConns)
//	db.SetMaxIdleConns(cfg.MaxIdleConns)
//	return
//}
//
//func Close() {
//	_ = db.Close()
//}

func InitDB(cfg *settings.MysqlConfig) (err error) {

	gormConf := &gorm.Config{
		Logger: logger.NewGormZapLogger(zap.L()),
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
			NoLowerCase:   false,
		},
	}
	if err = sqls.Open(cfg, gormConf, models.Models...); err != nil {
		zap.L().Error("InitDB error...", zap.Error(err))
	}
	return err
}

func Close() {
	sqls.Close()
}
