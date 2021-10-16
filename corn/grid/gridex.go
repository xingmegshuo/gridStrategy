package grid

import (
	"context"
	"errors"
	"fmt"
	"time"
	model "zmyjobs/corn/models"

	"github.com/shopspring/decimal"
)

type ExTrader struct {
	arg       *model.Args     // 策略参数
	grids     []model.Grid    // 预设策略
	RealGrids []model.Grid    // 成交信息
	base      int             // 当前单数
	ErrString string          // 错误信息
	over      bool            // 是否结束
	automatic bool            // 是否用户主动清仓
	u         model.User      // 用户
	pay       decimal.Decimal // 投入金额
	cost      decimal.Decimal // average price
	amount    decimal.Decimal // amount held
	last      decimal.Decimal // 上次交易价格
	basePrice decimal.Decimal // 第一次交易价格
	hold      decimal.Decimal // 余额
	SellOrder map[string]int  // 卖出订单号
	SellMoney decimal.Decimal // 卖出金额
	OrderOver bool            // 一次订单是否结束
	goex      *Cliex          // goex
	canBuy    bool            // 是否可以买入
	centMoney bool            // 是否可以分红

}

// 策略预览内容
type Grid struct {
	Id         int
	Price      decimal.Decimal // 价格
	AmountBuy  decimal.Decimal // 买入数量
	Decline    float64         // 跌幅,或者说盈利
	AmountSell decimal.Decimal // 卖出数量
	TotalBuy   decimal.Decimal // money
}

// log 交易日志
func (t *ExTrader) log(orderId string, price decimal.Decimal, ty string, num int,
	hold decimal.Decimal, rate float64, buy string) {
	p, _ := price.Float64()
	h, _ := hold.Float64()
	pay, _ := price.Mul(hold).Float64()
	money, _ := price.Mul(hold).Float64()
	var b, q string
	if buy == "买入" {
		b = t.goex.symbol.QuoteCurrency
		q = t.goex.symbol.BaseCurrency
	} else {
		q = t.goex.symbol.BaseCurrency
		b = t.goex.symbol.QuoteCurrency
		pay = 0
	}
	info := model.RebotLog{
		BaseCoin:  b,
		GetCoin:   q,
		OrderId:   orderId,
		Price:     p,
		UserID:    uint(t.u.ObjectId),
		Types:     ty,
		AddNum:    num,
		HoldNum:   h,
		HoldMoney: money,
		PayMoney:  pay,
		AddRate:   rate,
		BuyOrSell: buy,
		Status:    "创建订单",
		// AccountMoney: account,
		Category: t.goex.symbol.Category,
		Custom:   uint(t.u.Custom),
	}
	info.New()
}

