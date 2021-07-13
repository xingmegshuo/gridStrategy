package grid

import (
	"context"
	"errors"
	"fmt"
	l "log"
	"strings"
	"time"
	"zmyjobs/exchange/huobi"
	"zmyjobs/logs"
	model "zmyjobs/models"

	"github.com/huobirdcenter/huobi_golang/logging/applogger"
	"github.com/huobirdcenter/huobi_golang/pkg/model/order"
	"github.com/shopspring/decimal"
	"github.com/xyths/hs"
)

type SymbolCategory struct {
	Category        string
	Symbol          string
	AmountPrecision int32
	PricePrecision  int32
	Label           string
	Key             string
	Secret          string
	Host            string
	MinTotal        decimal.Decimal
	MinAmount       decimal.Decimal
	BaseCurrency    string
	QuoteCurrency   string
}

var GridDone = make(chan int)
var log = logs.Log

type Cli struct {
	huobi *huobi.Client
}

type Args struct {
	FirstBuy  float64
	Price     float64
	Rate      float64
	MinBuy    float64
	AddRate   float64
	NeedMoney float64
	MinPrice  float64
	Callback  float64
	Reduce    float64
	Stop      float64
	AddMoney  float64
}

type Trader struct {
	ex            *Cli
	symbol        *SymbolCategory
	baseCurrency  string
	quoteCurrency string
	grids         []hs.Grid
	// scale         decimal.Decimal
	base      int
	cost      decimal.Decimal // average price
	amount    decimal.Decimal // amount held
	ErrString string
	last      decimal.Decimal
	u         model.User
	basePrice decimal.Decimal
}

// Run 跑模型交易
func Run(ctx context.Context, grid *[]hs.Grid, u model.User, symbol *SymbolCategory, arg *Args) {
	//var ctx *cli.Context
	status := 0
	for {
		select {
		case <-ctx.Done():
			log.Println("结束了协程2")
			return
		default:
		}
		for i := 0; i < 1; i++ {
			if status == 0 {
				status = 1
				log.Println("开启协程2中的任务")
				g, e := NewGrid(grid, symbol)
				if e != nil {
					u.IsRun = -10
					u.Error = "api 请求超时，或api接口更改"
					u.Update()
					//GridDone <- 1
				} else {
					defer g.Close(ctx)
					log.Println("开启websocket服务连接,真正的执行策略-------")
					g.u = u
					g.Trade(ctx, arg)
				}
			}
		}
	}
}

// NewGrid 实例化对象，并验证api key的正确性
func NewGrid(grid *[]hs.Grid, symbol *SymbolCategory) (*Trader, error) {
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
	}, nil
}

// Trade 创建websocket 并执行策略中的交易任务
func (t *Trader) Trade(ctx context.Context, arg *Args) {
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
				if err := t.ReBalance(ctx, arg); err != nil {
					t.u.IsRun = -10
					t.u.Error = err.Error()
					t.u.Update()
					// 执行报错就关闭
					GridDone <- 1
				} else {
					l.Println("我执行几次")
					t.setupGridOrders(ctx, arg)
					if t.ErrString != "" {
						t.u.IsRun = -10
						t.u.Error = t.ErrString
						t.u.Update()
						// 执行报错就关闭
						GridDone <- 1
					} else {
						// 策略执行完毕
						t.u.IsRun = 1
						t.u.Update()
						GridDone <- 1
					}
				}
			}
		}
	}

}

func (t *Trader) log(price decimal.Decimal, ty string, num int, hold decimal.Decimal, rate float64, pay float64, money float64) {
	p, _ := price.Float64()
	h, _ := hold.Float64()
	info := model.RebotLog{
		Price:     p,
		UserID:    t.u.ID,
		Types:     ty,
		AddNum:    num,
		HoldNum:   h,
		HoldMoney: money,
		PayMoney:  pay,
		AddRate:   rate,
	}
	info.New()
}

