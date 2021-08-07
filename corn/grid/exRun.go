package grid

import (
    "context"
    "encoding/json"
    "runtime"
    "time"
    model "zmyjobs/corn/models"

    "github.com/shopspring/decimal"
)

func RunEx(ctx context.Context, u model.User) {
    //var ctx *cli.Context
    status := 0
    for {
        select {
        case <-ctx.Done():
            log.Printf("%v停止交易", u.ObjectId)
            return
        default:
        }
        for i := 0; i < 1; i++ {
            if status == 0 {
                status = 1
                g := NewExStrategy(u)
                if len(g.grids) != int(u.Number) {
                    u.IsRun = -10
                    u.Error = "api 请求超时，或api接口更改"
                    log.Println(u.Error)
                    u.Update()
                    GridDone <- 1
                } else {
                    g.u = u
                    go g.Trade(ctx)
                }
            }
        }
    }
}

// NewGrid 实例化对象，并验证api key的正确性
func NewExStrategy(u model.User) (ex *ExTrader) {
    arg := model.StringArg(u.Arg)
    grid, _ := model.SourceStrategy(u, false)
    var realGrid []Grid
    _ = json.Unmarshal([]byte(u.RealGrids), &realGrid)
    symbol := model.StringSymobol(u.Symbol)
    ex = &ExTrader{
        grids:     *grid,
        arg:       &arg,
        RealGrids: realGrid,
        goex:      NewEx(&symbol),
    }
    return
}

// Trade 创建websocket 并执行策略中的交易任务
func (t *ExTrader) Trade(ctx context.Context) {
    //_ = t.Print()
    c := 0
    for {
        select {
        case <-ctx.Done():
            log.Printf("%v结束交易", t.u.ObjectId)
            return
        default:
        }
        for i := 0; i < 1; i++ {
            if c == 0 {
                c = 1
                log.Printf("尝试获取%v用户账户数据，校验余额，api 等信息正确性", t.u.ObjectId)
                if err := t.ReBalance(ctx); err != nil {
                    log.Printf("校验%v账户余额不足够，策略不开始----", t.u.ObjectId)
                    t.u.IsRun = -10
                    t.u.Error = err.Error()
                    log.Println(err, t.u.ObjectId)
                    t.u.Update()
                    // 执行报错就关闭
                    GridDone <- 1
                } else {
                    t.setupGridOrders(ctx)
                    if t.ErrString != "" {
                        log.Println("网络链接问题：", t.u.ObjectId)
                        t.u.IsRun = -10
                        t.u.Error = t.ErrString
                        t.u.Update()
                        model.StrategyError(t.u.ObjectId, t.ErrString)
                        // 执行报错就关闭
                        GridDone <- 1
                    } else if t.over {
                        // 策略执行完毕 to do 计算盈利
                        log.Println("策略一次执行完毕:", t.u.ObjectId, "盈利:", t.CalCulateProfit())
                        p, _ := t.CalCulateProfit().Float64()
                        // 盈利ctx
                        t.u.IsRun = 1
                        t.u.BasePrice = p
                        model.RunOver(p, t.u.Custom)
                        model.LogStrategy(t.goex.symbol.Category, t.goex.symbol.QuoteCurrency, t.u.ObjectId,
                            t.u.Custom, t.amount, t.cost, t.arg.IsHand, t.CalCulateProfit().Abs())
                        t.u.RealGrids = "***"
                        t.u.Base = 0
                        t.u.Update()
                        model.DB.Exec("update users set base = 0 where id = ?", t.u.ID)
                        GridDone <- 1
                    }
                }
            }
        }
    }
}