// 初始化
func (t *ExTrader) ReBalance(ctx context.Context) error {
	t.base = t.u.Base
	t.amount = t.CountHold()
	moneyNeed := t.CountNeed()
	if len(t.RealGrids) == 0 {
		t.base = 0
	}
	for i := 0; i < len(t.RealGrids); i++ {
		if i < t.base {
			t.SellMoney = t.SellMoney.Add(t.RealGrids[i].AmountSell)
		}
	}
	t.pay = t.CountPay()
	log.Printf("数量:%v;用户:%v;需要:%v,投入金额:%v", t.amount, t.u.ObjectId, moneyNeed, t.pay)

	moneyNeed = decimal.Decimal{}
	b, moneyHeld, coinHeld := t.goex.GetAccount()
	if b {
		log.Printf("账户余额: %s, 币种:  %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
		if moneyNeed.Cmp(moneyHeld) == 1 {
			return errors.New("no enough money")
			// return nil
		}

	} else {
		b, moneyHeld, coinHeld = t.goex.GetAccount()
		if b {
			log.Printf("账户余额: %s, 币种:  %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
			if moneyNeed.Cmp(moneyHeld) == 1 {
				return errors.New("no enough money")
				// return nil
			}
		} else {
			return errors.New("account or api error")
			// return nil
		}
	}
	return nil
}

// CountBuy 计算买入数量
func (t *ExTrader) CountBuy() (amount decimal.Decimal) {
	for _, b := range t.RealGrids {
		amount = amount.Add(b.AmountBuy)
	}
	return
}

// CountNeed 计算需要资金
func (t *ExTrader) CountNeed() (moneyNeed decimal.Decimal) {
	for i := 0; i < len(t.grids); i++ {
		if i >= t.base {
			moneyNeed = moneyNeed.Add(t.grids[i].TotalBuy)
		}
	}
	return
}

/*
	 @title   : buy
	 @desc    : 买入操作
	 @auth    : small_ant / time(2021/08/12 10:06:40)
	 @param   : price,amount,rate / decimal,decimal,float64 / `价格,数量,跌幅`
	 @return  :  / string,string,err / `orderid，clientid，错误`
**/
func (t *ExTrader) buy(price, amount decimal.Decimal, rate float64) (*OneOrder, error) {
	orderType := BuyL
	msg := "限价买入"
	if t.arg.OrderType == 2 {
		orderType = BuyM
		if t.goex.symbol.Category == "火币" {
			amount = t.grids[t.base].TotalBuy
		}
		msg = "市价买入"
	}
	// log.Println(t.arg.Crile)
	if t.arg.Crile == 5 || t.arg.Crile == 3 {
		orderType = OpenDL
		if t.u.Future == 2 || t.u.Future == 4 {
			amount = t.grids[t.base].TotalBuy
		} else {
			amount = t.grids[t.base].AmountBuy
		}
		msg = "开多"
	} else if t.arg.Crile == 6 || t.arg.Crile == 4 {
		orderType = OpenLL
		if t.u.Future == 2 || t.u.Future == 4 {
			amount = t.grids[t.base].TotalBuy
		} else {
			amount = t.grids[t.base].AmountBuy
		}
		msg = "开空"
	}
	if t.goex.symbol.Category == "OKex" && t.u.Future > 0 {
		amount = t.grids[t.base].Mesure
	}
	// amount = decimal.Decimal{}
	o, err := t.goex.Exchanges(amount, price, orderType, false)
	clientId, orderId := o.ClientId, o.OrderId
	log.Printf("[Order][buy] 价格: %s, 数量: %s, 用户:%d,订单号:%v,自定义订单号:%v,类型:%s", price, amount, t.u.ObjectId, orderId, clientId, msg)
	// 增加一个真实成交
	if err == nil {
		log.Printf("买入下单返回的数据用户%v[%v][%v]:amount:%v,price:%v", t.u.ObjectId, o.Slide, o.Type, o.Amount, o.Price)
		t.log(orderId, price, msg, t.base, amount, rate, "买入")
		g := model.Grid{
			Id:      t.base + 1,
			Decline: rate,
			Order:   orderId,
		}
		if t.u.Future > 0 {
			g.Mesure = t.grids[t.base].Mesure
			g.AmountBuy = t.grids[t.base].AmountBuy
		}
		t.RealGrids = append(t.RealGrids, g)
	}
	return o, err
}

// 卖出
func (t *ExTrader) sell(price, amount decimal.Decimal, rate float64, n int) (*OneOrder, error) {
	orderType := SellL
	msg := "限价卖出"
	if t.arg.OrderType == 2 {
		orderType = SellM
		price = decimal.Decimal{}
		msg = "市价卖出"
	}
	if t.arg.Crile == 5 || t.arg.Crile == 3 {
		msg = "平多"
		orderType = OpenDM
	} else if t.arg.Crile == 6 || t.arg.Crile == 4 {
		orderType = OpenLM
		msg = "平空"
	}
	if t.u.Future == 2 || t.u.Future == 4 || t.goex.symbol.Category == "OKex" && t.u.Future > 0 {
		amount = t.CountMesure()
	}
	log.Printf("类型:%v,数量:%v,价格:%v;用户:%v", msg, amount, price, t.u.ObjectId)
	o, err := t.goex.Exchanges(amount, price, orderType, false)
	clientId, orderId := o.ClientId, o.OrderId
	log.Printf("[Order][sell] 价格: %s, 数量: %s, 用户:%d,订单号:%v,自定义订单号:%v", price, amount, t.u.ObjectId, orderId, clientId)
	// 增加一个真实成交
	if err == nil {
		log.Printf("卖出下单返回的数据用户%v[%v]:amount:%v,price:%v", t.u.ObjectId, o.Type, o.Amount, o.Price)
		t.RealGrids[n].Decline = rate
		t.log(orderId, price, msg, n, amount, rate, "卖出")
		g := map[string]int{orderId: n}
		t.SellOrder = g
	}
	return o, err
}

// SearchOrder 查询订单
func (t *ExTrader) SearchOrder(clientOrderId string, client string) bool {
	// log.Println("查询订单", client, "依据此查找:", clientOrderId)
	has, over, order := t.goex.SearchOrder(clientOrderId)
	if has && over {
		t.ParseOrder(order)
		t.Tupdate()
		log.Printf("实际操作%+v", t.RealGrids)
		return true
	}
	return false
}

// ParseOrder
func (t *ExTrader) ParseOrder(order *OneOrder) {
	price := decimal.NewFromFloat(order.Price)
	amount := decimal.NewFromFloat(order.Amount)
	fee := decimal.NewFromFloat(order.Fee)
	if t.u.Future == 2 || t.u.Future == 4 {
		if t.u.Category != "OKex" {
			amount = decimal.NewFromFloat(order.Cash)
		}
	}

	t.hold = t.myMoney()
	log.Printf("订单成功--- 价格:%v  数量: %v  手续费: %v 成交额: %v 订单号: %v", order.Price, order.Amount, order.Fee, order.Cash, order.OrderId)
	if b, ok := t.SellOrder[order.OrderId]; ok {
		log.Printf("当前单数:%v,卖出单数:%v;%v", t.base, b, t.SellOrder)
		if t.goex.symbol.Category == "OKex" && t.u.Future == 1 {
			if b == 0 {
				t.RealGrids[b].AmountSell = t.CountHold().Mul(price)
			} else {
				t.RealGrids[b].AmountSell = t.RealGrids[b].AmountBuy.Mul(price)
			}
		} else {
			sellMoney := price.Mul(amount).Abs().Sub(fee)
			t.SellMoney = t.SellMoney.Add(sellMoney) // 卖出钱
			t.RealGrids[b].AmountSell = sellMoney    // 修改卖出
			// 币本位记录卖出币种数量
			if t.u.Future == 2 || t.u.Future == 4 {
				t.RealGrids[b].AmountSell = decimal.NewFromFloat(order.Cash)
			}
		}
		t.amount = t.CountHold()
		t.pay = t.CountPay()
		model.RebotUpdateBy(order.OrderId, price, t.RealGrids[b].AmountBuy, fee, t.RealGrids[b].TotalBuy, t.hold, "成功", order.ClientId, t.u.Future, t.arg.CoinId)
		if b == 0 {
			t.over = true
			model.AsyncData(t.u.ObjectId, 0.00, 0.00, 0.00, 0)
		} else {
			model.AsyncData(t.u.ObjectId, t.amount, t.pay.Div(amount), t.pay, b)
		}
	} else {
		log.Printf("当前单数:%v,用户:%v", t.base, t.u.ObjectId)
		if t.goex.symbol.Category != "OKex" && t.u.Future <= 0 {
			t.RealGrids[t.base].AmountBuy = amount.Sub(fee)
		}
		t.RealGrids[t.base].Price = price
		t.RealGrids[t.base].TotalBuy = price.Mul(amount.Sub(fee))
		if t.u.Future == 1 || t.u.Future == 3 {
			t.RealGrids[t.base].TotalBuy = decimal.NewFromFloat(order.Cash)
		}
		if t.u.Future == 2 || t.u.Future == 4 {
			t.RealGrids[t.base].Mesure = t.RealGrids[t.base].AmountBuy
			t.RealGrids[t.base].AmountBuy = amount.Abs()
		}
		if t.goex.symbol.Category == "OKex" && t.u.Future > 0 {
			t.RealGrids[t.base].TotalBuy = t.RealGrids[t.base].AmountBuy.Mul(price)
			if t.u.Future == 2 {
				t.RealGrids[t.base].AmountBuy = amount.Abs().Mul(decimal.NewFromFloat(10)).Div(t.RealGrids[t.base].Price)
				t.RealGrids[t.base].TotalBuy = amount.Abs().Mul(decimal.NewFromFloat(10))
			}

		}
		if t.goex.symbol.Category == "OKex" && t.u.Future == 0 {
			t.RealGrids[t.base].TotalBuy = decimal.NewFromFloat(order.Cash)
			t.RealGrids[t.base].AmountBuy = decimal.NewFromFloat(order.Amount).Sub(decimal.NewFromFloat(order.Fee).Abs())
			t.RealGrids[t.base].Price = decimal.NewFromFloat(order.Price)
		}
		t.amount = t.CountHold()
		t.pay = t.CountPay()
		t.cost = t.CostPrice()
		model.RebotUpdateBy(order.OrderId, price, t.RealGrids[t.base].AmountBuy, fee, t.RealGrids[t.base].TotalBuy, t.hold, "成功", order.ClientId, t.u.Future, t.arg.CoinId)
		model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
	}
}

// 获取当前持仓数量
func (t *ExTrader) CountHold() (amount decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			amount = amount.Add(g.AmountBuy)
		}
	}
	log.Printf("用户%v计算持仓量:%v", t.u.ObjectId, amount)
	return
}

