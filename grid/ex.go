/***************************
@File        : ex.go
@Time        : 2021/07/28 15:19:24
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 使用goex
****************************/

package grid

import (
	"fmt"
	model "zmyjobs/models"
	"zmyjobs/util"

	"github.com/nntaoli-project/goex"
)

type Cliex struct {
	ex goex.API
}

func NewEx(symbol *model.SymbolCategory) *Cliex {
	return &Cliex{ex: util.NewApi(&util.Config{APIKey: symbol.Key, Secreet: symbol.Secret, Host: symbol.Host})}
}

// GetAccount 获取账户信息验证api正确与否
func (c *Cliex) GetAccount() {
	// fmt.Println(c.ex)
	info, _ := c.ex.GetAccount()
	b := MakeCurrency("USDT")
	fmt.Println(info.SubAccounts[b].Amount)
}

// MakeCurrency 创造一个currency
func MakeCurrency(name string) goex.Currency {
	return goex.Currency{Symbol: name, Desc: ""}
}
