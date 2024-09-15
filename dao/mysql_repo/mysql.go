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