// 平均价格
func (t *ExTrader) CostPrice() (amount decimal.Decimal) {
	num := 1
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			amount = amount.Add(g.Price)
			num += 1
		}
	}
	if num-1 == 0 {
		num = 1
	} else {
		num = num - 1
	}
	amount = amount.Div(decimal.NewFromInt(int64(num)))
	log.Printf("用户%v计算平均价格:%v", t.u.ObjectId, amount)
	return
}

// 获取当前投入金额
func (t *ExTrader) CountPay() (pay decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			pay = pay.Add(g.TotalBuy)
		}
	}
	log.Printf("用户%v计算投入金额:%v", t.u.ObjectId, pay)
	return
}

// CalCulateProfit 计算盈利
func (t *ExTrader) CalCulateProfit() decimal.Decimal {
	var pay, my decimal.Decimal
	for _, b := range t.RealGrids {
		pay = pay.Add(b.TotalBuy)
		my = my.Add(b.AmountSell)
	}
	// 币本位的盈利计算
	if t.u.Future == 2 || t.u.Future == 4 {
		pay = decimal.NewFromFloat(0)
		my = my.Abs()
		for _, b := range t.RealGrids {
			pay = pay.Add(b.AmountBuy)
		}
		log.Printf("用户%v投入币种:%v;清仓获得币种:%v", t.u.ObjectId, pay, my)
		if t.arg.Crile == 4 || t.arg.Crile == 6 {
			return my.Sub(pay).Mul(t.last)
		} else {
			return pay.Sub(my).Mul(t.last)
		}
	}
	// 正常u本位和现货盈利计算
	log.Printf("用户%v投入资金:%v;清仓获得资金:%v", t.u.ObjectId, pay, my)
	if t.arg.Crile == 4 || t.arg.Crile == 6 {
		return pay.Sub(my)
	}
	return my.Sub(pay)
}

