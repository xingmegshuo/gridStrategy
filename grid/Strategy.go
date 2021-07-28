package grid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	symbol := model.StringSymobol(u.Symbol)
	arg := model.StringArg(u.Arg)
	grid, _ := model.SourceStrategy(u, false)
	var realGrid []model.Grid

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
						model.RunOver(t.u.ObjectId, t.u.BasePrice)
						model.LogStrategy(t.symbol.Category, t.symbol.QuoteCurrency, t.u.ObjectId,
							t.u.Custom, t.amount, t.cost, t.arg.IsHand, t.CalCulateProfit().Abs())
						t.u.Update()
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
	t.setupBi(ctx)
}

func (t *Trader) Buy(price decimal.Decimal) (uint64, string, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	orderId, err := t.buy(clientOrderId, price, t.grids[t.base].TotalBuy.Div(price).Round(t.symbol.AmountPrecision), t.GetRate())
	return orderId, clientOrderId, err
}

func (t *Trader) Sell(price decimal.Decimal, rate float64) (uint64, string, error) {
	clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
	t.amount = t.CountBuy()
	orderId, err := t.sell(clientOrderId, price.Round(t.symbol.PricePrecision), t.amount.Truncate(t.symbol.AmountPrecision).Round(t.symbol.AmountPrecision), rate)
	// t.amount.RoundBank(-t.symbol.AmountPrecision))
	t.SellOrder = clientOrderId
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
	orderId, clientOrder, err := t.Sell(price, rate)
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
	if len(t.u.RealGrids) > 0 {
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
	data, b, err := t.ex.huobi.SearchOrder(clientOrderId)
	if err == nil {
		if b {
			if data != nil {
				amount, _ := decimal.NewFromString(data["amount"])
				price, _ := decimal.NewFromString(data["price"])
				transact, _ := decimal.NewFromString(data["fee"])
				oldTotal := t.amount.Mul(t.cost)
				t.GetMoeny()
				if clientOrderId == t.SellOrder {
					t.SellMoney = t.SellMoney.Add(price.Mul(amount)).Abs().Sub(transact)
					t.RealGrids[t.base-1].AmountSell = t.SellMoney
					t.over = true
					hold := t.GetMycoin()
					model.RebotUpdateBy(client, t.RealGrids[t.base-1].Price, amount.Abs(), transact, t.RealGrids[t.base-1].AmountSell, t.hold, "成功")
					model.AsyncData(t.u.ObjectId, hold, price, hold.Mul(price), t.base)
				} else {
					t.amount = t.amount.Add(amount).Sub(transact)
					tradeTotal := amount.Mul(price)
					newTotal := oldTotal.Add(tradeTotal)
					t.cost = newTotal.Div(t.amount)
					t.TradeGrid()
					t.RealGrids[t.base].AmountBuy = amount.Sub(transact)
					t.RealGrids[t.base].Price = price
					t.RealGrids[t.base].TotalBuy = t.RealGrids[t.base].Price.Mul(amount)
					model.RebotUpdateBy(client, t.RealGrids[t.base].Price, t.RealGrids[t.base].AmountBuy, transact, t.RealGrids[t.base].TotalBuy, t.hold, "成功")
					t.pay = t.pay.Add(t.RealGrids[t.base].TotalBuy)
					model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
				}
				return true
			}
		}
	}
	return false
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
