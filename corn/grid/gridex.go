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

// 买入
func (t *ExTrader) buy(price, amount decimal.Decimal, rate float64) (string, string, error) {
    orderType := BuyL
    if t.arg.OrderType == 2 {
        orderType = BuyM
        amount = t.grids[t.base].TotalBuy
    }
    log.Printf("[Order][buy] 价格: %s, 数量: %s, 用户:%d", price, amount, t.u.ObjectId)
    clientId, orderId, err := t.goex.Exchanges(amount, price, orderType)
    // 增加一个真实成交
    t.log(clientId, price, orderType, t.base, amount, rate, "买入")
    t.RealGrids = append(t.RealGrids, Grid{
        Id:      t.base + 1,
        Decline: rate,
        Order:   orderId,
    })
    return orderId, clientId, err
}

// 卖出
func (t *ExTrader) sell(price, amount decimal.Decimal, rate float64, n int) (string, string, error) {
    orderType := SellL
    if t.arg.OrderType == 2 {
        orderType = SellM
    }
    log.Printf("[Order][sell] 价格: %s, 数量: %s, 用户:%d", price, amount, t.u.ObjectId)
    clientId, orderId, err := t.goex.Exchanges(amount, price, orderType)
    // 增加一个真实成交
    if err == nil {
        t.RealGrids[n-1].Decline = rate
        t.log(clientId, price, orderType, t.base, amount, rate, "卖出")
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
        t.hold = t.myMoney()
        log.Printf("订单成功--- 价格:%v  数量: %v  手续费: %v 成交额: %v 订单号: %v", order.Price, order.Amount, order.Fee, order.Cash, order.OrderId)
        if b, ok := t.SellOrder[clientOrderId]; ok {
            sellMoney := price.Mul(amount).Abs().Sub(fee)
            t.SellMoney = t.SellMoney.Add(sellMoney)  // 卖出钱
            t.RealGrids[b-1].AmountSell = t.SellMoney // 修改卖出
            t.amount = t.CountHold()
            t.pay = t.CountPay()
            model.RebotUpdateBy(client, price, amount.Abs(), fee, t.RealGrids[b-1].TotalBuy, t.hold, "成功")
            if b == t.base {
                t.over = true
                model.AsyncData(t.u.ObjectId, 0.00, 0.00, 0.00, 0)
            } else {
                model.AsyncData(t.u.ObjectId, t.amount, t.pay.Div(t.amount), t.pay, b)
            }
        } else {
            t.RealGrids[t.base].AmountBuy = amount.Sub(fee)
            t.RealGrids[t.base].Price = price
            t.RealGrids[t.base].TotalBuy = price.Mul(amount.Sub(fee))
            t.amount = t.CountHold()
            t.pay = t.CountPay()
            t.cost = t.pay.Div(t.amount)
            model.RebotUpdateBy(client, price, amount.Sub(fee), fee, t.RealGrids[t.base].TotalBuy, t.hold, "成功")
            model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
        }
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
        time.Sleep(time.Second * 1)
        if t.SearchOrder(orderId, cli) {
            return true
        }
        if time.Since(start) >= time.Second*30 {
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
            t.base++
            t.last = price
            return nil
        } else {
            t.goex.CancelOrder(orderId)
            return errors.New("买入出错")
        }
    }
}

// 获取账户
func (t *ExTrader) myMoney() (m decimal.Decimal) {
    _, m, _ = t.goex.GetAccount()
    return
}