// setupGridOrders 测试
func (t *ExTrader) setupGridOrders(ctx context.Context) {
    count := 0
    t.GetLastPrice()
    log.Println("上次交易:", t.last, "基础价格:", t.basePrice, "投入金额:", t.pay, "当前持仓:", t.amount, "策略开始", "用户:", t.u.ObjectId)
    var (
        low  = t.last
        high = t.last
    )
    for {
        count++
        time.Sleep(time.Millisecond * 500) // 间隔0.5秒查询
        price, err := t.goex.GetPrice()    // 获取当前价格
        if err != nil {
            t.ErrString = err.Error()
            log.Println(err, t.u.ObjectId)
            return
        }
        low, high = ChangeHighLow(price, high, low)
        // 计算盈利
        win := float64(0)
        if t.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
            win, _ = (price.Mul(t.amount).Sub(t.pay)).Div(t.pay).Float64() // 计算盈利 当前价值-投入价值
        }
        reduce, _ := high.Sub(price).Div(t.last).Float64() // 当前回降
        top, _ := price.Sub(low).Div(t.last).Float64()     // 当前回调
        die, _ := t.last.Sub(price).Div(t.last).Float64()  // 当前跌幅
        // 输出日志
        // if count%50 == 0 {
        //     log.Println("当前盈利:", win*100, "单数:", t.base, "下跌:", die*100, "上次交易:", t.last, "当前价格：",
        //         price, "持仓:", t.amount, "最高价:", high, "最低价:", low, "回降比例:", reduce*100, "回调比例:", top*100,
        //         "用户:", t.u.ObjectId)
        // }
        select {
        case <-ctx.Done():
            log.Println("close get price ", t.u.ObjectId)
            runtime.Goexit()
        case op := <-model.OperateCh:
            if op.Id == float64(t.u.ObjectId) {
                if op.Op == 1 {
                    t.arg.AllSell = true
                    log.Printf("用户%d清仓操作----", t.u.ObjectId)
                }
                if op.Op == 2 {
                    t.arg.OneBuy = true
                    log.Printf("用户%d一键补仓----", t.u.ObjectId)
                }
                if op.Op == 3 {
                    t.arg.StopBuy = true
                    log.Printf("用户%d停止买入----", t.u.ObjectId)
                }
                if op.Op == 4 {
                    if t.arg.StopBuy {
                        t.arg.StopBuy = false
                        log.Printf("用户%d恢复买入----", t.u.ObjectId)
                    }
                }
            }
        default:
        }

        //  第一单 进场时机无所谓
        if t.base == 0 && !t.arg.StopBuy {
            willbuy := false
            if t.arg.IsLimit && price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh)) < 1 {
                log.Println(price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh)), price, t.arg.LimitHigh, "限价启动", t.arg.IsLimit)
                willbuy = true

            } else if !t.arg.IsLimit {
                time.Sleep(time.Second * 2)
                willbuy = true
            }
            if count == 30 || willbuy {
                log.Printf("首次买入信息:{价格:%v,数量:%v,用户:%v,钱:%v}", price, t.grids[t.base].AmountBuy, t.u.ObjectId, t.grids[t.base].TotalBuy)
                err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), 0)
                if err != nil {
                    log.Printf("买入错误: %d, err: %s", t.base, err)
                    time.Sleep(time.Second * 5)
                    t.over = true
                } else {
                    high = price
                    low = price
                    log.Println("首次买入成功")
                    t.last = t.RealGrids[0].Price
                    t.base = t.base + 1
                    t.Tupdate()
                }
            }
        }
        // 后续买入按照跌幅+回调来下单
        if 0 < t.base && t.base < len(t.grids) && !t.arg.StopBuy {
            if die*100 >= t.grids[t.base].Decline && top*100 >= t.arg.Reduce {
                log.Printf("第%d买入信息:{价格:%v,数量:%v,用户:%v,钱:%v,跌幅:%v}", t.base+1, price, t.grids[t.base].AmountBuy, t.u.ObjectId, t.grids[t.base].TotalBuy, die)
                err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), die*100)
                if err != nil {
                    log.Printf("买入错误: %d, err: %s", t.base, err)
                    time.Sleep(time.Second * 5)
                    t.ErrString = err.Error()
                    t.over = true
                } else {
                    high = price
                    low = price
                    t.last = t.RealGrids[t.base].Price
                    t.base = t.base + 1
                    t.Tupdate()
                }
            }
        }

        // 智乘方
        if t.arg.StrategyType == 1 || t.arg.StrategyType == 3 {
            if err := t.setupBi(win, reduce, price); err != nil {
                t.ErrString = err.Error()
                t.over = true
            } else {
                t.Tupdate()
            }
            if t.arg.AllSell {
                log.Printf("%v用户智乘方清仓", t.u.ObjectId)
                t.arg.AllSell = false
                t.AllSellMy()
                err := t.WaitSell(price, t.SellCount(t.CountHold()), win*100, len(t.RealGrids)-1)
                if err != nil {
                    time.Sleep(time.Second * 5)
                    t.ErrString = err.Error()
                    t.over = true
                } else {
                    t.over = true
                    t.Tupdate()
                }
            }
        }
        // 智多元
        if t.arg.StrategyType == 2 || t.arg.StrategyType == 4 {
            if t.SetupBeMutiple(price, reduce, win) != nil {
                t.ErrString = "卖出错误"
                t.over = true
            } else {
                t.Tupdate()
            }
            if t.arg.AllSell {
                log.Printf("%v用户智多元清仓", t.u.ObjectId)
                t.arg.AllSell = false
                t.AllSellMy()
                for {
                    for _, g := range t.RealGrids {
                        err := t.WaitSell(price, t.SellCount(g.AmountBuy), win*100, g.Id)
                        if err != nil {
                            time.Sleep(time.Second * 5)
                            t.ErrString = err.Error()
                            t.over = true
                            break
                        } else {
                            t.Tupdate()
                        }
                    }
                    time.Sleep(time.Second)
                    if t.CountHold().Cmp(decimal.Decimal{}) < 1 {
                        break
                    } else {
                        continue
                    }
                }
            }
        }

        // 立即买入
        if t.arg.OneBuy && t.base < len(t.grids)-1 {
            log.Printf("%v用户一键补仓", t.u.ObjectId)
            t.arg.OneBuy = false
            model.OneBuy(t.u.ObjectId)
            err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), die*100)
            if err != nil {
                log.Printf("买入错误: %d, err: %s", t.base, err)
                time.Sleep(time.Second * 5)
                t.ErrString = err.Error()
                t.over = true
            } else {
                high = price
                low = price
                t.last = t.RealGrids[t.base].Price
                t.base = t.base + 1
                t.Tupdate()
            }
        }
        //  如果不相等更新
        // if t.base != t.u.Base {
        //     t.Tupdate()
        // }
        if t.over {
            log.Printf("%v用户任务结束", t.u.ObjectId)
            break
        }
    }
}

// GetLastPrice 获取上次交易价格
func (t *ExTrader) GetLastPrice() {
    if len(t.u.RealGrids) > 0 && t.base > 1 {
        t.last = t.RealGrids[t.base-1].Price
    } else {
        t.last = t.grids[0].Price
    }
}

// Tupdate 更新数据
func (t *ExTrader) Tupdate() {
    t.u.Base = t.base
    t.u.Total = t.amount.String()
    s, _ := json.Marshal(t.grids)
    t.u.Grids = string(s)
    t.u.RealGrids = model.ToStringJson(t.RealGrids)
    t.u.Update()
}

// AllSellMy 平仓
func (t *ExTrader) AllSellMy() {
    log.Println("一键平仓：", t.u.ObjectId)
    t.arg.AllSell = false
    t.u.Arg = model.ToStringJson(&t.arg)
    // t.u.Update()
    model.OneSell(t.u.ObjectId)
}

func (t *ExTrader) ToPrecision(p decimal.Decimal) decimal.Decimal {
    return p.Truncate(t.goex.symbol.AmountPrecision).Round(t.goex.symbol.AmountPrecision)
}
