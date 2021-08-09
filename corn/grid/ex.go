/***************************
@File        : ex.go
@Time        : 2021/07/28 15:19:24
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 使用goex
****************************/

package grid

import (
	"encoding/json"
	"fmt"
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/util"
	"zmyjobs/goex"

	"github.com/shopspring/decimal"
)

const (
	BuyM   = "buy"    // 市价买入
	BuyL   = "lbuy"   // 限价买入
	SellM  = "sell"   // 市价卖出
	SellL  = "selll"  // 限价卖出
	OpenDL = "opendl" // 开多
	OpenDM = "opendm" // 平多
	OpenLL = "openll" // 开空
	OpenLM = "openlM" // 平空
)

type Cliex struct {
	Ex     goex.API              // 现货
	Future goex.FutureRestAPI    // 期货
	symbol *model.SymbolCategory // 交易
}

// OneOrder 交易成功订单
type OneOrder struct {
	Amount   float64 // 数量
	Price    float64 // 价格
	Fee      float64 // 手续费
	OrderId  string  // id
	ClientId string  // 自定义id
	Type     string  // 类型
	Slide    string  // 买入还是卖出
	Cash     float64 // 现金
}

/*
@title        : NewEx
@desc         : 新建http客户端
@auth         : small_ant                   time(2021/08/03 11:07:13)
@param        : symbol  *model.SymbolCategory         `交易对信息`
@return       : cli *Cliex                             `cli对象`
*/

func NewEx(symbol *model.SymbolCategory) (cli *Cliex) {
	c := util.Config{Name: symbol.Category, APIKey: symbol.Key, Secreet: symbol.Secret,
		Host: symbol.Host, ClientID: symbol.Label}
	// fmt.Println(symbol.Future)
	if symbol.Future {
		cli = &Cliex{Future: util.NewFutrueApi(&c), symbol: symbol}
	} else {
		cli = &Cliex{Ex: util.NewApi(&c), symbol: symbol}
	}
	return
}

/**
 *@title        : GetAccount
 *dDesc         : 验证api获取账户资产
 *@auth         : small_ant                   time(2021/08/03 11:01:35)
 *@param        : 不需要参数                            ``
 *@return       : r/money/coin bool/decimal/decimal    `正确与否/余额/币种数量`
 */
func (c *Cliex) GetAccount() (r bool, money decimal.Decimal, coin decimal.Decimal) {
	if c.symbol.Future {
		acc, err := c.Future.GetFutureUserinfo()
		if err == nil {
			for _, u := range acc.FutureSubAccounts {
				if u.Currency.String() == c.symbol.QuoteCurrency {
					r = true
					money = decimal.NewFromFloat(u.CanEX)
				}
			}
		}
	} else {
		info, err := c.Ex.GetAccount()
		d := MakeCurrency(util.UpString(c.symbol.BaseCurrency))
		b := MakeCurrency(util.UpString(c.symbol.QuoteCurrency))
		if err == nil {
			r = true
			money = decimal.NewFromFloat(info.SubAccounts[b].Amount)
			coin = decimal.NewFromFloat(info.SubAccounts[d].Amount)
		}
	}
	return
}

/*
@title    	GetPrice
@desc   	获取当前交易对价格
@auth      	small_ant       时间（2019/6/18   10:57 ）
@param     	无
@return     price        	decimal.Decimal         "价格"
@return 	err 			error					"错误"
*/
func (c *Cliex) GetPrice() (price decimal.Decimal, err error) {
	symbol := c.MakePair()
	// fmt.Println(symbol)
	b, err := c.Ex.GetTicker(symbol)
	if err == nil {
		price = decimal.NewFromFloat(b.Last)
	}
	return
}

