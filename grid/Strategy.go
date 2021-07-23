package grid

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"time"
	"zmyjobs/exchange/huobi"
	model "zmyjobs/models"

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
		// time.Sleep(time.Second * 3)
		// log.Println("i am run", status)
		for i := 0; i < 1; i++ {
			if status == 0 {
				status = 1
				log.Println("尝试获取用户账户数据，校验余额，api 等信息正确性---", u.ObjectId)
				g, e := NewGrid(u)
				// g.Locks.Lock()
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
	var realGrid []model.Grid
	symbol := model.StringSymobol(u.Symbol)
	arg := model.StringArg(u.Arg)
	grid, _ := model.SourceStrategy(u, false)
	_ = json.Unmarshal([]byte(u.RealGrids), &realGrid)
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
		ex:        cli,
		symbol:    &symbol,
		arg:       &arg,
		RealGrids: realGrid,
	}, nil
}

// Trade 创建websocket 并执行策略中的交易任务
func (t *Trader) Trade(ctx context.Context) {
	//_ = t.Print()
	clientId := fmt.Sprintf("%d", time.Now().Unix())
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
				log.Println("开启websocket--------", t.u.ObjectId)
				go t.ex.huobi.SubscribeOrder(ctx, t.symbol.Symbol, clientId, t.OrderUpdateHandler)
				go t.ex.huobi.SubscribeTradeClear(ctx, t.symbol.Symbol, clientId, t.TradeClearHandler)
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
							t.pay = t.pay.Add(t.grids[i].TotalBuy)
							t.SellMoney = t.SellMoney.Add(t.grids[i].AmountSell)
						}
					}
					t.setupGridOrders(ctx)
					// t.testGridOrder(ctx)
					if t.ErrString != "" {
						log.Println("网络链接问题：", t.u.ObjectId)
						t.u.IsRun = -10
						t.u.Error = t.ErrString
						t.u.Update()
						// 执行报错就关闭
						GridDone <- 1
					}
					if t.over {
						// 策略执行完毕 to do 计算盈利
						log.Println("策略一次执行完毕:", t.u.ObjectId, "盈利:", t.SellMoney.Sub(t.pay))
						// 盈利ctx
						t.u.IsRun = 1
						t.u.Update()
						GridDone <- 1
					}
				}
			}
		}
	}
}

// cancel撤单操作
func (t *Trader) cancel(order uint64) {
	log.Println("撤单", t.SellOrder)
	for {
		if _, err := t.ex.huobi.CancelOrder(order); err == nil {
			break
		} else {
			time.Sleep(time.Second * 5)
			continue
		}
	}
}

