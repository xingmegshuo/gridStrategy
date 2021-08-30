package grid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"time"
	"zmyjobs/corn/exchange/huobi"
	model "zmyjobs/corn/models"

	"github.com/shopspring/decimal"
)

// Run 跑模型交易
func Run(ctx context.Context, u model.User) {
	//var ctx *cli.Context
	status := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("停止交易", u.ObjectId)
			return
		default:
		}
		for i := 0; i < 1; i++ {
			if status == 0 {
				status = 1
				log.Println("尝试获取用户账户数据，校验余额，api 等信息正确性---", u.ObjectId)
				g, e := NewGrid(u)
				if e != nil || len(g.grids) != int(u.Number) {
					u.IsRun = -10
					u.Error = "api 请求超时，或api接口更改"
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
func NewGrid(u model.User) (*Trader, error) {
	symbol := model.StringSymobol(u.Symbol)
	arg := model.StringArg(u.Arg)
	grid, _ := model.SourceStrategy(u, false)
	var realGrid []model.Grid

	_ = json.Unmarshal([]byte(u.RealGrids), &realGrid)

	// **准备丢弃
	var cli *Cli
	var err error
	switch symbol.Category {
	case "火币":
		c, e := huobi.New(huobi.Config{Host: symbol.Host, Label: symbol.Label, SecretKey: symbol.Secret, AccessKey: symbol.Key})
		err = e
		cli = &Cli{c}
	}
	if err != nil {
		return nil, err
	}

	return &Trader{
		grids:     *grid,
		ex:        cli,     //**
		symbol:    &symbol, //**
		arg:       &arg,
		RealGrids: realGrid,
		goex:      NewEx(&symbol),
	}, nil
}

// Trade 创建websocket 并执行策略中的交易任务
func (t *Trader) Trade(ctx context.Context) {
	//_ = t.Print()
	// clientId := fmt.Sprintf("%d", time.Now().Unix())
	c := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("结束交易和websocket-----", t.u.ObjectId)
			return
		default:
		}
		for i := 0; i < 1; i++ {
			if c == 0 {
				c = 1
				// **不必开启websocket
				log.Println("开启websocket--------", t.u.ObjectId)
				// go t.ex.huobi.SubscribeOrder(ctx, t.symbol.Symbol, clientId, t.OrderUpdateHandler)
				// go t.ex.huobi.SubscribeTradeClear(ctx, t.symbol.Symbol, clientId, t.TradeClearHandler)

				if err := t.ReBalance(ctx); err != nil {
					log.Println("校验账户余额不足够，策略不开始----", t.u.ObjectId)
					t.u.IsRun = -10
					t.u.Error = err.Error()
					t.u.Update()
					// 执行报错就关闭
					GridDone <- 1
				} else {
					for i := 0; i < len(t.grids); i++ {
						if i < t.base {
							t.pay = t.pay.Add(t.RealGrids[i].TotalBuy)
							t.SellMoney = t.SellMoney.Add(t.RealGrids[i].AmountSell)
						}
					}
					t.setupGridOrders(ctx)
					// t.SearchOrder("330993239964613")
					// t.testGridOrder(ctx)
					if t.ErrString != "" {
						log.Println("网络链接问题：", t.u.ObjectId)
						t.u.IsRun = -10
						t.u.Error = t.ErrString
						t.u.Update()

						model.StrategyError(t.u.ObjectId, t.ErrString)
						// 执行报错就关闭
						GridDone <- 1
					}
					if t.over {
						// 策略执行完毕 to do 计算盈利
						log.Println("策略一次执行完毕:", t.u.ObjectId, "盈利:", t.CalCulateProfit())
						p, _ := t.CalCulateProfit().Float64()
						// 盈利ctx
						t.u.IsRun = 1
						t.u.BasePrice = p
						// model.RunOver(t.u.Custom, t.u.BasePrice, float64(t.u.ObjectId))
						// model.LogStrategy(t.arg.CoinId, t.symbol.Category, t.symbol.QuoteCurrency, t.u.ObjectId,
						// 	t.u.Custom, t.amount, t.cost, t.arg.IsHand, p,)
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

// cancel撤单操作
func (t *Trader) cancel(order uint64, order_id string) {
	for {
		if _, err := t.ex.huobi.CancelOrder(order); err == nil {
			log.Println("撤单成功", t.SellOrder)
			model.DeleteRebotLog(order_id)
			break
		} else {
			time.Sleep(time.Second * 5)
			continue
		}
	}
}

// setupGridOrders 测试
func (t *Trader) setupGridOrders(ctx context.Context) {
	count := 0
	t.GetLastPrice()
	log.Println("上次交易:", t.last, "基础价格:", t.basePrice, "投入金额:", t.pay, "当前持仓:", t.amount, "---------策略开始", "用户:", t.u.ObjectId)
	var (
		low  = t.last
		high = t.last
	)
	t.amount = t.CountHold()
	error := 0
	t.pay = t.CountPay()
	for {
		count++
		time.Sleep(time.Millisecond * 500)                 // 间隔0.5秒查询
		t.GetMoeny()                                       // 获取当前money和持仓
		price, err := t.ex.huobi.GetPrice(t.symbol.Symbol) // 获取当前价格
		if err != nil {
			t.ErrString = err.Error()
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
		if count%600 == 0 {
			log.Println("当前盈利:", win*100, "单数:", t.base, "下跌:", die*100, "上次交易:", t.last, "当前价格：", "\n",
				price, "持仓:", t.amount, "最高价:", high, "最低价:", low, "回降比例:", reduce*100, "回调比例:", top*100)
		}
		select {
		case <-ctx.Done():
			log.Println("close get price ", t.u.ObjectId)
			runtime.Goexit()
		case op := <-model.OperateCh:
			if op.Id == float64(t.u.ObjectId) {
				if op.Op == 1 {
					t.arg.AllSell = true
					log.Println("卖出")
				}
				if op.Op == 2 {
					t.arg.OneBuy = true
					log.Println("买入")
				}
				if op.Op == 3 {
					t.arg.StopBuy = true
					log.Println("停止买入")
				}
				if op.Op == 4 {
					t.arg.StopBuy = false
					log.Println("恢复买入")
				}
			}
		default:
		}

		//  第一单 进场时机无所谓
		if t.base == 0 && !t.arg.StopBuy {
			willbuy := false
			t.OrderOver = false
			if t.arg.IsLimit && price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh)) < 1 {
				willbuy = true
			} else if !t.arg.IsLimit {
				willbuy = true
			}
			if count >= 10 && willbuy {
				log.Println("开始进场首次买入:---价格", price, "---数量:", t.grids[t.base].AmountBuy, "---money", t.grids[t.base].TotalBuy)
				err := t.WaitBuy(price)
				if err != nil {
					log.Printf("买入错误: %d, err: %s", t.base, err)
					time.Sleep(time.Second * 5)
					if error < 5 {
						error += 1
						continue
					} else {
						t.ErrString = "意料之外的错误,联系开发者处理"
						t.over = true
					}
				} else {
					high = price
					low = price
					t.last = price
				}
			}
		}
		// 后续买入按照跌幅+回调来下单
		if 0 < t.base && t.base < len(t.grids) && !t.arg.StopBuy {
			if die*100 >= t.grids[t.base].Decline && top*100 >= t.arg.Reduce {
				t.OrderOver = false
				log.Println(t.base, "买入:", price, t.grids[t.base].AmountBuy, "下降幅度:", die, "价格:", t.grids[t.base].Price, "----------", price.Cmp(t.grids[t.base].Price))
				err := t.WaitBuy(price)
				if err != nil {
					log.Printf("买入错误: %d, err: %s", t.base, err)
					time.Sleep(time.Second * 5)
					continue
				} else {
					high = price
					low = price
					t.last = price
				}
			}
		}

		// 智乘方
		if t.arg.StrategyType == 1 || t.arg.StrategyType == 3 {
			if t.setupBi(win, reduce, price) != nil {
				continue
			} else {
				t.Tupdate()
			}
		}
		// 智多元
		if t.arg.StrategyType == 2 || t.arg.StrategyType == 4 {
			if t.SetupBeMutiple(price, reduce, win) != nil {
				continue
			} else {

				t.Tupdate()
			}
		}

		//  如果不相等更新
		if t.base != t.u.Base {
			t.Tupdate()
		}

		if t.over {
			log.Println("任务结束", t.u.ObjectId)
			break
		}

		// 清仓
		if t.arg.AllSell {
			log.Println("卖出咯")
			t.arg.AllSell = false
			t.AllSellMy()
			err := t.WaitSell(price, win*100)
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			} else {
				t.over = true
				t.Tupdate()
				break
			}
		}
		// 立即买入
		if t.arg.OneBuy && t.base < len(t.grids)-1 {
			log.Println("一键补仓")
			t.arg.OneBuy = false
			model.OneBuy(t.u.ObjectId)
			err := t.WaitBuy(price)
			if err != nil {
				log.Printf("买入错误: %d, err: %s", t.base, err)
				time.Sleep(time.Second * 5)
				continue
			} else {
				high = price
				low = price
				t.last = price
			}
		}
	}
}

func (t *Trader) Buy(price decimal.Decimal) (uint64, string, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	orderId, err := t.buy(clientOrderId, price, t.grids[t.base].TotalBuy.Div(price).Round(t.symbol.AmountPrecision), t.GetRate())
	return orderId, clientOrderId, err
}

func (t *Trader) Sell(price decimal.Decimal, rate float64, n int) (uint64, string, error) {
	clientOrderId := fmt.Sprintf("s-%d-%d", t.base, time.Now().Unix())
	t.amount = t.CountHold()
	log.Println("卖出数量:", t.amount, t.u.ObjectId)
	orderId, err := t.sell(clientOrderId, price.Round(t.symbol.PricePrecision), t.amount.Truncate(t.symbol.AmountPrecision).Round(t.symbol.AmountPrecision), rate, n)
	// t.amount.RoundBank(-t.symbol.AmountPrecision))

	g := map[string]int{clientOrderId: 1}
	t.SellOrder = g

	return orderId, clientOrderId, err
}

func (t *Trader) SellAmount(price decimal.Decimal, rate float64, amount decimal.Decimal, n int) (uint64, string, error) {
	clientOrderId := fmt.Sprintf("s-%d-%d", len(t.SellOrder)+1, time.Now().Unix())
	orderId, err := t.sell(clientOrderId, price.Round(t.symbol.PricePrecision), amount.Truncate(t.symbol.AmountPrecision).Round(t.symbol.AmountPrecision), rate, n)
	// t.amount.RoundBank(-t.symbol.AmountPrecision))
	log.Println("卖出数量:", amount, t.u.ObjectId)
	g := map[string]int{clientOrderId: n}
	t.SellOrder = g
	return orderId, clientOrderId, err
}

// GetRate 获取跌幅
func (t *Trader) GetRate() float64 {
	if t.base > 0 && t.base < len(t.grids) {
		return t.arg.AddRate*float64(t.base-1) + t.arg.Rate
	} else {
		return 0
	}
}

func (t *Trader) WaitOrder(orderId string, cli string) bool {
	log.Println("等待订单交易.......")
	start := time.Now()
	for {
		if t.OrderOver {
			return true
		}
		if time.Since(start) >= time.Second*30 {
			if t.SearchOrder(orderId, cli) {
				return true
			} else {
				return false
			}
		}
	}
}

// WaitBuy 买入等待
func (t *Trader) WaitBuy(price decimal.Decimal) error {
	t.OrderOver = false
	orderId, clientOrder, err := t.Buy(price)
	if err != nil {
		log.Printf("买入错误: %d, err: %s", t.base, err)
		return err
	} else {
		if t.WaitOrder(strconv.FormatUint(orderId, 10), clientOrder) {
			t.base++
			t.last = price
			return nil
		} else {
			t.cancel(orderId, clientOrder)
			return errors.New("买入出错")
		}
	}
}

// WaitSell 卖出等待
func (t *Trader) WaitSell(price decimal.Decimal, rate float64) error {
	t.OrderOver = false
	orderId, clientOrder, err := t.Sell(price, rate, t.base)
	if err != nil {
		log.Printf("卖出错误: %d, err: %s", t.base, err)
		return err
	} else {
		if t.WaitOrder(strconv.FormatUint(orderId, 10), clientOrder) {
			t.last = price
			return nil
		} else {
			t.cancel(orderId, clientOrder)
			return errors.New("卖出出错")
		}
	}
}

// WaitSellLimit 指定卖出数量
func (t *Trader) WaitSellLimit(price decimal.Decimal, rate float64, amount decimal.Decimal, n int) error {
	t.OrderOver = false
	orderId, clientOrder, err := t.SellAmount(price, rate, amount, n)
	if err != nil {
		log.Printf("卖出错误: %d, err: %s", t.base, err)
		return err
	} else {
		if t.WaitOrder(strconv.FormatUint(orderId, 10), clientOrder) {
			t.last = price
			return nil
		} else {
			t.cancel(orderId, clientOrder)
			return errors.New("卖出出错")
		}
	}
}

// GetLastPrice 获取上次交易价格
func (t *Trader) GetLastPrice() {
	if len(t.u.RealGrids) > 0 && t.base > 1 {
		t.last = t.RealGrids[t.base-1].Price
	} else {
		t.last = t.grids[0].Price
	}
}

// ChangeHighLow 修改价格标定
func ChangeHighLow(price decimal.Decimal, high decimal.Decimal, low decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if low.Cmp(price) == 1 { // 最低价
		low = price
	}
	if high.Cmp(price) == -1 { // 最高价
		high = price
	}
	return low, high
}

// Tupdate 更新数据
func (t *Trader) Tupdate() {
	t.u.Base = t.base
	t.u.Total = t.amount.String()
	s, _ := json.Marshal(t.grids)
	t.u.Grids = string(s)
	t.u.RealGrids = model.ToStringJson(t.RealGrids)
	t.u.Update()
}

// AllSellMy 平仓
func (t *Trader) AllSellMy() {
	log.Println("一键平仓：", t.u.ObjectId)
	t.arg.AllSell = false
	t.u.Arg = model.ToStringJson(&t.arg)
	t.u.Update()
	model.OneSell(t.u.ObjectId)
}

// SearchOrder 查询订单
func (t *Trader) SearchOrder(clientOrderId string, client string) bool {
	log.Println("查询订单", client, "依据此查找:", clientOrderId)
	data, b, err := t.ex.huobi.SearchOrder(clientOrderId)
	if err == nil {
		if b {
			if data != nil {
				amount, _ := decimal.NewFromString(data["amount"])
				price, _ := decimal.NewFromString(data["price"])
				transact, _ := decimal.NewFromString(data["fee"])
				t.GetMoeny()
				if b, ok := t.SellOrder[client]; ok {
					sellMoney := price.Mul(amount).Abs().Sub(transact)
					t.SellMoney = t.SellMoney.Add(sellMoney)  // 卖出钱
					t.RealGrids[b-1].AmountSell = t.SellMoney // 修改卖出
					// model.RebotUpdateBy(client, price, amount.Abs(), transact, t.RealGrids[b-1].TotalBuy, t.hold, "成功")
					t.amount = t.CountHold()
					t.pay = t.CountPay()
					t.cost = t.pay.Div(t.amount)
					if b == t.base {
						t.over = true
						model.AsyncData(t.u.ObjectId, 0.00, 0.00, 0.00, 0)
					} else {
						model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.CountNum())
					}
				} else {
					t.RealGrids[t.base].AmountBuy = amount.Sub(transact)
					t.RealGrids[t.base].Price = price
					t.RealGrids[t.base].TotalBuy = price.Mul(amount.Sub(transact))
					t.amount = t.CountHold()
					t.pay = t.CountPay()
					t.cost = t.pay.Div(t.amount)
					// model.RebotUpdateBy(client, price, amount.Sub(transact), transact, t.RealGrids[t.base].TotalBuy, t.hold, "成功")
					model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
				}
				return true
			}
		}
	}
	return false
}

// 获取当前持仓数量
func (t *Trader) CountHold() (amount decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			amount = amount.Add(g.AmountBuy)
		}
	}
	return
}

// 获取当前投入金额
func (t *Trader) CountPay() (pay decimal.Decimal) {
	for _, g := range t.RealGrids {
		if g.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			pay = pay.Add(g.TotalBuy)
		}
	}
	return
}

// CalCulateProfit 计算盈利
func (t *Trader) CalCulateProfit() decimal.Decimal {
	var pay, my decimal.Decimal
	for _, b := range t.RealGrids {
		pay = pay.Add(b.TotalBuy)
		my = my.Add(b.AmountSell)
	}
	// log.Println("pay:", pay, "my:", my)
	return my.Sub(pay)
}

// CountBuy 计算买入数量
func (t *Trader) CountBuy() decimal.Decimal {
	var amount decimal.Decimal
	for _, b := range t.RealGrids {
		amount = amount.Add(b.AmountBuy)
	}
	return amount
}

func (t *Trader) TradeGrid() {
	if len(t.RealGrids) < t.base+1 {
		t.RealGrids = append(t.RealGrids, model.Grid{Id: t.base + 1})
	}
}

// 持仓单数
func (t *Trader) CountNum() int {
	var n = 0
	for _, b := range t.RealGrids {
		if b.AmountSell.Cmp(decimal.Decimal{}) < 1 {
			n++
		}
	}
	return n
}
