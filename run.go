/***************************
@File        : main.go
@Time        : 2021/07/01 15:44:14
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : go corn jobs
****************************/

package main

import (
	"fmt"

	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"zmyjobs/logs"
	job "zmyjobs/task"
	"zmyjobs/xhttp"
)

var exitChan chan os.Signal
var l sync.Mutex

func exitHandle() {
	<-exitChan
	fmt.Println("接收到信号")
	l.Lock()
	job.Exit()
	l.Unlock()
	logs.Log.Info("退出程序")
	defer job.Exit()
	os.Exit(1) //如果ctrl+c 关不掉程序，使用os.Exit强行关掉
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	logs.NoneLog()
	logs.Log.Println("what fuck")
	exitChan = make(chan os.Signal)
	signal.Notify(exitChan, os.Interrupt, syscall.SIGTERM)
	go exitHandle()
	xhttp.RunServer()
	job.Init()
	job.C.Start()
	defer job.C.Stop()
	job.Wg.Wait()
}