// setupGridOrders 测试
func (t *Trader) setupGridOrders(ctx context.Context) {
	// 计数
	d := 0 // 跌
	z := 0 // 涨
	count := 0
	if len(t.u.RealGrids) > 0 {
		t.last = t.RealGrids[t.base-1].Price
	} else {
		t.last = t.grids[0].Price
	}
	// t.basePrice = t.grids[0].Price // 第一次交易价格

	log.Println("上次交易:", t.last, "基础价格:", t.basePrice, "投入金额:", t.pay, "当前持仓:", t.amount, "---------策略开始", "用户:", t.u.ObjectId)
	var (
		low  = t.last
		high = t.last
	)
	for {
		select {
		case <-ctx.Done():
			log.Println("close get price ", t.u.ObjectId)
			runtime.Goexit()
		default:
		}
		count++
		time.Sleep(time.Second * 1)
		t.GetMoeny()                                       // 获取当前money和持仓
		price, err := t.ex.huobi.GetPrice(t.symbol.Symbol) //获取当前价格
		if err != nil {
			t.ErrString = err.Error()
			return
		}
		if t.last.Cmp(price) == 1 { // 价格在下跌
			d++
		} else { // 价格在上涨
			z++
		}
		if low.Cmp(price) == 1 { // 最低价
			low = price
		}
		if high.Cmp(price) == -1 { // 最高价
			high = price
		}
		// 计算盈利
		win := float64(0)
		if t.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
			win, _ = (price.Mul(t.amount).Sub(t.pay)).Div(t.pay).Float64() // 计算盈利 当前价值-投入价值
		}
		reduce, _ := high.Sub(price).Div(t.last).Float64()
		top, _ := price.Sub(low).Div(t.last).Float64()
		die, _ := t.last.Sub(price).Div(t.last).Float64()
		if count%30 == 0 {
			log.Println("当前盈利", win*100, "单数:", t.base, "下跌:", die*100, "上次交易:", t.last, "当前价格：", price, "持仓:", t.amount,
				"最高价:", high, "\n", "最低价:", low, "回降比例:", reduce*100, "回调比例:", top*100)
		}
		//  第一单
		if t.base == 0 {
			// 在上涨
			if count == 10 {
				t.OrderOver = false
				log.Println("开始进场首次买入:", price, t.grids[t.base].AmountBuy)
				orderId, err := t.Buy(price)
				if err != nil {
					log.Printf("买入错误: %d, err: %s", t.base, err)
					continue
				} else {
					t.grids[t.base].Order = orderId
					if t.WaitOrder() {
						t.base++
						high = price
						low = price
						t.last = price
					} else {
						t.cancel(orderId)
					}
				}
			}
		}

		if 0 < t.base && t.base < len(t.grids) {
			// c, _ := price.Sub(t.last).Div(t.basePrice).Float64()
			if die*100 >= t.arg.Decline*100 && top*100 >= t.arg.Reduce {
				t.OrderOver = false
				log.Println(t.base, "买入:", price, t.grids[t.base].AmountBuy, "下降幅度:", die, "价格:", t.grids[t.base].Price, "----------", price.Cmp(t.grids[t.base].Price))
				orderId, err := t.Buy(price)
				if err != nil {
					log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
					continue
				} else {
					if t.WaitOrder() {
						t.grids[t.base].Order = orderId
						t.base++
						high = price
						low = price
						t.last = price

					} else {
						t.cancel(orderId)
					}
				}
			}
		}

		//  止盈 t.arg.Stop
		if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback {
			log.Println("盈利卖出", t.u.ObjectId)
			id, err := t.Sell(price, win)
			if err != nil {
				log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
				continue
			} else {
				if t.WaitOrder() {
					SellGrid := model.Grid{
						Id:         t.base + 1,
						Price:      price,
						Order:      id,
						AmountSell: t.amount,
						TotalBuy:   t.amount.Mul(price),
					}
					t.RealGrids = append(t.RealGrids, SellGrid)
					t.over = true
					t.base = len(t.grids)
				} else {
					t.cancel(id)
				}
			}
		}

		// if win <= -0.03 {
		// 	log.Println("亏损卖出")
		// 	id, err := t.Sell(price)
		// 	if err != nil {
		// 		log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
		// 		continue
		// 	} else {
		// 		t.SellOrder = id
		// 		break
		// 	}
		// }

		//  如果不相等更新
		if t.base != t.u.Base {
			t.u.Base = t.base
			t.u.Total = t.amount.String()
			s, _ := json.Marshal(t.grids)
			t.u.Grids = string(s)
			t.u.RealGrids = model.ToStringJson(t.RealGrids)
			t.u.Update()
		}
		if t.over {
			log.Println("任务结束", t.u.ObjectId)
			break
		}
	}
}

func (t *Trader) testGridOrder(ctx context.Context) {
	if len(t.u.RealGrids) > 0 {
		t.last = t.RealGrids[len(t.grids)-1].Price
	} else {
		t.last = t.grids[0].Price
	}
	c := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("close get price ")
			runtime.Goexit()
		default:
		}
		c++
		time.Sleep(time.Second * 1)
		t.GetMoeny()                                       // 获取当前money和持仓
		price, err := t.ex.huobi.GetPrice(t.symbol.Symbol) //获取当前价格
		log.Println("当前盈利", "单数:", t.base, "下跌:", price.Sub(t.last).Div(t.last), "上次交易:", t.last, "当前价格：", price, "持仓:", t.amount, "回降比例:", t.arg.Reduce)
		if err != nil {
			t.ErrString = err.Error()
			break
		}
		if c > 10 {
			t.over = true
			break
		}
	}
}

func (t *Trader) Buy(price decimal.Decimal) (uint64, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	orderId, err := t.buy(clientOrderId, price, t.grids[t.base].TotalBuy.Div(price).Round(t.symbol.AmountPrecision), t.GetRate())
	return orderId, err
}

func (t *Trader) Sell(price decimal.Decimal, rate float64) (uint64, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	t.amount = t.GetMycoin()
	orderId, err := t.sell(clientOrderId, price.Round(t.symbol.PricePrecision), t.amount.Truncate(t.symbol.AmountPrecision).Round(t.symbol.AmountPrecision), rate)
	// t.amount.RoundBank(-t.symbol.AmountPrecision))
	t.SellOrder = clientOrderId
	return orderId, err
}

func (t *Trader) GetRate() float64 {
	if t.base > 0 && t.base < len(t.grids) {
		return t.arg.AddRate*float64(t.base-1) + t.arg.Rate
	} else {
		return 0
	}
}

func (t *Trader) WaitOrder() bool {
	log.Println("等待订单交易.......")
	start := time.Now()
	for {
		if t.OrderOver {
			return true
		}
		if time.Since(start) >= time.Second*30 {
			return false
		}
	}
}
