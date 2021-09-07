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
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/uti"
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
	Ex       goex.API              // 现货
	Future   goex.FutureRestAPI    // 期货
	symbol   *model.SymbolCategory // 交易对信息
	Currency goex.CurrencyPair     // 币种
}

// OneOrder 交易成功订单
type OneOrder struct {
	Amount   float64          // 数量
	Price    float64          // 价格
	Fee      float64          // 手续费
	OrderId  string           // id
	ClientId string           // 自定义id
	Type     string           // 类型
	Slide    string           // 买入还是卖出
	Cash     float64          // 现金
	Status   goex.TradeStatus // 状态
}

// NewFromOrder 现货订单
func NewFromOrder(order *goex.Order) *OneOrder {
	return &OneOrder{
		Amount:   order.DealAmount,
		Fee:      order.Fee,
		Price:    order.AvgPrice,
		ClientId: order.Cid,
		OrderId:  order.OrderID2,
		Type:     order.Type,
		Slide:    order.Side.String(),
		Cash:     order.CashAmount,
		Status:   order.Status,
	}
}

// 期货交易类型转换
func FutureTypeString(b interface{}, t bool) (r interface{}) {
	// 类型转数字
	if t {
		switch b.(string) {
		case OpenDL:
			r = 1
		case OpenDM:
			r = 3
		case OpenLL:
			r = 2
		case OpenLM:
			r = 4
		default:
			r = 0
		}
	} else {
		switch b.(int) {
		case 1:
			r = "开多"
		case 2:
			r = "开空"
		case 3:
			r = "平多"
		case 4:
			r = "平空"
		default:
			r = "未知"
		}
	}
	return
}