// Print 打印网格策略模型
func (t *Trader) Print() error {
	//delta, _ := t.scale.Float64()
	//delta = 1 - delta
	//log.Printf("Scale is %s (%1.2f%%)", t.scale.String(), 100*delta)
	log.Printf("Id\tTotal\tPrice\tAmountBuy\tAmountSell")
	for _, g := range t.grids {
		log.Printf("%2d\t%s\t%s\t%s\t%s",
			g.Id, g.TotalBuy.StringFixed(t.symbol.AmountPrecision+t.symbol.PricePrecision),
			g.Price.StringFixed(t.symbol.PricePrecision),
			g.AmountBuy.StringFixed(t.symbol.AmountPrecision), g.AmountSell.StringFixed(t.symbol.AmountPrecision))
	}

	return nil
}

func (t *Trader) ReBalance(ctx context.Context, arg *Args) error {
	price, err := t.ex.huobi.GetPrice(t.symbol.Symbol)
	if err != nil {
		return err
	}
	t.base = 1
	moneyNeed := decimal.NewFromInt(int64(arg.NeedMoney))
	coinNeed := decimal.NewFromInt(0)
	//for i, g := range t.grids {
	//	if g.Price.Cmp(price) == 1 {
	//		t.base = i
	//		coinNeed = coinNeed.Add(g.AmountBuy)
	//	} else {
	//		moneyNeed = moneyNeed.Add(g.TotalBuy)
	//	}
	//}
	//log.Printf("now base = %d, moneyNeed = %s, coinNeed = %s", t.base, moneyNeed, coinNeed)
	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
		return err
	}
	moneyHeld := balance[t.quoteCurrency]
	coinHeld := balance[t.baseCurrency]
	log.Printf("account has money %s, coin %s", moneyHeld, coinHeld)
	t.cost = price
	t.amount = coinNeed
	if moneyNeed.Cmp(moneyHeld) == 1 {
		return errors.New("no enough money")
	}
	//direct, amount := t.assetRebalancing(moneyNeed, coinNeed, moneyHeld, coinHeld, price)
	//if direct == -2 || direct == 2 {
	//	t.ErrString = fmt.Sprintf("no enough money for rebalance, direct: %d", direct)
	//	return errors.New("money not enough")
	//} else if direct == 0 {
	//	log.Println("no need to rebalance")
	//} else if direct == -1 {
	//	// place sell order
	//	t.base++
	//	clientOrderId := fmt.Sprintf("p-s-%d", time.Now().Unix())
	//	orderId, err := t.sell(clientOrderId, price, amount)
	//	if err != nil {
	//		t.ErrString = fmt.Sprintf("error when rebalance: %s", err)
	//		return err
	//	}
	//	log.Printf("rebalance: sell %s coin at price %s, orderId is %d, clientOrderId is %s",
	//		amount, price, orderId, clientOrderId)
	//} else if direct == 1 {
	//	// place buy order
	//	clientOrderId := fmt.Sprintf("p-b-%d", time.Now().Unix())
	//	orderId, err := t.buy(clientOrderId, price, amount)
	//	if err != nil {
	//		t.ErrString = fmt.Sprintf("error when rebalance: %s", err)
	//		return err
	//	}
	//	log.Printf("rebalance: buy %s coin at price %s, orderId is %d, clientOrderId is %s",
	//		amount, price, orderId, clientOrderId)
	//}
	return nil
}

func (t *Trader) Close(ctx context.Context) {
	// _ = t.mdb.Client().Disconnect(ctx)
	log.Println("----------策略退出")
	t.cancelAllOrders(ctx)
	//runtime.Goexit()
}

