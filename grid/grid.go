package grid

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	arg           *Args
	ex            *Cli
	symbol        *SymbolCategory
	baseCurrency  string
	quoteCurrency string
	grids         []hs.Grid
	base          int
	cost          decimal.Decimal // average price
	amount        decimal.Decimal // amount held
	ErrString     string
	last          decimal.Decimal // 上次交易价格
	u             model.User
	basePrice     decimal.Decimal // 第一次交易价格
	over          bool
	hold          decimal.Decimal // 余额
	pay           decimal.Decimal
	SellOrder     uint64
	OrderOver     bool // 交易状态
	Locks         sync.Mutex
}

func (t *Trader) log(orderId string, price decimal.Decimal, ty string, num int,
	hold decimal.Decimal, rate float64, buy string) {
	p, _ := price.Float64()
	h, _ := hold.Float64()
	pay, _ := price.Mul(hold).Float64()
	money, _ := price.Mul(hold).Float64()
	t.GetMoeny()
	account, _ := t.hold.Float64()
	var b, q string
	if buy == "买入" {
		b = t.symbol.QuoteCurrency
		q = t.symbol.BaseCurrency
	} else {
		q = t.symbol.BaseCurrency
		b = t.symbol.QuoteCurrency
	}
	info := model.RebotLog{
		BaseCoin:     b,
		GetCoin:      q,
		OrderId:      orderId,
		Price:        p,
		UserID:       t.u.ID,
		Types:        ty,
		AddNum:       num,
		HoldNum:      h,
		HoldMoney:    money,
		PayMoney:     pay,
		AddRate:      rate,
		BuyOrSell:    buy,
		Status:       "创建订单",
		AccountMoney: account,
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

func (t *Trader) ReBalance(ctx context.Context) error {
	t.base = t.u.Base // 从暂停中恢复
	price, err := t.ex.huobi.GetPrice(t.symbol.Symbol)
	if err != nil {
		return err
	}
	moneyNeed := decimal.NewFromInt(int64(t.arg.NeedMoney)) // 策略需要总金额
	t.cost = price

	for i := 0; i < len(t.grids); i++ {
		if i < t.base {
			moneyNeed = moneyNeed.Sub(t.grids[i].TotalBuy)
			t.cost = t.grids[i].Price.Add(t.cost)
		}
	}
	if t.base > 0 {
		t.cost = t.cost.Div(decimal.NewFromInt(int64(t.base))) // 持仓均价
	}
	// coinNeed := decimal.NewFromInt(0)

	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
		return err
	}
	moneyHeld := balance[t.symbol.QuoteCurrency]
	coinHeld := balance[t.symbol.BaseCurrency]
	log.Printf("account has money %s, coin %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
	t.amount, _ = decimal.NewFromString(t.u.Total) // 当前持仓
	log.Println("资产:", moneyNeed, moneyHeld, balance)
	if moneyNeed.Cmp(moneyHeld) == 1 && t.base == 0 {
		return errors.New("no enough money")
	}
	return nil
}

func (t *Trader) GetMoeny() {
	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
	}
	moneyHeld := balance[t.symbol.QuoteCurrency]
	t.hold = moneyHeld
	// t.amount = balance[t.symbol.BaseCurrency]
}

func (t *Trader) GetMycoin() decimal.Decimal {
	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
	}
	// moneyHeld := balance[t.symbol.QuoteCurrency]
	return balance[t.symbol.BaseCurrency]

}

func (t *Trader) Close(ctx context.Context) {
	// _ = t.mdb.Client().Disconnect(ctx)
	log.Println("----------策略退出")
}

