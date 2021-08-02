/***************************
@File        : ex.go
@Time        : 2021/07/28 15:19:24
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 使用goex
****************************/

package grid

import (
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/util"
	"zmyjobs/goex"

	"github.com/shopspring/decimal"
)

type Cliex struct {
	Ex     goex.API
	symbol *model.SymbolCategory
}

func NewEx(symbol *model.SymbolCategory) *Cliex {
	return &Cliex{Ex: util.NewApi(&util.Config{Name: symbol.Category, APIKey: symbol.Key, Secreet: symbol.Secret,
		Host: symbol.Host, ClientID: symbol.Label}), symbol: symbol}
}

// GetAccount 获取账户信息验证api正确与否
func (c *Cliex) GetAccount() (r bool, money decimal.Decimal, coin decimal.Decimal) {
	info, err := c.Ex.GetAccount()
	b := MakeCurrency(c.symbol.BaseCurrency)
	d := MakeCurrency(c.symbol.QuoteCurrency)
	r = false
	if err == nil {
		r = true
		money = decimal.NewFromFloat(info.SubAccounts[b].Amount)
		coin = decimal.NewFromFloat(info.SubAccounts[d].Amount)
	}
	return
}

// GetPrice 获取价格
func (c *Cliex) GetPrice() (price decimal.Decimal, err error) {
	symbol := c.MakePair()
	b, err := c.Ex.GetTicker(symbol)
	if err == nil {
		price = decimal.NewFromFloat(b.Last)
	}
	return
}

// MakePair 交易对
func (c *Cliex) MakePair() goex.CurrencyPair {
	return goex.CurrencyPair{
		CurrencyA:      MakeCurrency(c.symbol.QuoteCurrency),
		CurrencyB:      MakeCurrency(c.symbol.BaseCurrency),
		AmountTickSize: int(c.symbol.AmountPrecision),
		PriceTickSize:  int(c.symbol.PricePrecision),
	}
}

// MakeCurrency 创造一个currency
func MakeCurrency(name string) goex.Currency {
	return goex.Currency{Symbol: name, Desc: ""}
}
