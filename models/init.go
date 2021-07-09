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
	"zmyjobs/logs"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
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
	serverDB, _ := sql.Open("mysql", "zmy:com1Chybay!@tcp(localhost:3306)/corn?charset=utf8mb4&parseTime=True&loc=Local")
	db, err := gorm.Open(mysql.New(mysql.Config{Conn: serverDB}), &gorm.Config{})
	// db, err := gorm.Open(sqlite.Open("config.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	// dsn := "root:528012@tcp(127.0.0.1:3306)/ot_zhimayi?charset=utf8mb4&parseTime=True&loc=Local"
	dsn := "ot_ptus209:2xHwO9bksHH@tcp(rm-j6cnwil9l9701sw92.mysql.rds.aliyuncs.com:3306)/ot_zhimayi?charset=utf8mb4&parseTime=True&loc=Local"
	sqlDB, _ := sql.Open("mysql", dsn)
	userDB, e := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{})
	if e != nil {
		panic("failed to connect user database")
	}
	UserDB = userDB
	DB = db
	db.AutoMigrate(&Job{}, &Host{}, User{})

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
