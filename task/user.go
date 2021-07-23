/***************************
@File        : userJob.go
@Time        : 2021/7/2 11:15
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 获取用户任务
****************************/

package job

import (
	"encoding/json"
	"runtime"
	"sync"
	"time"
	model "zmyjobs/models"
)

var user = model.NewJob(model.ConfigMap["jobType1"], "LoadDB", "@every 5s")

var db = model.UserDB

var updateCount sync.Mutex

func UserJobRun() {
	runtime.GOMAXPROCS(2)
	LoadUser()
}

// LoadUser 数据库读入缓存
func LoadUser() {
	userData()
	for i := 1; i < 2; i++ {
		go model.NewUser()
		go RunWG()
	}
}

func userData() {
	updateCount.Lock()
	user.Count++
	user.UpdateJob()
	WriteCache("db_task_order", time.Second*5)
	WriteCache("db_task_api", time.Second*5)
	WriteCache("db_task_category", time.Second*5)
	updateCount.Unlock()
}

// WriteCache mysql查询写入redis
func WriteCache(name string, t time.Duration) {
	if !model.CheckCache(name) {
		var Data []map[string]interface{}
		switch name {
		case "db_task_api":
			db.Raw("select apikey,secretkey,member_id,category_id from db_task_api").Scan(&Data)
		case "db_task_category":
			db.Raw("select `id`,`name` from db_task_category").Scan(&Data)
		case "db_task_order":
			db.Raw("select * from db_task_order").Scan(&Data)
			coin := ""
			for _, v := range Data {
				db.Raw("select `name` from db_task_coin where `en_name` = ?", v["task_coin_name"]).Scan(&coin)
				v["task_coin_name"] = coin
				// v["now_price"] = 0.0717
				// v["money"] = 300
			}
		}
		byteData, _ := json.Marshal(Data)
		model.SetCache(name, string(byteData), t)
	}
}