func (t *Trader) OrderUpdateHandler(response interface{}) {
	subOrderResponse, ok := response.(order.SubscribeOrderV2Response)
	if !ok {
		log.Printf("Received unknown response: %v", response)
	}
	//log.Printf("subOrderResponse = %#v", subOrderResponse)
	if subOrderResponse.Action == "sub" {
		if subOrderResponse.IsSuccess() {
			// o := subOrderResponse.Data
			log.Printf("Subscription topic %s successfully", subOrderResponse.Ch)
			// model.RebotUpdateBy(o.ClientOrderId, "成功")
			// model.AsyncData(t.u.ObjectId, t.amount, GetPrice(t.symbol.Symbol), t.hold)
			// if o.OrderId == int64(t.SellOrder) {
			// 	t.over = true
			// }
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
		log.Printf("Order update, event: %s, symbol: %s, type: %s, id: %d, clientId: %s, status: %s",
			o.EventType, o.Symbol, o.Type, o.OrderId, o.ClientOrderId, o.OrderStatus)
		switch o.EventType {
		case "creation":
			log.Printf("order created, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)

		case "cancellation":
			log.Printf("order cancelled, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)
		case "trade":
			t.OrderOver = true
			log.Printf("交易成功 order filled, orderId: %d, clientOrderId: %s, fill type: %s", o.OrderId, o.ClientOrderId, o.OrderStatus)
			t.GetMoeny()
			model.RebotUpdateBy(o.ClientOrderId, t.hold, "成功")
			model.AsyncData(t.u.ObjectId, t.amount, GetPrice(t.symbol.Symbol), t.hold)

			if o.OrderId == int64(t.SellOrder) {
				t.over = true
			}
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
	// 计算均价
}

func (t *Trader) buy(clientOrderId string, price, amount decimal.Decimal, rate float64) (uint64, error) {
	log.Printf("[Order][buy] price: %s, amount: %s", price, amount)
	orderId, err := t.ex.huobi.PlaceOrder(huobi.OrderTypeBuyLimit, t.symbol.Symbol, clientOrderId, price, amount)
	if err == nil {
		t.log(clientOrderId, price, "限价", t.base, amount, rate, "买入")
		t.grids[t.base].Price = price
	}
	return orderId, err
}
func (t *Trader) sell(clientOrderId string, price, amount decimal.Decimal) (uint64, error) {
	log.Printf("[Order][sell] price: %s, amount: %s", price, amount)
	orderId, err := t.ex.huobi.PlaceOrder(huobi.OrderTypeSellLimit, t.symbol.Symbol, clientOrderId, price, amount)
	if err == nil {
		t.log(clientOrderId, price, "限价", t.base+1, amount, 0, "卖出")
	}
	return orderId, err
}

// func (t *Trader) cancelAllOrders(ctx context.Context) {
// 	for i := 0; i < len(t.grids); i++ {
// 		if t.grids[i].Order == 0 {
// 			continue
// 		}
// 		if ret, err := t.ex.huobi.CancelOrder(t.grids[i].Order); err == nil {
// 			log.Printf("cancel order successful, orderId: %d", t.grids[i].Order)
// 			t.grids[i].Order = 0

// 		} else {
// 			log.Printf("cancel order error: orderId: %d, return code: %d, err: %s", t.grids[i].Order, ret, err)
// 		}
// 	}
// }

// 0: no need
// 1: buy
// -1: sell
// 2: no enough money
// -2: no enough coin
// func (t *Trader) assetRebalancing(moneyNeed, coinNeed, moneyHeld, coinHeld, price decimal.Decimal) (direct int, amount decimal.Decimal) {
// 	if moneyNeed.GreaterThan(moneyHeld) {
// 		// sell coin
// 		moneyDelta := moneyNeed.Sub(moneyHeld)
// 		sellAmount := moneyDelta.Div(price).Round(t.symbol.AmountPrecision)
// 		if coinHeld.Cmp(coinNeed.Add(sellAmount)) == -1 {
// 			log.Errorf("no enough coin for rebalance: need hold %s and sell %s (%s in total), only have %s",
// 				coinNeed, sellAmount, coinNeed.Add(sellAmount), coinHeld)
// 			direct = -2
// 			return
// 		}

// 		if sellAmount.LessThan(t.symbol.MinAmount) {
// 			log.Printf("sell amount %s less than minAmount(%s), won't sell", sellAmount, t.symbol.MinAmount)
// 			direct = 0
// 			return
// 		}
// 		if t.symbol.MinTotal.GreaterThan(price.Mul(sellAmount)) {
// 			log.Printf("sell total %s less than minTotal(%s), won't sell", price.Mul(sellAmount), t.symbol.MinTotal)
// 			direct = 0
// 			return
// 		}
// 		direct = -1
// 		amount = sellAmount
// 	} else {
// 		// buy coin
// 		if coinNeed.LessThan(coinHeld) {
// 			log.Printf("no need to rebalance: need coin %s, has %s, need money %s, has %s",
// 				coinNeed, coinHeld, moneyNeed, moneyHeld)
// 			direct = 0
// 			return
// 		}
// 		coinDelta := coinNeed.Sub(coinHeld).Round(t.symbol.AmountPrecision)
// 		buyTotal := coinDelta.Mul(price)
// 		if moneyHeld.LessThan(moneyNeed.Add(buyTotal)) {
// 			log.Printf("no enough money for rebalance: need hold %s and spend %s (%s in total)，only have %s",
// 				moneyNeed, buyTotal, moneyNeed.Add(buyTotal), moneyHeld)
// 			direct = 2
// 			return
// 		}
// 		if coinDelta.LessThan(t.symbol.MinAmount) {
// 			log.Printf("buy amount %s less than minAmount(%s), won't sell", coinDelta, t.symbol.MinAmount)
// 			direct = 0
// 			return
// 		}
// 		if buyTotal.LessThan(t.symbol.MinTotal) {
// 			log.Printf("buy total %s less than minTotal(%s), won't sell", buyTotal, t.symbol.MinTotal)
// 			direct = 0
// 			return
// 		}
// 		direct = 1
// 		amount = coinDelta
// 	}
// 	return
// }

// func (t *Trader) up() {
// 	// make sure base >= 0
// 	if t.base == 0 {
// 		log.Printf("grid base = 0, up OUT")
// 		return
// 	}
// 	t.base = t.base - 1
// 	if t.base < len(t.grids)-2 {
// 		// place buy order
// 		clientOrderId := fmt.Sprintf("b-%d-%d", t.base+1, time.Now().Unix())
// 		if orderId, err := t.buy(clientOrderId, t.grids[t.base+1].Price, t.grids[t.base+1].AmountBuy); err == nil {
// 			t.grids[t.base+1].Order = orderId
// 		}
// 	}
// }

// func (t *Trader) down() {
// 	// make sure base <= len(grids)
// 	if t.base == len(t.grids) {
// 		log.Printf("grid base = %d, down OUT", t.base)
// 		return
// 	}
// 	t.base++
// 	if t.base > 0 {
// 		// place sell order
// 		clientOrderId := fmt.Sprintf("s-%d-%d", t.base-1, time.Now().Unix())
// 		if orderId, err := t.sell(clientOrderId, t.grids[t.base-1].Price, t.grids[t.base-1].AmountSell); err == nil {
// 			t.grids[t.base-1].Order = orderId
// 		}
// 	}
// }

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
