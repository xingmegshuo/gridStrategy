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
						grid.GridDone <- 1
					}
					break OuterLoop
				}
			default:
			}
			if u.Status == 2 && model.UpdateStatus(u.ID) == int64(-1) && start == 0 {
				// log.Println("符合要求", model.UpdateStatus(u.ID))
				for i := 1; i < 2; i++ {
					start = 1
					log.Println("协程开始-用户:", u.ObjectId, "--交易币种:", u.Name, u.Grids)
					go RunStrategy(u)
				}
			}
			// 循环策略进入
			if model.UpdateStatus(u.ID) == int64(100) && u.Status == 3 {
				log.Println("等待重新开始", u.ObjectId)
				u.IsRun = 99
				u.Base = 0
				u.RunCount++
				u.Update()
				time.Sleep(time.Second * 60)
				u.IsRun = -1
				model.AddRun(u.ObjectId, u.RunCount)
				u = model.UpdateUser(u)
				u.Update()
				log.Println("重新开始", u.ObjectId)
				runtime.Goexit()
			}
			break OuterLoop
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
		case <-grid.GridDone:
			log.Println("收到消息,暂停策略,exiting ......", u.ObjectId)
			if model.UpdateStatus(u.ID) == 10 {
				u.IsRun = 2
				u.Update()
			}
			break OuterLoop
		default:
		}
		// 执行任务不是一次执行
		if model.UpdateStatus(u.ID) == -1 {
			u.IsRun = 10
			u.Update()
			for i := 0; i < 1; i++ {
				go grid.RunEx(ctx, u) //goex
				// go grid.Run(ctx, u)
			}
		}
	}
}