/*
@title        : Exchanges
@Desc         : 交易内容
@auth         : small_ant                   time(2021/08/03 10:28:03)
@param        : amount,price,name decimal/decimal/string                            `数量/价格/交易类型`
@return       : clientId,orderId,err string/string/error                            `自定义id/id/错误`
*/
func (c *Cliex) Exchanges(amount decimal.Decimal, price decimal.Decimal, name string) (string, string, error) {
	var (
		order *goex.Order
		err   error
	)
	symbol := c.MakePair()
	if c.symbol.Future {
		var FutureOrder *goex.FutureOrder
		num := 1
		switch name {
		case OpenDL:
			num = 1
		case OpenLL:
			num = 2
		case OpenLM:
			num = 3
		case OpenDM:
			num = 4
		}
		if c.symbol.QuoteCurrency == "USDT" {
			FutureOrder, err = c.Future.LimitFuturesOrder(symbol, goex.SWAP_CONTRACT, price.String(), amount.String(), num)
		} else {
			FutureOrder, err = c.Future.LimitFuturesOrder(symbol, goex.SWAP_USDT_CONTRACT, price.String(), amount.String(), num)
		}
		if err == nil {
			return FutureOrder.OrderID2, FutureOrder.ClientOid, err
		}
	} else {
		switch name {
		case BuyL:
			order, err = c.Ex.LimitBuy(amount.String(), price.String(), symbol)
		case SellL:
			order, err = c.Ex.LimitSell(amount.String(), price.String(), symbol)
		case BuyM:
			order, err = c.Ex.MarketBuy(amount.String(), price.String(), symbol)
		case SellM:
			order, err = c.Ex.MarketSell(amount.String(), price.String(), symbol)
		}
		if err == nil {
			return order.Cid, order.OrderID2, err
		}
	}

	// log.Println(amount, price, symbol, "交易信息")

	return "", "", err
}

/*
@title        : SearchOrder
@Desc         : 查找订单状态
@auth         : small_ant                   				time(2021/08/03 10:36:05)
@param        : orderId string                             `订单id`
@return       : b/status/order  bool/bool/*OneOrder         `是否查找到订单/订单是否结束/订单结束返回的需要数据`
*/
func (c *Cliex) SearchOrder(orderId string) (bool, bool, *OneOrder) {
	order, err := c.Ex.GetOneOrder(orderId, c.MakePair())
	var (
		o OneOrder
		// b goex.TradeStatus
	)
	// b = 2
	if err == nil {
		if order.Status == 2 {
			o.Amount = order.DealAmount
			o.Fee = order.Fee
			o.Price = order.AvgPrice
			o.ClientId = order.Cid
			o.OrderId = order.OrderID2
			o.Type = order.Type
			o.Slide = order.Side.String()
			o.Cash = order.CashAmount
			return true, true, &o
		} else {
			o.Amount = order.DealAmount
			o.Fee = order.Fee
			o.Price = order.AvgPrice
			o.ClientId = order.Cid
			o.OrderId = order.OrderID2
			o.Type = order.Type
			o.Slide = order.Side.String2()
			o.Cash = order.CashAmount
			return true, false, &o
		}
	} else {
		return false, false, nil
	}
}

/*
@title        : CancelOrder
@Desc         :	撤销订单
@auth         : small_ant                   time(2021/08/03 10:53:41)
@param        : orderId string                            `订单id`
@return       : b  bool                          `撤单成功或失败`
*/
func (c *Cliex) CancelOrder(orderId string) bool {
	b, err := c.Ex.CancelOrder(orderId, c.MakePair())
	if b && err == nil {
		return true
	}
	data := map[string]interface{}{}
	_ = json.Unmarshal([]byte(err.Error()), &data)
	if data["order-state"] == float64(7) {
		return true
	}
	return false
}

/*
@title        : MakePair
@Desc         : 创建交易对
@auth         : small_ant                   time(2021/08/03 10:58:40)
@param        : 无                            ``
@return       : symbol  goex.CurrencyPair     `交易对信息`
*/
func (c *Cliex) MakePair() goex.CurrencyPair {
	return goex.CurrencyPair{
		CurrencyA:      MakeCurrency(util.UpString(c.symbol.BaseCurrency)),
		CurrencyB:      MakeCurrency(util.UpString(c.symbol.QuoteCurrency)),
		AmountTickSize: int(c.symbol.AmountPrecision),
		PriceTickSize:  int(c.symbol.PricePrecision),
	}
}

/*
@title        : MakeCurrency
@Desc         : 创建一个币种
@auth         : small_ant                   time(2021/08/03 11:00:10)
@param        : name string                            `币种名称大写`
@return       : bi goex.Currency                        `币种`
*/
func MakeCurrency(name string) goex.Currency {
	return goex.Currency{Symbol: name, Desc: ""}
}

/**
 *@title        : FutureAccount
 *@desc         : 获取期货账户资金
 *@auth         : small_ant / time(2021/08/06 14:49:09)
 *@param        : t / string / `输入参数为u本位合约或币本位合约`
 *@return       : / / ``
 */
func (c *Cliex) FutureAccount(t string) {
	account, err := c.Future.GetFutureUserinfo()
	fmt.Println(account, err)
}
