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
	"strconv"

	"zmyjobs/corn/exchange/huobi"

	"github.com/shopspring/decimal"
)

var (
	HUOBI_MIN = decimal.NewFromInt(5)
	BIAN_MIN  = decimal.NewFromFloat(10)
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
		Label:           "goex-" + u.Category + "-" + u.Name + "-" + strconv.Itoa(int(u.ObjectId)),
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
	if CheckMinToal(u.Category, decimal.NewFromFloat(arg.FirstBuy)) != nil {
		log.Printf("total %s less than minTotal(%s)", decimal.NewFromFloat(arg.FirstBuy), symbol.MinTotal)
		return nil, errors.New(" total not enough")
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
	log.Println(u.ObjectId, "需要资金", arg.NeedMoney)
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
	// log.Println("用户修改:", u.ObjectId, u.IsRun)
	if arg.StrategyType == 1 {
		arg.IsChange = false
	}
	if data["frequency"].(float64) == 2 {
		arg.Crile = true
	}
	// log.Println(data)
	arg.Decline = ParseStringFloat(data["decline"].(string)) // 暂设跌幅
	if data["allSell"].(float64) == 2 {
		if UpdateStatus(u.ID) == 10 {
			log.Println("发送清仓", u.ObjectId)
			OperateCh <- Operate{Id: float64(u.ObjectId), Op: 1}
		}
	}
	if data["one_buy"].(float64) == 2 {
		if UpdateStatus(u.ID) == 10 {
			log.Println("发送补仓", u.ObjectId)
			OperateCh <- Operate{Id: float64(u.ObjectId), Op: 2}
		}
	}
	if data["limit_high"].(float64) == 2 {
		arg.IsLimit = true
		if data["high_price"] != nil {
			arg.LimitHigh = ParseStringFloat(data["high_price"].(string))
		}
	}
	if data["double"].(float64) == 2 {
		arg.FirstDouble = true
	}
	if data["stop_buy"].(float64) == 2 && UpdateStatus(u.ID) == 10 {
		log.Println("发送停止买入", u.ObjectId)
		OperateCh <- Operate{Id: float64(u.ObjectId), Op: 3}
	}
	if data["order_type"].(float64) == 2 {
		arg.IsHand = true
	}
	if arg.StrategyType == 3 || arg.StrategyType == 4 {
		arg.OrderType = 2
	}
	return &arg
}

func GetPrice(symbol string) decimal.Decimal {
	price, err := huobi.GetPriceSymbol(symbol)
	if err == nil {
		return price
	} else {
		log.Println(err)
	}
	return decimal.NewFromFloat(1)
}

func ToStringJson(v interface{}) string {
	s, _ := json.Marshal(v)
	return string(s)
}

func CheckMinToal(name string, f decimal.Decimal) (e error) {
	switch name {
	case "币安":
		if BIAN_MIN.Cmp(f) == 1 {
			e = errors.New("sorry first should late 10 ")
		}
	default:
		if HUOBI_MIN.Cmp(f) == 1 {
			e = errors.New("sorry first should late 5 ")
		}
	}
	return
}

// StringArg string to
func StringArg(data string) (a Args) {
	_ = json.Unmarshal([]byte(data), &a)
	return
}

func ArgString(a *Args) string {
	s, _ := json.Marshal(&a)
	return string(s)
}

func StringSymobol(data string) (a SymbolCategory) {
	_ = json.Unmarshal([]byte(data), &a)
	return
}
