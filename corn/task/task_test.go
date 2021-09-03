/***************************
@File        : task_test.go
@Time        : 2021/08/10 16:54:47
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 抓取货币价格测试
****************************/

package job

import (
    "fmt"
    "testing"
    "time"
)

func TestCraw(t *testing.T) {

    go Begin()
    go func() {
        for {
            time.Sleep(time.Second * 5)
            fmt.Println("发送关闭")
            Stop <- 1
            time.Sleep(time.Second * 5)
        }
    }()
    for {
        ;
    }

}
