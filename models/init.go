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
	"zmyjobs/logs"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var log = logs.Log

var DB *gorm.DB

var UserDB *gorm.DB

//var CacheDB
var ctx = context.Background()
var CacheDB *redis.Client

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
	serverDB, _ := sql.Open("mysql", "root:528012@tcp(localhost:3306)/corn?charset=utf8mb4&parseTime=True&loc=Local")

	// serverDB, _ := sql.Open("mysql", "zmy:com1Chybay!@tcp(localhost:3306)/corn?charset=utf8mb4&parseTime=True&loc=Local")

	serverDB.SetMaxIdleConns(2000) //设置最大连接数
	serverDB.SetMaxOpenConns(2000)
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: serverDB}), &gorm.Config{Logger: newLogger, PrepareStmt: true})

	// db, err := gorm.Open(sqlite.Open("config.db"), &gorm.Config{
	// 	SkipDefaultTransaction: true,
	// 	PrepareStmt:            true,
	// 	Logger:                 newLogger,
	// })

	if err != nil {
		panic("failed to connect database")
	}
	dsn := "root:528012@tcp(127.0.0.1:3306)/ot_zhimayi?charset=utf8mb4&parseTime=True&loc=Local"
	// dsn := "ot_ptus209:2xHwO9bksHH@tcp(rm-j6cnwil9l9701sw92.mysql.rds.aliyuncs.com:3306)/ot_zhimayi?charset=utf8mb4&parseTime=True&loc=Local"
	sqlDB, _ := sql.Open("mysql", dsn)
	sqlDB.SetMaxIdleConns(2000) //设置最大连接数
	sqlDB.SetMaxOpenConns(2000)
	sqlDB.SetConnMaxLifetime(time.Second * 600) // SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	userDB, e := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{
		Logger:      newLogger,
		PrepareStmt: true,
		// SkipDefaultTransaction: true,
	})
	if e != nil {
		panic("failed to connect user database")
	}
	UserDB = userDB
	// .Session(&gorm.Session{PrepareStmt: true})
	DB = db
	// .Session(&gorm.Session{PrepareStmt: true})

	db.AutoMigrate(&Job{}, &Host{}, &User{}, &RebotLog{})

	rdb := redis.NewClient(&redis.Options{
		Addr:     "127.0.0.1:6379",
		Password: "",
		DB:       0,
	})

	_, er := rdb.Ping(ctx).Result()
	if er != nil {
		log.Infof("连接redis出错，错误信息：%v", err)
	}
	CacheDB = rdb
}