func (t *ExTrader) WaitOrder(orderId string, cli string) bool {
	log.Println("等待订单交易.......", t.u.ObjectId, orderId)
	start := time.Now()
	for {
		if t.SearchOrder(orderId, cli) {
			return true
		}
		if time.Since(start) >= time.Minute*30 {
			fmt.Println(fmt.Sprintf("用户%v下单id%v无法找到", t.u.ObjectId, orderId))
			return false
		} else {
			time.Sleep(time.Second * 2)
			continue
		}
	}
}

// WaitSell 卖出等待
func (t *ExTrader) WaitSell(price decimal.Decimal, amount decimal.Decimal, rate float64, n int) error {
	t.OrderOver = false
	o, err := t.sell(price, amount, rate, n)
	if err != nil {
		log.Printf("卖出错误: %d, err: %s", t.base, err)
		return err
	} else {
		// if o.ClientId != "" {
		// 	t.ParseOrder(o)
		// 	t.last = price
		// 	return nil
		// }
		if t.WaitOrder(o.OrderId, o.ClientId) {
			// t.ParseOrder(o)
			t.last = price
			return nil
		} else {
			// t.goex.CancelOrder(o.OrderId)
			return errors.New("卖出出错")
		}
	}
}

// WaitBuy 买入等待
func (t *ExTrader) WaitBuy(price decimal.Decimal, amount decimal.Decimal, rate float64) error {
	t.OrderOver = false
	o, err := t.buy(price, amount, rate)
	if err != nil {
		log.Printf("买入错误: %d, err: %s", t.base, err)
		return err
	} else {
		// if o.ClientId != "" && t.goex.symbol.Category != "OKex" {
		// 	t.ParseOrder(o)
		// 	t.last = decimal.NewFromFloat(o.Price)
		// 	return nil
		// }
		if t.WaitOrder(o.OrderId, o.ClientId) {
			return nil
		} else {
			t.sell(price, t.CountHold(), 0, 0) // 无法获取买入状态就结束任务，不再买入,防止以上买入订单成交,卖出一次
			return errors.New("买入出错")
		}
	}
}

// 获取账户余额
func (t *ExTrader) myMoney() (m decimal.Decimal) {
	_, m, _ = t.goex.GetAccount()
	return
}

// 获取coin
func (t *ExTrader) myCoin() (coin decimal.Decimal) {
	_, _, coin = t.goex.GetAccount()
	return
}

/**
 *@title        : SellCount
 *@desc         : 计算卖出数量并设置精度为交易所需要精度, 如果要卖出数量大于用户持有货币数量就更改为用户持有货币数量
 *@auth         : small_ant / time(2021/08/07 13:55:15)
 *@param        : sell / decimal.Decimal / `要卖出数量`
 *@return       : coin / decimal.Decimal / `能够卖出的币种数量`
 */
func (t *ExTrader) SellCount(sell decimal.Decimal) (coin decimal.Decimal) {
	c := t.myCoin()
	coin = sell
	if coin.Cmp(c) == 1 && t.u.Future == 0 {
		coin = c
	}
	coin = t.ToPrecision(coin)

	log.Printf("用户%v卖出计算:%v", t.u.ObjectId, coin)
	return
}

// CountMesure 币本位计算卖出
func (t *ExTrader) CountMesure() (amount decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			amount = amount.Add(g.Mesure)
		}
	}
	log.Printf("用户%v卖出计算:%v", t.u.ObjectId, amount)
	return
}
