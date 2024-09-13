package sqls

import (
	"bluebell/settings"
	"database/sql"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type GormModel struct {
	Id int64 `gorm:"primaryKey;autoIncrement" json:"id" form:"id"`
}

var (
	db    *gorm.DB
	sqlDB *sql.DB
)

func Open(dbConfig *settings.MysqlConfig, config *gorm.Config, models ...interface{}) (err error) {
	if config == nil {
		config = &gorm.Config{}
	}

	if config.NamingStrategy == nil {
		config.NamingStrategy = schema.NamingStrategy{
			TablePrefix:   "t_",
			SingularTable: true,
			NoLowerCase:   false,
		}
	}
	url := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbConfig.User,
		dbConfig.Password,
		dbConfig.Host,
		dbConfig.Port,
		dbConfig.Database)

	if db, err = gorm.Open(mysql.Open(url), config); err != nil {
		zap.L().Error("opens database failed: %s", zap.Error(err))
		return
	}

	if sqlDB, err = db.DB(); err == nil {
		sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
		sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	} else {
		zap.L().Error("init database settings error", zap.Error(err))
	}

	if err = db.AutoMigrate(models...); nil != err {
		zap.L().Error("auto migrate tables failed: %s", zap.Error(err))
	}
	return
}

func DB() *gorm.DB {
	return db
}

func Close() {
	if sqlDB == nil {
		return
	}
	if err := sqlDB.Close(); nil != err {
		zap.L().Error("Disconnect from database failed: %s", zap.Error(err))
	}
}
