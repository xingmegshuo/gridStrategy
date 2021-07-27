/***************************
@File        : log_test.go
@Time        : 2021/07/24 11:13:28
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 测试分红
****************************/

package model

import (
    "fmt"
    "testing"
)

func TestGetMoney(t *testing.T) {
    fmt.Println("testing start .....")
    DeleteRebotLog("b-0-1627365995")
}
