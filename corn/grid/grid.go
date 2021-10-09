package grid

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
	"zmyjobs/corn/exchange/huobi"
	model "zmyjobs/corn/models"

	"github.com/huobirdcenter/huobi_golang/logging/applogger"
	"github.com/huobirdcenter/huobi_golang/pkg/model/order"
	"github.com/shopspring/decimal"
)

type Cli struct {
	huobi *huobi.Client
}

type Trader struct {
	arg       *model.Args           // 策略参数
	ex        *Cli                  // 平台
	symbol    *model.SymbolCategory // 交易对信息
	grids     []model.Grid          // 预设策略
	RealGrids []model.Grid          // 成交信息
	base      int                   // 当前单数
	ErrString string                // 错误信息
	over      bool                  // 是否结束
	u         model.User            // 用户
	pay       decimal.Decimal       // 投入金额
	cost      decimal.Decimal       // average price
	amount    decimal.Decimal       // amount held
	last      decimal.Decimal       // 上次交易价格
	basePrice decimal.Decimal       // 第一次交易价格
	hold      decimal.Decimal       // 余额
	SellOrder map[string]int        // 卖出订单号
	SellMoney decimal.Decimal       // 卖出金额
	OrderOver bool                  // 一次订单是否结束
	goex      *Cliex                // goex
}

// log 交易日志
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
		pay = 0
	}
	info := model.RebotLog{
		BaseCoin:     b,
		GetCoin:      q,
		OrderId:      orderId,
		Price:        p,
		UserID:       uint(t.u.ObjectId),
		Types:        ty,
		AddNum:       num,
		HoldNum:      h,
		HoldMoney:    money,
		PayMoney:     pay,
		AddRate:      rate,
		BuyOrSell:    buy,
		Status:       "创建订单",
		AccountMoney: account,
		Category:     t.symbol.Category,
		Custom:       uint(t.u.Custom),
	}
	info.New()
}

func (t *Trader) ReBalance(ctx context.Context) error {
	t.base = t.u.Base // 从暂停中恢复

	// **
	price, err := t.ex.huobi.GetPrice(t.symbol.Symbol)
	if err != nil {
		return err
	}
	t.cost = price
	if t.base > 0 {
		t.cost = t.cost.Div(decimal.NewFromInt(int64(t.base))) // 持仓均价
	}

	moneyNeed := decimal.NewFromInt(int64(t.arg.NeedMoney)) // 策略需要总金额
	for i := 0; i < len(t.grids); i++ {
		if i >= t.base {
			moneyNeed = moneyNeed.Add(t.grids[i].TotalBuy)
		}
	}
	t.amount = t.CountBuy()

	// coinNeed := decimal.NewFromInt(0)

	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
		return err
	}
	moneyHeld := balance[t.symbol.QuoteCurrency]
	coinHeld := balance[t.symbol.BaseCurrency]
	log.Printf("account has money %s, coin %s,orderFor %d", moneyHeld, coinHeld, t.u.ObjectId)
	// t.amount, _ = decimal.NewFromString(t.u.Total) // 当前持仓
	if moneyNeed.Cmp(moneyHeld) == 1 {
		return errors.New("no enough money")
	}
	return nil
}