func (t *Trader) OrderUpdateHandler(response interface{}) {
	subOrderResponse, ok := response.(order.SubscribeOrderV2Response)
	if !ok {
		log.Printf("Received unknown response: %v", response)
	}
	//log.Printf("subOrderResponse = %#v", subOrderResponse)
	if subOrderResponse.Action == "sub" {
		if subOrderResponse.IsSuccess() {
			log.Printf("Subscription topic %s successfully", subOrderResponse.Ch)
		} else {
			log.Fatalf("Subscription topic %s error, code: %d, message: %s",
				subOrderResponse.Ch, subOrderResponse.Code, subOrderResponse.Message)
		}
	} else if subOrderResponse.Action == "push" {
		if subOrderResponse.Data == nil {
			log.Printf("SubscribeOrderV2Response has no data: %#v", subOrderResponse)
			return
		}
		o := subOrderResponse.Data
		//log.Printf("Order update, event: %s, symbol: %s, type: %s, id: %d, clientId: %s, status: %s",
		//	o.EventType, o.Symbol, o.Type, o.OrderId, o.ClientOrderId, o.OrderStatus)
		switch o.EventType {
		case "creation":
			log.Printf("order created, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)
		case "cancellation":
			log.Printf("order cancelled, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)
		case "trade":
			log.Printf("order filled, orderId: %d, clientOrderId: %s, fill type: %s", o.OrderId, o.ClientOrderId, o.OrderStatus)
			go t.processOrderTrade(o.TradeId, uint64(o.OrderId), o.ClientOrderId, o.OrderStatus, o.TradePrice, o.TradeVolume, o.RemainAmt)
		default:
			log.Printf("unknown eventType, should never happen, orderId: %d, clientOrderId: %s, eventType: %s",
				o.OrderId, o.ClientOrderId, o.EventType)
		}
	}
}

func (t *Trader) TradeClearHandler(response interface{}) {
	subResponse, ok := response.(order.SubscribeTradeClearResponse)
	if ok {
		if subResponse.Action == "sub" {
			if subResponse.IsSuccess() {
				applogger.Info("Subscription TradeClear topic %s successfully", subResponse.Ch)
			} else {
				applogger.Error("Subscription TradeClear topic %s error, code: %d, message: %s",
					subResponse.Ch, subResponse.Code, subResponse.Message)
			}
		} else if subResponse.Action == "push" {
			if subResponse.Data == nil {
				log.Printf("SubscribeOrderV2Response has no data: %#v", subResponse)
				return
			}
			o := subResponse.Data
			trade := huobi.Trade{
				Symbol:            subResponse.Data.Symbol,
				OrderId:           subResponse.Data.OrderId,
				OrderType:         subResponse.Data.OrderType,
				Aggressor:         subResponse.Data.Aggressor,
				Id:                subResponse.Data.TradeId,
				Time:              subResponse.Data.TradeTime,
				Price:             decimal.RequireFromString(subResponse.Data.TradePrice),
				Volume:            decimal.RequireFromString(subResponse.Data.TradeVolume),
				TransactFee:       decimal.RequireFromString(subResponse.Data.TransactFee),
				FeeDeduct:         decimal.RequireFromString(subResponse.Data.FeeDeduct),
				FeeDeductCurrency: subResponse.Data.FeeDeductType,
			}
			log.Printf("Order update, symbol: %s, order id: %d, price: %s, volume: %s",
				o.Symbol, o.OrderId, o.TradePrice, o.TradeVolume)
			go t.processClearTrade(trade)
		}
	} else {
		log.Printf("Received unknown response: %v", response)
	}

}

func (t *Trader) processOrderTrade(tradeId int64, orderId uint64, clientOrderId, orderStatus, tradePrice, tradeVolume, remainAmount string) {
	log.Println("process order trade",
		"tradeId", tradeId,
		"orderId", orderId,
		"clientOrderId", clientOrderId,
		"orderStatus", orderStatus,
		"tradePrice", tradePrice,
		"tradeVolume", tradeVolume,
		"remainAmount", remainAmount)
	if strings.HasPrefix(clientOrderId, "p-") {
		log.Printf("rebalance order filled/partial-filled, tradeId: %d, orderId: %d, clientOrderId: %s", tradeId, orderId, clientOrderId)
		return
	}
	// update grid
	remain, err := decimal.NewFromString(remainAmount)
	if err != nil {
		log.Println("Error when prase remainAmount",
			"error", err,
			"remainAmount", remainAmount)
	}
	if !remain.Equal(decimal.NewFromInt(0)) && orderStatus != "filled" {
		log.Println("not full-filled order",
			"orderStatus", orderStatus,
			"remainAmount", remainAmount)
		return
	}
	if strings.HasPrefix(clientOrderId, "b-") {
		// buy order filled
		if orderId != t.grids[t.base+1].Order {
			log.Printf("[ERROR] buy order position is NOT the base+1, base: %d", t.base)
			return
		}
		t.grids[t.base+1].Order = 0
		t.down()
	} else if strings.HasPrefix(clientOrderId, "s-") {
		// sell order filled
		if orderId != t.grids[t.base-1].Order {
			log.Printf("[ERROR] sell order position is NOT the base+1, base: %d", t.base)
			return
		}
		t.grids[t.base-1].Order = 0
		t.up()
	} else {
		log.Printf("I don't know the clientOrderId: %s, maybe it's a manual order: %d", clientOrderId, orderId)
	}
}

func (t *Trader) processClearTrade(trade huobi.Trade) {
	log.Println("process trade clear",
		"tradeId", trade.Id,
		"orderId", trade.OrderId,
		"orderType", trade.OrderType,
		"price", trade.Price,
		"volume", trade.Volume,
		"transactFee", trade.TransactFee,
		"feeDeduct", trade.FeeDeduct,
		"feeDeductCurrency", trade.FeeDeductCurrency,
	)
	oldTotal := t.amount.Mul(t.cost)
	if trade.OrderType == huobi.OrderTypeSellLimit {
		trade.Volume = trade.Volume.Neg()
	}
	t.amount = t.amount.Add(trade.Volume)
	tradeTotal := trade.Volume.Mul(trade.Price)
	newTotal := oldTotal.Add(tradeTotal)
	t.cost = newTotal.Div(t.amount)
	log.Println("Average cost update", "cost", t.cost)
}

func (t *Trader) buy(clientOrderId string, price, amount decimal.Decimal) (uint64, error) {
	log.Printf("[Order][buy] price: %s, amount: %s", price, amount)
	return t.ex.huobi.PlaceOrder(huobi.OrderTypeBuyLimit, t.symbol.Symbol, clientOrderId, price, amount)
}

func (t *Trader) sell(clientOrderId string, price, amount decimal.Decimal) (uint64, error) {
	log.Printf("[Order][sell] price: %s, amount: %s", price, amount)
	return t.ex.huobi.PlaceOrder(huobi.OrderTypeSellLimit, t.symbol.Symbol, clientOrderId, price, amount)
}

func (t *Trader) Buy() {

}

// setupGridOrders 测试
func (t *Trader) setupGridOrders(ctx context.Context, arg *Args) {
	// 计数
	count := 0
	// 跌
	d := 0
	t.base = t.u.Base
	z := 0
	t.last = decimal.NewFromFloat(arg.Price)
	t.basePrice = t.last
	//var now decimal.Decimal
	var (
		low  = t.last
		high = t.last
		base = t.last
	)
	for {
		count++
		time.Sleep(time.Second * 2)
		price, err := t.ex.huobi.GetPrice(t.symbol.Symbol)
		if err != nil {
			t.ErrString = err.Error()
			return
		}
		if t.last.Cmp(price) == 1 {
			z++
		} else {
			d++
		}
		if low.Cmp(price) == 1 {
			low = price
		}
		if high.Cmp(price) == -1 {
			high = price
		}
		// 计算盈利
		win, _ := price.Sub(t.basePrice).Mul(t.amount).Float64()
		if t.base == 1 {
			l.Println("第一次买入:", price, t.grids[t.base].AmountBuy)
			// t.base++
			clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
			orderId, err := t.buy(clientOrderId, price, t.grids[t.base].AmountBuy)
			if err != nil {
				log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
				continue
			} else {
				t.base++
				t.last = price
				low = price
				high = price
				t.amount = t.amount.Add(t.grids[t.base].AmountBuy)
				t.log(price, "限价", t.base, t.amount, 0, 0, 0)
			}
			t.grids[t.base].Order = orderId
		}
		// if win <= -0.15 {
		// 	l.Println("卖出:", price, t.amount)
		// 	t.log(price, "限价", t.base, t.amount, 0, 0, 0)
		// 	return
		// }
		if count%2 == 0 {
			// 上涨
			if z > d {
				// 现在价值,等待盈利卖出
				s, _ := high.Sub(price).Div(t.basePrice).Float64()
				if win >= arg.Stop || s > arg.Reduce {
					l.Println("卖出:", price, t.amount)

					clientOrderId := fmt.Sprintf("s-%d-%d", t.base, time.Now().Unix())
					orderId, err := t.sell(clientOrderId, price, t.amount)
					if err != nil {
						log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
						continue
					} else {
						t.log(price, "限价/卖出", t.base, t.amount, float64(orderId), 0, 0)
					}
				}
				return
			}
		} else {
			// 下跌
			if t.base != len(t.grids) {
				c, _ := low.Sub(t.last).Div(base).Float64()
				if c >= arg.AddRate*float64(t.base-2)+arg.Rate {
					// 继续跌
					// l.Println("第次买入:", t.base, price, t.grids[t.base].AmountBuy, c, win)
					// t.base++
					clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
					orderId, err := t.buy(clientOrderId, price, t.grids[t.base].AmountBuy)
					if err != nil {
						log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
						continue
					} else {
						t.base++
						t.last = price
						t.amount = t.amount.Add(t.grids[t.base].AmountBuy)
						low = price
						high = price
						t.log(price, "限价/卖出", t.base, t.amount, 0, arg.AddRate*float64(t.base-2)+arg.Rate, 0)
					}
					t.grids[t.base].Order = orderId
				}
			} else {
				// 最后一单 回调
				c, _ := price.Sub(low).Div(base).Float64()
				if c >= arg.Callback {
					l.Println("最后一次买入:", price, t.grids[t.base].AmountBuy, c)
					clientOrderId := fmt.Sprintf("b-%d-%d", t.base, time.Now().Unix())
					orderId, err := t.buy(clientOrderId, price, t.grids[t.base].AmountBuy)
					if err != nil {
						log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
						continue
					} else {
						t.base++
						t.last = price
						low = price
						high = price
						t.amount = t.amount.Add(t.grids[t.base].AmountBuy)
						t.log(price, "限价", t.base, t.amount, 0, arg.Callback, 0)
					}
					t.grids[t.base].Order = orderId
				}
			}
		}

		if t.base != t.u.Base {
			t.u.Base = t.base
			t.u.Update()
		}
	}

	//for i := t.base - 1; i >= 0; i-- {
	//	// sell
	//	clientOrderId := fmt.Sprintf("s-%d-%d", i, time.Now().Unix())
	//	orderId, err := t.sell(clientOrderId, t.grids[i].Price, t.grids[i].AmountSell)
	//	if err != nil {
	//		log.Printf("error when setupGridOrders, grid number: %d, err: %s", i, err)
	//		continue
	//	}
	//	t.grids[i].Order = orderId
	//}
	//for i := t.base + 1; i < len(t.grids); i++ {
	//	// buy
	//	clientOrderId := fmt.Sprintf("b-%d-%d", i, time.Now().Unix())
	//	orderId, err := t.buy(clientOrderId, t.grids[i].Price, t.grids[i].AmountBuy)
	//	if err != nil {
	//		log.Printf("error when setupGridOrders, grid number: %d, err: %s", i, err)
	//		continue
	//	}
	//	t.grids[i].Order = orderId
	//}
}

func (t *Trader) cancelAllOrders(ctx context.Context) {
	for i := 0; i < len(t.grids); i++ {
		if t.grids[i].Order == 0 {
			continue
		}
		if ret, err := t.ex.huobi.CancelOrder(t.grids[i].Order); err == nil {
			log.Printf("cancel order successful, orderId: %d", t.grids[i].Order)
			t.grids[i].Order = 0

		} else {
			log.Printf("cancel order error: orderId: %d, return code: %d, err: %s", t.grids[i].Order, ret, err)
		}
	}
}

// 0: no need
// 1: buy
// -1: sell
// 2: no enough money
// -2: no enough coin
func (t *Trader) assetRebalancing(moneyNeed, coinNeed, moneyHeld, coinHeld, price decimal.Decimal) (direct int, amount decimal.Decimal) {
	if moneyNeed.GreaterThan(moneyHeld) {
		// sell coin
		moneyDelta := moneyNeed.Sub(moneyHeld)
		sellAmount := moneyDelta.Div(price).Round(t.symbol.AmountPrecision)
		if coinHeld.Cmp(coinNeed.Add(sellAmount)) == -1 {
			log.Errorf("no enough coin for rebalance: need hold %s and sell %s (%s in total), only have %s",
				coinNeed, sellAmount, coinNeed.Add(sellAmount), coinHeld)
			direct = -2
			return
		}

		if sellAmount.LessThan(t.symbol.MinAmount) {
			log.Printf("sell amount %s less than minAmount(%s), won't sell", sellAmount, t.symbol.MinAmount)
			direct = 0
			return
		}
		if t.symbol.MinTotal.GreaterThan(price.Mul(sellAmount)) {
			log.Printf("sell total %s less than minTotal(%s), won't sell", price.Mul(sellAmount), t.symbol.MinTotal)
			direct = 0
			return
		}
		direct = -1
		amount = sellAmount
	} else {
		// buy coin
		if coinNeed.LessThan(coinHeld) {
			log.Printf("no need to rebalance: need coin %s, has %s, need money %s, has %s",
				coinNeed, coinHeld, moneyNeed, moneyHeld)
			direct = 0
			return
		}
		coinDelta := coinNeed.Sub(coinHeld).Round(t.symbol.AmountPrecision)
		buyTotal := coinDelta.Mul(price)
		if moneyHeld.LessThan(moneyNeed.Add(buyTotal)) {
			log.Printf("no enough money for rebalance: need hold %s and spend %s (%s in total)，only have %s",
				moneyNeed, buyTotal, moneyNeed.Add(buyTotal), moneyHeld)
			direct = 2
			return
		}
		if coinDelta.LessThan(t.symbol.MinAmount) {
			log.Printf("buy amount %s less than minAmount(%s), won't sell", coinDelta, t.symbol.MinAmount)
			direct = 0
			return
		}
		if buyTotal.LessThan(t.symbol.MinTotal) {
			log.Printf("buy total %s less than minTotal(%s), won't sell", buyTotal, t.symbol.MinTotal)
			direct = 0
			return
		}
		direct = 1
		amount = coinDelta
	}
	return
}

func (t *Trader) up() {
	// make sure base >= 0
	if t.base == 0 {
		log.Printf("grid base = 0, up OUT")
		return
	}
	t.base = t.base - 1
	if t.base < len(t.grids)-2 {
		// place buy order
		clientOrderId := fmt.Sprintf("b-%d-%d", t.base+1, time.Now().Unix())
		if orderId, err := t.buy(clientOrderId, t.grids[t.base+1].Price, t.grids[t.base+1].AmountBuy); err == nil {
			t.grids[t.base+1].Order = orderId
		}
	}
}

func (t *Trader) down() {
	// make sure base <= len(grids)
	if t.base == len(t.grids) {
		log.Printf("grid base = %d, down OUT", t.base)
		return
	}
	t.base++
	if t.base > 0 {
		// place sell order
		clientOrderId := fmt.Sprintf("s-%d-%d", t.base-1, time.Now().Unix())
		if orderId, err := t.sell(clientOrderId, t.grids[t.base-1].Price, t.grids[t.base-1].AmountSell); err == nil {
			t.grids[t.base-1].Order = orderId
		}
	}
}

func (t *Trader) ErrLoad() string {
	return t.ErrString
}

func GetPrice(symbol string) decimal.Decimal {
	price, err := huobi.GetPriceSymbol(symbol)
	if err == nil {
		return price
	}
	return decimal.NewFromFloat(0)
}
