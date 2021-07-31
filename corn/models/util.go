/***************************
@File        : util.go
@Time        : 2021/07/23 11:44:04
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package model

import (
	"encoding/json"
	"errors"

	"zmyjobs/corn/exchange/huobi"

	"github.com/shopspring/decimal"
)

// NewSymbol 生成grid.SymbolCategory
func NewSymbol(u User) *SymbolCategory {
	var (
		pricePrecision  float64
		amountPrecision float64
		minAmount       float64
		minToal         float64
		baseCurrency    string
		quoteCurrency   string
	)
	for _, v := range LoadSymbols("火币") {
		if v["symbol"] == u.Name {
			pricePrecision = v["price-precision"].(float64)
			amountPrecision = v["amount-precision"].(float64)
			minAmount = v["min-order-amt"].(float64)
			minToal = v["min-order-value"].(float64)
			baseCurrency = v["base-currency"].(string)
			quoteCurrency = v["quote-currency"].(string)
			// fmt.Println(fmt.Sprintf("%+v",v))
		}
	}
	h := Host{}
	h.Get(u.Category)
	return &SymbolCategory{
		Category:        u.Category,
		Symbol:          u.Name,
		AmountPrecision: int32(amountPrecision),
		PricePrecision:  int32(pricePrecision),
		Key:             u.ApiKey,
		Secret:          u.Secret,
		Host:            h.Url,
		MinTotal:        decimal.NewFromFloat(minToal),
		MinAmount:       decimal.NewFromFloat(minAmount),
		BaseCurrency:    baseCurrency,
		QuoteCurrency:   quoteCurrency,
	}
}

// MakeStrategy 网格模型
func MakeStrategy(u User) (*[]Grid, error) {
	// scale := decimal.NewFromFloat(math.Pow(arg.MinPrice/arg.Price, 1.0/float64(number)))
	arg := StringArg(u.Arg)
	symbol := StringSymobol(u.Symbol)
	currentPrice := decimal.NewFromFloat(arg.Price).Round(symbol.PricePrecision)
	// fmt.Println(fmt.Sprintf("%+v----%+v", arg, symbol))
	preTotal := decimal.NewFromFloat(arg.FirstBuy).Div(currentPrice).Round(symbol.AmountPrecision)
	var grids []Grid
	currentGrid := Grid{
		Id:        1,
		Price:     currentPrice,
		Decline:   arg.Decline,
		AmountBuy: preTotal,
		TotalBuy:  decimal.NewFromFloat(arg.FirstBuy),
	}
	grids = append(grids, currentGrid)
	// 补仓
	for i := 2; i <= int(u.Number); i++ {
		// 补仓比例 10
		decline := grids[i-2].Decline + arg.AddRate*0.01*grids[i-2].Decline // 当前价格跌幅
		price := grids[i-2].Price.Sub(currentPrice.Mul(decimal.NewFromFloat(decline * 0.01)))
		var TotalBuy decimal.Decimal
		if arg.IsChange {
			// 固定金额
			TotalBuy = grids[i-2].TotalBuy.Add(decimal.NewFromFloat(arg.AddMoney))
		} else {
			rate := arg.Rate*0.01 + 1 // 补仓比例
			TotalBuy = grids[i-2].TotalBuy.Mul(decimal.NewFromFloat(rate))
		}
		if TotalBuy.Cmp(symbol.MinTotal) == -1 {
			log.Printf("total %s less than minTotal(%s)", TotalBuy, symbol.MinTotal)
			return nil, errors.New(" total not enough")
		}
		currentGrid = Grid{
			Id:        i,
			Price:     price,
			Decline:   decline,
			AmountBuy: TotalBuy.Div(price).Round(symbol.AmountPrecision),
			TotalBuy:  TotalBuy,
		}
		grids = append(grids, currentGrid)
	}
	if arg.FirstDouble {
		grids[0].TotalBuy = grids[0].TotalBuy.Add(grids[0].TotalBuy)
		grids[0].AmountBuy = grids[0].TotalBuy.Div(grids[0].Price).Round(symbol.AmountPrecision)
	}
	for _, g := range grids {
		log.Printf("id:%2d - - moeny:%s - - 价格: %s - - 买入量: %s - - 卖出:%s--跌幅: %f",
			g.Id, g.TotalBuy.StringFixed(symbol.AmountPrecision+symbol.PricePrecision),
			g.Price.StringFixed(symbol.PricePrecision),
			g.AmountBuy.StringFixed(symbol.AmountPrecision), g.AmountSell.StringFixed(symbol.AmountPrecision), g.Decline)
		if u.Base < g.Id {
			m, _ := g.TotalBuy.Float64()
			arg.NeedMoney += m
		}
	}
	log.Println("需要资金", arg.NeedMoney)
	return &grids, nil
}

// ParseStrategy 解析策略
func ParseStrategy(u User) *Args {

	var data = map[string]interface{}{}
	var arg Args
	_ = json.Unmarshal([]byte(u.Strategy), &data)
	arg.Price, _ = GetPrice(u.Name).Float64()
	u.BasePrice = arg.Price
	arg.FirstBuy = ParseStringFloat(data["FirstBuy"].(string))
	arg.Rate = ParseStringFloat(data["rate"].(string))
	arg.MinBuy = arg.FirstBuy
	arg.AddRate = ParseStringFloat(data["growth"].(string))
	arg.Callback = ParseStringFloat(data["callback"].(string))
	arg.Reduce = ParseStringFloat(data["reduce"].(string))
	arg.Stop = ParseStringFloat(data["stop"].(string))
	arg.AddMoney = ParseStringFloat(data["add"].(string))
	arg.StrategyType = int64(data["Strategy"].(float64))
	if arg.StrategyType == 1 {
		arg.IsChange = false
	}
	if data["frequency"].(float64) == 2 {
		arg.Crile = true
	}
	// log.Println(data)
	arg.Decline = ParseStringFloat(data["decline"].(string)) // 暂设跌幅
	// log.Println(data)
	if data["allSell"].(float64) == 2 {
		OperateCh <- Operate{Id: float64(u.ObjectId), Op: 1}
	}
	if data["one_buy"].(float64) == 2 {
		OperateCh <- Operate{Id: float64(u.ObjectId), Op: 2}
		// arg.OneBuy = true
	}
	if data["limit_high"].(float64) == 2 {
		arg.IsLimit = true
		// arg.LimitHigh = data["high_price"].(float64)
	}
	if data["double"].(float64) == 2 {
		arg.FirstDouble = true
	}
	if data["stop_buy"].(float64) == 2 {
		OperateCh <- Operate{Id: float64(u.ObjectId), Op: 3}
		// arg.StopBuy = true
	}
	if data["order_type"].(float64) == 2 {
		arg.IsHand = true
	}
	// log.Println(fmt.Sprintf("%+v", arg))

	return &arg
}

func GetPrice(symbol string) decimal.Decimal {
	price, err := huobi.GetPriceSymbol(symbol)
	if err == nil {
		return price
	}
	return decimal.NewFromFloat(0)
}

func ToStringJson(v interface{}) string {
	s, _ := json.Marshal(v)
	return string(s)
}