// NewFromFutureOrder 期货订单
func NewFromFutureOrder(order *goex.FutureOrder) *OneOrder {
	// fmt.Println(fmt.Sprintf("%+v", order))
	OrdType := "市价"
	if order.AlgoType == 1 {
		OrdType = "限价"
	}
	return &OneOrder{
		Amount:   order.DealAmount,
		Fee:      order.Fee,
		Price:    order.AvgPrice,
		OrderId:  order.OrderID2,
		Slide:    FutureTypeString(order.OType, false).(string),
		Cash:     order.Cash,
		Status:   order.Status,
		Type:     OrdType,
		ClientId: order.ClientOid,
	}
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
		Host: symbol.Host, ClientID: symbol.Label, Lever: symbol.Lever}
	if symbol.Future {
		cli = &Cliex{Future: util.NewFutrueApi(&c), symbol: symbol}
	} else {
		cli = &Cliex{Ex: util.NewApi(&c), symbol: symbol}
	}
	cli.Currency = cli.MakePair()
	// fmt.Println(cli.Currency, "生成的交易对")
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
	// fmt.Println(c.symbol.Future)
	if c.symbol.Future {
		var (
			p []goex.FuturePosition
		)
		acc, err := c.Future.GetFutureUserinfo(c.Currency)
		if err == nil {
			for _, u := range acc.FutureSubAccounts {
				if u.Currency.String() == c.symbol.QuoteCurrency {
					r = true
					money = decimal.NewFromFloat(u.CanEX)
				}
			}
		}
		if c.symbol.QuoteCurrency == "USDT" {
			p, err = c.Future.GetFuturePosition(c.Currency, goex.SWAP_USDT_CONTRACT)
		} else {
			p, err = c.Future.GetFuturePosition(c.Currency, goex.SWAP_CONTRACT)
		}
		if err == nil && len(p) > 0 {
			coin = decimal.NewFromFloat(p[0].BuyAmount)
		}
	} else {
		info, err := c.Ex.GetAccount()
		d := MakeCurrency(util.UpString(c.symbol.BaseCurrency))
		b := MakeCurrency(util.UpString(c.symbol.QuoteCurrency))
		if err == nil {
			r = true
			for _, account := range info.SubAccounts {
				// log.Printf("%T:%v;%T:%v,%v", k.Symbol, k.Symbol, d.Symbol, d.Symbol, k.Symbol == d.Symbol)
				if account.Currency.Symbol == d.Symbol {
					coin = decimal.NewFromFloat(account.Amount)
				}
				if account.Currency.Symbol == b.Symbol {
					money = decimal.NewFromFloat(account.Amount)
				}
				// fmt.Println(account)
			}
			// log.Printf("用户数据:%+v,%+v;%+v", info.SubAccounts[b], info.SubAccounts[d], info.SubAccounts)
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
	var b *goex.Ticker
	if c.symbol.Future {
		if c.symbol.QuoteCurrency == "USDT" {
			b, err = c.Future.GetFutureTicker(c.Currency, goex.SWAP_USDT_CONTRACT)
		} else {
			b, err = c.Future.GetFutureTicker(c.Currency, goex.SWAP_CONTRACT)
		}
	} else {
		b, err = c.Ex.GetTicker(c.Currency)
	}
	if err == nil {
		price = decimal.NewFromFloat(b.Last)
	}
	return
}

/**
 *@title        : Exchanges
 *@Desc         : 交易内容
 *@auth         : small_ant                   time(2021/08/03 10:28:03)
 *@param        : amount,price,name decimal/decimal/string                            `数量/价格/交易类型`
 *@return       : clientId,orderId,err string/string/error                            `自定义id/id/错误`
 */
func (c *Cliex) Exchanges(amount decimal.Decimal, price decimal.Decimal, name string, futureLimit bool) (*OneOrder, error) {
	var (
		order *goex.Order
		err   error
	)
	if c.symbol.Future {
		var FutureOrder *goex.FutureOrder
		if c.symbol.QuoteCurrency == "USDT" {
			if futureLimit {
				FutureOrder, err = c.Future.LimitFuturesOrder(c.Currency, goex.SWAP_USDT_CONTRACT, price.String(), amount.String(), FutureTypeString(name, true).(int))
			} else {
				FutureOrder, err = c.Future.MarketFuturesOrder(c.Currency, goex.SWAP_USDT_CONTRACT, amount.String(), FutureTypeString(name, true).(int))
			}
		} else {
			if futureLimit {
				FutureOrder, err = c.Future.LimitFuturesOrder(c.Currency, goex.SWAP_CONTRACT, price.String(), amount.String(), FutureTypeString(name, true).(int))
			} else {
				FutureOrder, err = c.Future.MarketFuturesOrder(c.Currency, goex.SWAP_CONTRACT, amount.String(), FutureTypeString(name, true).(int))
			}
		}
		if err == nil {
			return NewFromFutureOrder(FutureOrder), err
		}
	} else {
		switch name {
		case BuyL:
			order, err = c.Ex.LimitBuy(amount.String(), price.String(), c.Currency)
		case SellL:
			order, err = c.Ex.LimitSell(amount.String(), price.String(), c.Currency)
		case BuyM:
			order, err = c.Ex.MarketBuy(amount.String(), price.String(), c.Currency)
		case SellM:
			order, err = c.Ex.MarketSell(amount.String(), price.String(), c.Currency)
		}
		if err == nil {
			return NewFromOrder(order), err
		}
	}
	return &OneOrder{}, err
}

/*
@title        : SearchOrder
@Desc         : 查找订单状态
@auth         : small_ant                   				time(2021/08/03 10:36:05)
@param        : orderId string                             `订单id`
@return       : b/status/order  bool/bool/*OneOrder         `是否查找到订单/订单是否结束/订单结束返回的需要数据`
*/
func (c *Cliex) SearchOrder(orderId string) (bool, bool, *OneOrder) {
	var (
		o *OneOrder
		// b goex.TradeStatus
		err    error
		order  *goex.Order
		FOrder *goex.FutureOrder
	)
	// fmt.Println(c.symbol)
	if c.symbol.Future {
		// fmt.Println(c.symbol.QuoteCurrency)
		if c.symbol.QuoteCurrency == "USDT" {
			// fmt.Println("查找")
			if FOrder, err = c.Future.GetFutureOrder(orderId, c.Currency, goex.SWAP_USDT_CONTRACT); FOrder != nil {
				o = NewFromFutureOrder(FOrder)
			}
		} else {
			if FOrder, err = c.Future.GetFutureOrder(orderId, c.Currency, goex.SWAP_CONTRACT); FOrder != nil {
				o = NewFromFutureOrder(FOrder)
			}
		}

	} else {
		if order, err = c.Ex.GetOneOrder(orderId, c.Currency); order != nil {
			o = NewFromOrder(order)
		}
	}
	if err == nil {
		if o.Status == 2 {
			return true, true, o
		} else {
			return true, false, o
		}
	} else {
		// fmt.Println(err)
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
	var (
		b   bool
		err error
	)
	if c.symbol.Future {
		if c.symbol.QuoteCurrency == "USDT" {
			b, err = c.Future.FutureCancelOrder(c.Currency, goex.SWAP_USDT_CONTRACT, orderId)
		} else {
			b, err = c.Future.FutureCancelOrder(c.Currency, goex.SWAP_CONTRACT, orderId)
		}
	} else {
		b, err = c.Ex.CancelOrder(orderId, c.Currency)
	}
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
	if c.symbol.Future {
		currency := goex.NewCurrencyPair2(c.symbol.Symbol)
		return currency
	}
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
