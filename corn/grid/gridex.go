package grid

import (
	"context"
	"errors"
	"time"
	model "zmyjobs/corn/models"

	"github.com/shopspring/decimal"
)

type ExTrader struct {
	arg       *model.Args     // 策略参数
	grids     []model.Grid    // 预设策略
	RealGrids []Grid          // 成交信息
	base      int             // 当前单数
	ErrString string          // 错误信息
	over      bool            // 是否结束
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
}

// 策略预览内容
type Grid struct {
	Id         int
	Price      decimal.Decimal // 价格
	AmountBuy  decimal.Decimal // 买入数量
	Decline    float64         // 跌幅,或者说盈利
	AmountSell decimal.Decimal // 卖出数量
	TotalBuy   decimal.Decimal // money
	Order      string          // 订单id
	Mesure     decimal.Decimal // b本位计量单位张
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
	log.Printf("数量:%v;用户:%v;需要:%v", t.amount, t.u.ObjectId, moneyNeed)
	for i := 0; i < len(t.RealGrids); i++ {
		if i < t.base {
			t.pay = t.pay.Add(t.RealGrids[i].TotalBuy)
			t.SellMoney = t.SellMoney.Add(t.RealGrids[i].AmountSell)
		}
	}
	moneyNeed = decimal.Decimal{}
	b, moneyHeld, coinHeld := t.goex.GetAccount()
	if b {
		log.Printf("账户余额: %s, 币种:  %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
		if moneyNeed.Cmp(moneyHeld) == 1 {
			return errors.New("no enough money")
		}

	} else {
		b, moneyHeld, coinHeld = t.goex.GetAccount()
		if b {
			log.Printf("账户余额: %s, 币种:  %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
			if moneyNeed.Cmp(moneyHeld) == 1 {
				return errors.New("no enough money")
			}
		} else {
			return errors.New("account or api error")
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
func (t *ExTrader) buy(price, amount decimal.Decimal, rate float64) (string, string, error) {
	orderType := BuyL
	msg := "限价买入"
	if t.arg.OrderType == 2 {
		orderType = BuyM
		amount = t.grids[t.base].TotalBuy
		msg = "市价买入"
	}
	log.Println(t.arg.Crile)
	if t.goex.symbol.Future && t.arg.Crile == 3 {
		orderType = OpenDL
		amount = t.grids[t.base].TotalBuy
		msg = "开多"
	} else if t.goex.symbol.Future && t.arg.Crile == 4 {
		orderType = OpenLL
		amount = t.grids[t.base].TotalBuy
		msg = "开空"
	}
	clientId, orderId, err := t.goex.Exchanges(amount, price, orderType, false)
	log.Printf("[Order][buy] 价格: %s, 数量: %s, 用户:%d,订单号:%v,自定义订单号:%v", price, amount, t.u.ObjectId, orderId, clientId)
	// 增加一个真实成交
	if err == nil {
		// fmt.Println("已经挂单的买入", t.u.ObjectId)
		t.log(orderId, price, msg, t.base, amount, rate, "买入")
		g := Grid{
			Id:      t.base + 1,
			Decline: rate,
			Order:   orderId,
		}
		if t.u.Future > 0 {
			g.Mesure = t.grids[t.base].AmountBuy
		}
		t.RealGrids = append(t.RealGrids, g)
	}
	return orderId, clientId, err
}

// 卖出
func (t *ExTrader) sell(price, amount decimal.Decimal, rate float64, n int) (string, string, error) {
	orderType := SellL
	msg := "限价卖出"
	if t.arg.OrderType == 2 {
		orderType = SellM
		price = decimal.Decimal{}
		msg = "市价卖出"
	}
	if t.goex.symbol.Future && t.arg.Crile == 3 {
		msg = "平多"
		orderType = OpenDM
	} else if t.goex.symbol.Future && t.arg.Crile == 4 {
		orderType = OpenLM
		msg = "平空"
	}
	if t.u.Future == 2 || t.u.Future == 4 {
		amount = t.CountMesure()
	}
	log.Printf("类型:%v,数量:%v,价格:%v;用户:%v", msg, amount, price, t.u.ObjectId)
	clientId, orderId, err := t.goex.Exchanges(amount, price, orderType, false)
	log.Printf("[Order][sell] 价格: %s, 数量: %s, 用户:%d,订单号:%v,自定义订单号:%v", price, amount, t.u.ObjectId, orderId, clientId)
	// 增加一个真实成交
	if err == nil {
		t.RealGrids[n].Decline = rate
		t.log(orderId, price, msg, n, amount, rate, "卖出")
		g := map[string]int{orderId: n}
		t.SellOrder = g
	}
	return orderId, clientId, err
}

// SearchOrder 查询订单
func (t *ExTrader) SearchOrder(clientOrderId string, client string) bool {
	// log.Println("查询订单", client, "依据此查找:", clientOrderId)
	has, over, order := t.goex.SearchOrder(clientOrderId)
	if has && over {
		price := decimal.NewFromFloat(order.Price)
		amount := decimal.NewFromFloat(order.Amount)
		fee := decimal.NewFromFloat(order.Fee)
		if t.u.Future == 2 || t.u.Future == 4 {
			amount = decimal.NewFromFloat(order.Cash)
		}
		t.hold = t.myMoney()
		log.Printf("订单成功--- 价格:%v  数量: %v  手续费: %v 成交额: %v 订单号: %v", order.Price, order.Amount, order.Fee, order.Cash, order.OrderId)
		if b, ok := t.SellOrder[clientOrderId]; ok {
			log.Printf("当前单数:%v,卖出单数:%v", t.base, b)
			sellMoney := price.Mul(amount).Abs().Sub(fee)
			t.SellMoney = t.SellMoney.Add(sellMoney) // 卖出钱
			t.RealGrids[b].AmountSell = t.SellMoney  // 修改卖出
			t.amount = t.CountHold()
			t.pay = t.CountPay()
			model.RebotUpdateBy(clientOrderId, price, amount.Abs(), fee, t.RealGrids[b].TotalBuy, t.hold, "成功", order.ClientId, t.u.Future)
			if b == 0 {
				t.over = true
				model.AsyncData(t.u.ObjectId, 0.00, 0.00, 0.00, 0)
			} else {
				model.AsyncData(t.u.ObjectId, t.amount, t.pay.Div(t.amount), t.pay, b)
			}
		} else {
			t.RealGrids[t.base].AmountBuy = amount.Sub(fee)
			t.RealGrids[t.base].Price = price
			t.RealGrids[t.base].TotalBuy = price.Mul(amount.Sub(fee))
			if t.u.Future == 1 || t.u.Future == 3 {
				t.RealGrids[t.base].TotalBuy = decimal.NewFromFloat(order.Cash)
			}
			t.amount = t.CountHold()
			t.pay = t.CountPay()
			t.cost = t.pay.Div(t.amount)
			model.RebotUpdateBy(clientOrderId, price, amount.Sub(fee), fee, t.RealGrids[t.base].TotalBuy, t.hold, "成功", order.ClientId, t.u.Future)
			model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
		}
		t.Tupdate()
		return true
	}
	return false
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

// 获取当前投入金额
func (t *ExTrader) CountPay() (pay decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			pay = pay.Add(g.TotalBuy)
		}
	}
	return
}

// CalCulateProfit 计算盈利
func (t *ExTrader) CalCulateProfit() decimal.Decimal {
	var pay, my decimal.Decimal
	for _, b := range t.RealGrids {
		pay = pay.Add(b.TotalBuy)
		my = my.Add(b.AmountSell)
	}
	return my.Sub(pay)
}

func (t *ExTrader) WaitOrder(orderId string, cli string) bool {
	log.Println("等待订单交易.......")
	start := time.Now()
	for {
		time.Sleep(time.Second * 10)
		if t.SearchOrder(orderId, cli) {
			return true
		}
		if time.Since(start) >= time.Second*100 {
			return false
		}
	}
}

// WaitSell 卖出等待
func (t *ExTrader) WaitSell(price decimal.Decimal, amount decimal.Decimal, rate float64, n int) error {
	t.OrderOver = false
	orderId, clientOrder, err := t.sell(price, amount, rate, n)
	if err != nil {
		log.Printf("卖出错误: %d, err: %s", t.base, err)
		return err
	} else {
		if t.WaitOrder(orderId, clientOrder) {
			t.last = price
			return nil
		} else {
			t.goex.CancelOrder(orderId)
			return errors.New("卖出出错")
		}
	}
}

// WaitBuy 买入等待
func (t *ExTrader) WaitBuy(price decimal.Decimal, amount decimal.Decimal, rate float64) error {
	t.OrderOver = false
	orderId, clientOrder, err := t.buy(price, amount, rate)
	if err != nil {
		log.Printf("买入错误: %d, err: %s", t.base, err)
		return err
	} else {
		if t.WaitOrder(orderId, clientOrder) {
			return nil
		} else {
			t.goex.CancelOrder(orderId)
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
	coin = t.CountHold()
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
	log.Println("用户%v卖出计算:%v", t.u.ObjectId, amount)
	return
}