func (t *Trader) GetMoeny() {
	// **
	balance, err := t.ex.huobi.GetSpotBalance()
	if err != nil {
		t.ErrString = fmt.Sprintf("error when get balance in rebalance: %s", err)
	}
	moneyHeld := balance[t.symbol.QuoteCurrency]
	t.hold = moneyHeld

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
	runtime.Goexit()
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
				ClientOrder:       subResponse.Data.ClientOrderId,
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

// processClearTrade 成交
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

	log.Printf("交易成功 order filled, orderId: %d, clientOrderId: %s, fill type: %s", trade.OrderId, trade.ClientOrder, trade.OrderType)
	time.Sleep(time.Second * 3)

	t.GetMoeny()
	if b, ok := t.SellOrder[trade.ClientOrder]; ok {
		log.Println(trade.Volume, "成交量-----------", b)
		t.SellMoney = trade.Price.Mul(trade.Volume).Add(trade.TransactFee)
		t.RealGrids[b-1].AmountSell = t.SellMoney.Abs()
		hold := t.GetMycoin()
		// model.RebotUpdateBy(trade.ClientOrder, trade.Price, trade.Volume.Abs(), trade.TransactFee, t.SellMoney.Abs(), t.hold, "成功")
		if b == t.base {
			t.over = true
			model.AsyncData(t.u.ObjectId, hold, trade.Price, hold.Mul(trade.Price), 0)
		} else {
			model.AsyncData(t.u.ObjectId, hold, trade.Price, hold.Mul(trade.Price), b-1)
		}
	} else {
		t.TradeGrid()
		t.RealGrids[t.base].AmountBuy = trade.Volume.Sub(trade.TransactFee)
		t.RealGrids[t.base].Price = trade.Price
		t.RealGrids[t.base].TotalBuy = t.RealGrids[t.base].Price.Mul(trade.Volume)
		// t.RealGrids[t.base].Order = uint64(trade.OrderId)
		// model.RebotUpdateBy(trade.ClientOrder, t.RealGrids[t.base].Price, t.RealGrids[t.base].AmountBuy, trade.TransactFee, t.RealGrids[t.base].TotalBuy, t.hold, "成功")
		t.pay = t.pay.Add(t.RealGrids[t.base].TotalBuy)
		t.amount = t.amount.Add(trade.Volume).Sub(trade.TransactFee)
		t.cost = t.pay.Div(t.amount)
		model.AsyncData(t.u.ObjectId, t.amount, t.cost, t.pay, t.base+1)
		tradeTotal := trade.Volume.Mul(trade.Price)
		newTotal := oldTotal.Add(tradeTotal)
		t.cost = newTotal.Div(t.amount)
		log.Println("Average cost update", "cost", t.cost)
	}
	t.OrderOver = true
}

func (t *Trader) buy(clientOrderId string, price, amount decimal.Decimal, rate float64) (uint64, error) {
	orderType := huobi.OrderTypeBuyLimit
	orderTypeString := "限价"
	if t.arg.OrderType == 2 {
		orderType = huobi.OrderTypeBuyMarket
		orderTypeString = "市价"
		price = decimal.Decimal{}
		amount = t.grids[t.base].TotalBuy
	}
	log.Printf("[Order][buy] price: %s, amount: %s", price, amount)
	orderId, err := t.ex.huobi.PlaceOrder(orderType, t.symbol.Symbol, clientOrderId, price, amount)
	// 增加一个真实成交
	t.log(clientOrderId, price, orderTypeString, t.base, amount, rate, "买入")
	t.RealGrids = append(t.RealGrids, model.Grid{
		Id: t.base + 1,
		// Price:   price,
		Decline: rate,
		// Order:   orderId,
	})
	return orderId, err
}
func (t *Trader) sell(clientOrderId string, price, amount decimal.Decimal, rate float64, n int) (uint64, error) {
	if n == t.base {
		n = 1
	}
	orderType := huobi.OrderTypeSellLimit
	orderTypeString := "限价"
	if t.arg.OrderType == 2 {
		orderType = huobi.OrderTypeSellMarket
		orderTypeString = "市价"
		price = decimal.Decimal{}
		// amount = t.grids[n-1].TotalBuy
	}
	log.Printf("[Order][sell] price: %s, amount: %s", price, amount)
	orderId, err := t.ex.huobi.PlaceOrder(orderType, t.symbol.Symbol, clientOrderId, price, amount)
	if err == nil {

		t.log(clientOrderId, price, orderTypeString, n, amount, rate, "卖出")
		t.RealGrids[n-1].Decline = rate
	}
	return orderId, err
}

func (t *Trader) ErrLoad() string {
	return t.ErrString
}
