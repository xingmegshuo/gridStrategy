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
	"time"
	model "zmyjobs/corn/models"
)

var (
	user = model.NewJob(model.ConfigMap["jobType1"], "LoadDB", "@every 5s")
	db   = model.UserDB
)

func UserJobRun() {
	userData()
	for i := 0; i < 1; i++ {
		go model.NewUser()
		go RunWG()
	}
}

func userData() {
	user.Count++
	user.UpdateJob()
	WriteCache("ZMYdb_task_order", time.Second*5)
	WriteCache("ZMYdb_task_api", time.Hour)
	WriteCache("ZMYdb_task_category", time.Hour)
}

// WriteCache mysql查询写入redis
func WriteCache(name string, t time.Duration) {
	// fmt.Println(model.CheckCache(name))
	// if !model.CheckCache(name) {
	var Data []map[string]interface{}
	switch name {
	case "ZMYdb_task_api":
		db.Raw("select apikey,secretkey,member_id,category_id from db_task_api").Scan(&Data)
	case "ZMYdb_task_category":
		db.Raw("select `id`,`name` from db_task_category").Scan(&Data)
	case "ZMYdb_task_order":
		db.Raw("select * from db_task_order").Scan(&Data)
		coin := map[string]interface{}{}
		for _, v := range Data {
			db.Raw("select `name`,`coin_type` from db_task_coin where `id` = ?", v["task_coin_id"]).Scan(&coin)
			v["task_coin_name"] = coin["name"]
			v["coin_type"] = coin["coin_type"]
		}
	}
	byteData, _ := json.Marshal(Data)
	// fmt.Println(name, string(byteData), Data, err)
	model.Del(name)
	model.SetCache(name, string(byteData), t)

}
