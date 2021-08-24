/***************************
@File        : jobs.go
@Time        : 2021/7/1 18:07
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package job

import (
	"sync"
	model "zmyjobs/corn/models"

	"github.com/robfig/cron"
)

var (
	C         = cron.New()
	Wg        sync.WaitGroup
	coinCount = 0
)

// Init 开始任务
func Init() {
	model.UserDB.Raw("select count(*) from db_task_coin").Scan(&coinCount)
	log.Println("start job")
	Wg.Add(1)
	// InitJob(*job,TestRun)
	InitJob(*user, UserJobRun)
	InitJob(*crawJob, CrawRun)
}

// Exit 退出任务
func Exit() {
	log.Println("job run over")
	JobExit(*user)
	JobExit(*crawJob)
	model.StopUser()
	//JobExit(*job)
}
