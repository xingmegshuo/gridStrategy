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
	Ex     goex.API
	symbol *model.SymbolCategory
}

func NewEx(symbol *model.SymbolCategory) *Cliex {
	return &Cliex{Ex: util.NewApi(&util.Config{APIKey: symbol.Key, Secreet: symbol.Secret, Host: symbol.Host}), symbol: symbol}
}

// GetAccount 获取账户信息验证api正确与否
func (c *Cliex) GetAccount() {
	// float64
	// fmt.Println(c.ex)
	info, _ := c.Ex.GetAccount()
	b := MakeCurrency(c.symbol.BaseCurrency)
	d := MakeCurrency(c.symbol.QuoteCurrency)
	fmt.Println(info.SubAccounts[b], info.SubAccounts[d])
	// return info.SubAccounts[b].Amount
}

// MakeCurrency 创造一个currency
func MakeCurrency(name string) goex.Currency {
	return goex.Currency{Symbol: name, Desc: ""}
}

// GetPrice 获取价格
func (c *Cliex) GetPrice() {
	// (decimal.Decimal, error)
	symbol := goex.CurrencyPair{
		CurrencyA:      MakeCurrency(c.symbol.QuoteCurrency),
		CurrencyB:      MakeCurrency(c.symbol.BaseCurrency),
		AmountTickSize: int(c.symbol.AmountPrecision),
		PriceTickSize:  int(c.symbol.PricePrecision),
	}
	fmt.Println(goex.BCC_BTC)
	fmt.Println(symbol)
	b, er := c.Ex.GetTicker(goex.BTC_USDT)
	fmt.Println(b, er, b.Last)
	ticker, err := c.Ex.GetDepth(2, symbol)
	fmt.Println("ticker:", ticker, err)
	t, e := c.Ex.GetTicker(symbol)
	fmt.Println(fmt.Scanf("%+v,%s,%d", t, e, t.Last))
	// if err != nil {
	// 	return 0, err
	// }
	// return ticker
}
