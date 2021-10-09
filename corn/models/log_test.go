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
	// CentsUser(100, 18, int64(20))
	UpdateBase(133)
	// LogStrategy("火币", "doge", 2, 2, 10, 2, false, 10)
	// var v = map[string]interface{}{}
	// c := UserDB.Raw("select id from db_task_category where name like ?", "火币").Scan(&v)
	// fmt.Println(v, c)
	// GotMoney(10, 13)
	// RunOver(2, 10, 33)
	// GetOldAmount(2)
	// var amount = map[string]interface{}{}
	// b := UserDB.Raw("select `meal_amount` from db_customer where id = ?", 1).Scan(&amount)
	// fmt.Println(amount, b)
	// a := GetAccount(float64(13))
	// fmt.Println(a)
}
