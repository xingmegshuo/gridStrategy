/***************************
@File        : log.go
@Time        : 2021/7/8 11:09
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package logs

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	// now := time.Now()
	logFilePath := ""
	if dir, err := os.Getwd(); err == nil {
		logFilePath = dir + "/logFile/"
	}
	if err := os.MkdirAll(logFilePath, 0777); err != nil {
		fmt.Println(err.Error())
	}
	logFileName := "log.log"

	//写入文件
	// src, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	writer, err := rotatelogs.New(
		logFilePath+"%Y-%m-%d.log",
		rotatelogs.WithLinkName(logFilePath+logFileName),
		rotatelogs.WithMaxAge(15*24*time.Hour),
		rotatelogs.WithRotationTime(time.Hour*24),
	)

	if err != nil {
		fmt.Println("err", err)
	}
	//设置输出
	Log = logrus.New()

	Log.Out = writer
	//设置日志级别
	// Log.SetLevel(logrus.DebugLevel)
	//设置日志格式

	Log.SetReportCaller(true)

	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			//处理文件名
			fileName := path.Base(frame.File) + fmt.Sprintf(":%d", frame.Line)
			return frame.Function, fileName
		},
	})
}

func NoneLog() {
	Log.Println("─────────▄──────────────▄──── ")
	Log.Println("─ wow ──▌▒█───────────▄▀▒▌─── ")
	Log.Println("────────▌▒▒▀▄───────▄▀▒▒▒▐─── ")
	Log.Println("───────▐▄▀▒▒▀▀▀▀▄▄▄▀▒▒▒▒▒▐───")
	Log.Println("─────▄▄▀▒▒▒▒▒▒▒▒▒▒▒█▒▒▄█▒▐───")
	Log.Println("───▄▀▒▒▒▒▒▒ such difference ─")
	Log.Println("──▐▒▒▒▄▄▄▒▒▒▒▒▒▒▒▒▒▒▒▒▀▄▒▒▌──")
	Log.Println("──▌▒▒▐▄█▀▒▒▒▒▄▀█▄▒▒▒▒▒▒▒█▒▐──")
	Log.Println("─▐▒▒▒▒▒▒▒▒▒▒▒▌██▀▒▒▒▒▒▒▒▒▀▄▌─")
	Log.Println("─▌▒▀▄██▄▒▒▒▒▒▒▒▒▒▒▒░░░░▒▒▒▒▌─")
	Log.Println("─▌▀▐▄█▄█▌▄▒▀▒▒▒▒▒▒░░░░░░▒▒▒▐─")
	Log.Println("▐▒▀▐▀▐▀▒▒▄▄▒▄▒▒▒ electrons ▒▌")
	Log.Println("▐▒▒▒▀▀▄▄▒▒▒▄▒▒▒▒▒▒░░░░░░▒▒▒▐─")
	Log.Println("─▌▒▒▒▒▒▒▀▀▀▒▒▒▒▒▒▒▒░░░░▒▒▒▒▌─")
	Log.Println("─▐▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▐──")
	Log.Println("──▀ amaze ▒▒▒▒▒▒▒▒▒▒▒▄▒▒▒▒▌──")
	Log.Println("────▀▄▒▒▒▒▒▒▒▒▒▒▄▄▄▀▒▒▒▒▄▀───")
	Log.Println("───▐▀▒▀▄▄▄▄▄▄▀▀▀▒▒▒▒▒▄▄▀─────")
	Log.Println("──▐▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▒▀▀────────")
}
