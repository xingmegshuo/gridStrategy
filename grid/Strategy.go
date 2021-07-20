package grid

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"zmyjobs/exchange/huobi"
	model "zmyjobs/models"

	"github.com/shopspring/decimal"
	"github.com/xyths/hs"
)

// Run 跑模型交易
func Run(ctx context.Context, grid *[]hs.Grid, u model.User, symbol *SymbolCategory, arg *Args) {
	//var ctx *cli.Context
	log.Println("i am string .......")
	status := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("结束了协程2")
			return
		default:
		}
		// time.Sleep(time.Second * 3)
		// log.Println("i am run", status)
		for i := 0; i < 1; i++ {
			if status == 0 {
				status = 1
				log.Println("开启协程2中的任务")
				g, e := NewGrid(grid, symbol, arg)
				// g.Locks.Lock()
				if e != nil {
					u.IsRun = -10
					u.Error = "api 请求超时，或api接口更改"
					u.Update()
					GridDone <- 1
				} else {
					defer g.Close(ctx)
					g.u = u
					go g.Trade(ctx)

					// g.Locks.Unlock()
					// log.Println("GAN .........")
					// runtime.Goexit()
				}
			}
		}
	}
}

// NewGrid 实例化对象，并验证api key的正确性
func NewGrid(grid *[]hs.Grid, symbol *SymbolCategory, arg *Args) (*Trader, error) {

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
		grids:  *grid,
		ex:     cli,
		symbol: symbol,
		arg:    arg,
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
			log.Println("协程2中的websocket被暂停")
			return
		default:
		}
		for i := 0; i < 1; i++ {
			if c == 0 {
				c = 1
				log.Println("开启协程2中任务的websocket")

				go t.ex.huobi.SubscribeOrder(ctx, t.symbol.Symbol, clientId, t.OrderUpdateHandler)
				go t.ex.huobi.SubscribeTradeClear(ctx, t.symbol.Symbol, clientId, t.TradeClearHandler)
				if err := t.ReBalance(ctx); err != nil {
					t.u.IsRun = -10
					t.u.Error = err.Error()
					t.u.Update()
					// 执行报错就关闭
					GridDone <- 1
				} else {
					// t.over = false
					for {
						for i := 0; i < len(t.grids); i++ {
							if i < t.base {
								t.pay = t.pay.Add(t.grids[i].TotalBuy)
							}
						}
						time.Sleep(time.Second * 10)
						log.Println("testing ............. ")
						// t.setupGridOrders(ctx)
						// if t.ErrString != "" {
						// 	t.u.IsRun = -10
						// 	t.u.Error = t.ErrString
						// 	t.u.Update()
						// 	// 执行报错就关闭
						// 	GridDone <- 1
						// 	break
						// }
						// time.Sleep(time.Second * 30)
						// if t.over && t.base >= len(t.grids) {
						// 	// 策略执行完毕 to do 计算盈利
						// 	t.u.Money, _ = t.cost.Mul(t.amount).Float64()
						// 	// 盈利ctx
						// 	t.u.Money = t.u.Money - t.arg.NeedMoney
						// 	t.u.IsRun = 1
						// 	t.u.Update()
						// 	GridDone <- 1
						// 	break
						// } else {
						// 	t.cancel(t.SellOrder)
						// 	continue
						// }
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
	if t.base > len(t.grids)-1 {
		t.last = t.grids[len(t.grids)-1].Price
	} else {
		t.last = t.grids[t.base].Price // 上次交易价格
	}
	t.basePrice = t.grids[0].Price // 第一次交易价格

	log.Println(t.last, t.basePrice, t.pay, "---------策略开始")
	var (
		low  = t.last
		high = t.last
	)
	for {
		count++
		time.Sleep(time.Second * 2)
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
		t.pay = t.cost.Mul(t.amount) // 当前支付
		if t.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
			// log.Println(t.pay, price.Mul(t.amount))
			win, _ = price.Sub(t.last).Mul(t.amount).Div(t.pay).Float64()
			// log.Println(t.pay, price.Mul(t.amount))
		}
		die, _ := t.last.Sub(price).Div(t.basePrice).Float64()
		if count%10 == 0 {
			// log.Println("支付:", t.pay, "现价值:", price.Mul(t.amount))
			log.Println("当前盈利", win*100, "单数:", t.base, "下跌:", die*100, "上次交易:", t.last, "当前价格：", price, t.amount)
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
						t.grids[t.base-1].Price = price
						t.u.BasePrice, _ = t.last.Float64()
						t.u.Update()

					} else {
						t.cancel(orderId)
					}
				}
			}
		}

		if 0 < t.base && t.base < len(t.grids)-1 {
			// c, _ := price.Sub(t.last).Div(t.basePrice).Float64()
			if die*100 >= float64(3) {
				// +float64(t.base-1)*t.arg.AddRate {
				//  t.GetRate() {
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
						t.grids[t.base].Price = price

					} else {
						t.cancel(orderId)
					}
				}
			}
		}

		//  止盈 t.arg.Stop
		if win*100 > t.arg.Stop {
			log.Println("盈利卖出")
			id, err := t.Sell(price)
			if err != nil {
				log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
				continue
			} else {
				t.SellOrder = id
				break
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

		if t.base == len(t.grids)-1 {
			// 最后一单 回调 当前价-最低价比例
			c, _ := high.Sub(price).Div(t.basePrice).Float64()
			// 满足回调条件最后一单
			if price.Cmp(low) == 1 && c*100 >= t.arg.Callback && price.Cmp(t.grids[t.base].Price) == -1 {
				t.OrderOver = false
				log.Println("最后一次买入:", price, t.grids[t.base].AmountBuy, c)
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
						t.grids[t.base].Price = price
					} else {
						t.cancel(orderId)
					}
				}
			}
		}

		//  如果不相等更新
		if t.base != t.u.Base {
			t.u.Base = t.base
			t.u.Total = t.amount.String()
			s, _ := json.Marshal(t.grids)
			t.u.Grids = string(s)
			t.u.Update()
		}
	}
}

func (t *Trader) Buy(price decimal.Decimal) (uint64, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	orderId, err := t.buy(clientOrderId, price, t.grids[t.base].AmountBuy, t.GetRate())
	return orderId, err
}

func (t *Trader) Sell(price decimal.Decimal) (uint64, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	t.amount = t.GetMycoin()
	orderId, err := t.sell(clientOrderId, price.Round(t.symbol.PricePrecision), t.amount.Truncate(t.symbol.AmountPrecision).Round(t.symbol.AmountPrecision))
	// t.amount.RoundBank(-t.symbol.AmountPrecision))
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
