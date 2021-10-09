/***************************
@File        : init.go
@Time        : 2021/07/01 16:09:54
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 数据库初始化
****************************/

package model

import (
	"context"
	"database/sql"
	"time"
	logs "zmyjobs/corn/logs"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	DB      *gorm.DB
	UserDB  *gorm.DB
	CacheDB *redis.Client

	ctx         = context.Background()
	log         = logs.Log
	HuobiSymbol = GetSymbols("火币")
	BianSymbol  = GetSymbols("币安")
	BianFutureU = GetSymbols("币安u")
	BianFutureB = GetSymbols("币安b")
	OkexSymbol  = GetSymbols("ok")
	OkexFuture  = GetSymbols("okSwap")
)

var ConfigMap = map[string]string{
	"jobType1":    "间隔任务",
	"jobType2":    "限制任务",
	"jobType3":    "定时任务",
	"jobStatus1":  "正常运行",
	"jobStatus2":  "正常停止",
	"jobStatus3":  "意外中断",
	"hostStatus1": "正常访问",
	"hostStatus2": "失效",
}

// init 初始化
func init() {
	newLogger := logger.New(
		log,
		logger.Config{
			SlowThreshold: 2 * time.Second,
			LogLevel:      logger.Warn,
			Colorful:      false,
		},
	)

	// server
	serverDB, _ := sql.Open("mysql", "zmy:com1Chybay!@tcp(localhost:3306)/corn?charset=utf8mb4&parseTime=True&loc=Local")
	dsn := "ot_ptus209:2xHwO9bksHH@tcp(rm-j6cnwil9l9701sw92.mysql.rds.aliyuncs.com:3306)/ot_zhimayi?charset=utf8mb4&parseTime=True&loc=Local"
	rdb := redis.NewClient(&redis.Options{
		Addr:     "172.31.213.84:6379",
		Password: "lookupld",
		DB:       0,
	})
	serverDB.SetMaxIdleConns(128) //设置最大连接数
	serverDB.SetMaxOpenConns(128)
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: serverDB}), &gorm.Config{Logger: newLogger, PrepareStmt: true})

	if err != nil {
		panic("failed to connect database")
	}
	sqlDB, _ := sql.Open("mysql", dsn)
	sqlDB.SetMaxIdleConns(24) //设置最大连接数
	sqlDB.SetMaxOpenConns(24)
	sqlDB.SetConnMaxLifetime(time.Second * 600) // SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	userDB, e := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
	})
	if e != nil {
		panic("failed to connect user database")
	}
	UserDB = userDB
	DB = db

	db.AutoMigrate(&Job{}, &Host{}, &User{}, &RebotLog{})

	_, er := rdb.Ping(ctx).Result()
	if er != nil {
		log.Infof("连接redis出错，错误信息：%v", err)
	}
	CacheDB = rdb
}
