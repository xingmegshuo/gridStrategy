/***************************
@File        : exchange.go
@Time        : 2021/7/2 15:08
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 生成策略并开启
****************************/

package job

import (
	"context"
	"runtime"
	"time"
	grid "zmyjobs/corn/grid"
	logs "zmyjobs/corn/logs"

	model "zmyjobs/corn/models"
)

var log = logs.Log

// todo 删除策略不重新开始，删除策略判断是否清仓

// RunWG 生成用户策略
func RunWG() {
	//time.Sleep(time.Second)
	users := LoadUsers()
	for _, u := range *users {
		start := 0
	OuterLoop:
		for {
			select {
			case v := <-model.Ch:
				if v.Run == 2 || v.Run == 3 {
					if model.UpdateStatus(v.Id) == 10 {
						if u.ObjectId == int32(v.Id) {
							log.Printf("向用户%v发送暂停信息", u.ObjectId)
							grid.GridDone <- u.ObjectId
						}
					}
					break OuterLoop
				}
			default:
				time.Sleep(time.Millisecond * 100)
				// fmt.Println(u.Status, u.IsRun)
				if u.Status == 2 && model.UpdateStatus(u.ID) == int64(-1) && start == 0 {
					log.Println("符合要求", model.UpdateStatus(u.ID), u.ObjectId)
					for i := 1; i < 2; i++ {
						start = 1
						u = model.GetUserFromDB(u.ObjectId)
						log.Println("协程开始-用户:", u.ObjectId, "--交易币种:", u.Name, u.RealGrids, u.Base)
						go RunStrategy(u)
					}
					// } else if model.UpdateStatus(u.ID) == int64(100) && model.UpdateRun(u.ID) == 2 {
					// 	log.Println("等待重新开始", u.ObjectId)
					// 	u.IsRun = 99
					// 	u.RealGrids = "***"
					// 	u.Base = 0
					// 	u.RunCount++
					// 	u.Update()
					// 	model.UpdateBase(u.ObjectId)
					// 	time.Sleep(time.Second * 5)
					// 	model.AddRun(u.ObjectId, u.RunCount)
					// 	u.IsRun = -1
					// 	u.Update()
					// 	// u = model.UpdateUser(u)
					// 	log.Printf("用户%v重新开始;单数:%v;状态:%v;is_run:%v;实际买入信息:%v", u.ObjectId, u.Base, u.Status, u.IsRun, u.RealGrids)
					// 	runtime.Goexit()
					// go RunStrategy(u)
				} else {
					runtime.Gosched()
				}
				break OuterLoop
			}
		}
	}
}

// LoadUsers 获取数据库用户内容
func LoadUsers() *[]model.User {
	return model.GetUserJob()
}

// RunStrategy 开启策略
func RunStrategy(u model.User) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
OuterLoop:
	for {
		select {
		case id := <-grid.GridDone:
			if id == u.ObjectId {
				log.Println("收到消息,暂停策略,exiting ......", u.ObjectId)
				if model.UpdateStatus(u.ID) == 10 {
					u.IsRun = 2
					u.Update()
				}
				break OuterLoop
			}
		default:
			time.Sleep(time.Millisecond * 100)
			if model.UpdateStatus(u.ID) == -1 {
				u.IsRun = 10
				u.Update()
				for i := 0; i < 1; i++ {
					go grid.RunEx(ctx, u) //goex
				}
			} else {
				runtime.Gosched()
			}
		}
	}
}
