/***************************
@File        : exchange.go
@Time        : 2021/7/2 15:08
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 生成策略并开启
****************************/

package job

import (
	"context"
	"encoding/json"
	"errors"
	"runtime"
	"time"
	"zmyjobs/grid"
	"zmyjobs/logs"
	model "zmyjobs/models"

	"github.com/shopspring/decimal"
	"github.com/xyths/hs"
)

var log = logs.Log

// RunWG 生成用户策略
func RunWG() {
	//time.Sleep(time.Second)
	users := LoadUsers()
	for _, u := range *users {
		start := 0
		Symbols := LoadSymbols(u.Category)
	OuterLoop:
		for {
			select {
			case v := <-model.Ch:
				if v.Run == 2 || v.Run == 3 {
					if model.UpdateStatus(v.Id) == 10 {
						grid.GridDone <- 1
					}
					break OuterLoop
				}
			default:
			}
			if u.Status == 1 && model.UpdateStatus(u.ID) == int64(-1) && Symbols != nil && start == 0 {
				log.Println("符合要求", model.UpdateStatus(u.ID))
				symbol, arg, straggly := PrintStrategy(u, Symbols)
				if straggly != nil {
					for i := 1; i < 2; i++ {
						start = 1
						log.Println("协程开始-用户:", u.ObjectId, "--交易币种:", u.Name, straggly, arg, symbol, u.Grids)
						go RunStrategy(&straggly, u, &symbol, arg)
					}
				}
			}
			if model.UpdateStatus(u.ID) == int64(1) && u.Status == 1 {
				_, arg, straggly := PrintStrategy(u, Symbols)
				if straggly != nil && arg.Crile {
					log.Println("重新开始", u.ObjectId)
					u.IsRun = -1
					u.Grids = ""
					u.Total = ""
					u.Base = 0
					u.RunCount++
					u.Update()
					time.Sleep(time.Second * 10)
				}
				runtime.Goexit()
			}
			break OuterLoop
		}
	}
}

// LoadUsers 获取数据库用户内容
func LoadUsers() *[]model.User {
	return model.GetUserJob()
}

// LoadSymbols 获取交易对
func LoadSymbols(name string) []map[string]interface{} {
	switch name {
	case "火币":
		//log.Println(model.GetCache("火币交易对"))
		return model.StringMap(model.GetCache("火币交易对"))
	default:
		return model.StringMap(model.GetCache("火币交易对"))
	}
}

// PrintStrategy 生成策略并输出
func PrintStrategy(u model.User, symbol []map[string]interface{}) (grid.SymbolCategory, *grid.Args, []hs.Grid) {
	var (
		pricePrecision  float64
		amountPrecision float64
		minAmount       float64
		minToal         float64
		baseCurrency    string
		quoteCurrency   string
	)

	for _, v := range symbol {
		if v["symbol"] == u.Name {
			pricePrecision = v["price-precision"].(float64)
			amountPrecision = v["amount-precision"].(float64)
			minAmount = v["min-order-amt"].(float64)
			minToal = v["min-order-value"].(float64)
			baseCurrency = v["base-currency"].(string)
			quoteCurrency = v["quote-currency"].(string)
		}
	}
	arg := ParseStrategy(u)
	if u.Category != "火币" || arg.MinBuy < minToal {
		log.Println("最小买入", arg.MinBuy)
		//  || arg.NeedMoney <= u.Money { 账户余额暂不判断
		return grid.SymbolCategory{}, arg, nil
	}
	NminAmount := decimal.NewFromFloat(minAmount)
	NminToal := decimal.NewFromFloat(minToal)

	strategy, e := MakeStrategy(arg, int(u.Number), int32(amountPrecision), int32(pricePrecision), NminAmount, NminToal, u)

	if e != nil {
		u.Error = e.Error()
		u.Status = float64(0)
		u.IsRun = int64(-10)
		u.Update()
		return grid.SymbolCategory{}, arg, nil
	} else {
		h := model.Host{}
		h.Get(u.Category)
		symbolData := grid.SymbolCategory{
			Category:        u.Category,
			Symbol:          u.Name,
			AmountPrecision: int32(amountPrecision),
			PricePrecision:  int32(pricePrecision),
			Key:             u.ApiKey,
			Secret:          u.Secret,
			Host:            h.Url,
			MinTotal:        NminToal,
			MinAmount:       NminAmount,
			BaseCurrency:    baseCurrency,
			QuoteCurrency:   quoteCurrency,
		}
		return symbolData, arg, strategy
	}
}

// MakeStrategy 网格模型
func MakeStrategy(arg *grid.Args, number int, amountPrecision int32,
	pricePrecision int32, minAmount decimal.Decimal, minTotal decimal.Decimal, u model.User) ([]hs.Grid, error) {
	// scale := decimal.NewFromFloat(math.Pow(arg.MinPrice/arg.Price, 1.0/float64(number)))

	currentPrice := decimal.NewFromFloat(arg.Price).Round(pricePrecision)
	preTotal := decimal.NewFromFloat(arg.FirstBuy).Div(currentPrice).Round(amountPrecision)
	var grids []hs.Grid
	// 如果策略已经存在
	if s, ok := LoadStrategy(u); ok {
		for i := 2; i <= number; i++ {
			// 补仓比例 10
			if u.Base <= i-1 {
				decline := arg.Decline + arg.AddRate // 当前价格跌幅
				(*s)[i-1].Price = (*s)[i-2].Price.Sub(currentPrice.Mul(decimal.NewFromFloat(decline * 0.01)))
				if arg.IsChange {
					// 固定金额
					(*s)[i-1].TotalBuy = (*s)[i-2].TotalBuy.Add(decimal.NewFromFloat(arg.AddMoney))
				} else {
					rate := arg.Rate*0.01 + 1 // 补仓比例
					(*s)[i-1].TotalBuy = (*s)[i-2].TotalBuy.Mul(decimal.NewFromFloat(rate))
				}
				(*s)[i-1].AmountBuy = (*s)[i-1].TotalBuy.Div((*s)[i-1].Price).Round(amountPrecision)
			}
		}
		grids = *s
	} else {
		// 第一次买入
		currentGrid := hs.Grid{
			Id:        1,
			Price:     currentPrice,
			AmountBuy: preTotal,
			TotalBuy:  decimal.NewFromFloat(arg.FirstBuy),
		}
		grids = append(grids, currentGrid)
		// 补仓
		for i := 2; i <= number; i++ {
			// 补仓比例 10
			decline := arg.Decline + arg.AddRate // 当前价格跌幅
			price := grids[i-2].Price.Sub(currentPrice.Mul(decimal.NewFromFloat(decline * 0.01)))
			var TotalBuy decimal.Decimal
			if arg.IsChange {
				// 固定金额
				TotalBuy = grids[i-2].TotalBuy.Add(decimal.NewFromFloat(arg.AddMoney))
			} else {
				rate := arg.Rate*0.01 + 1 // 补仓比例
				TotalBuy = grids[i-2].TotalBuy.Mul(decimal.NewFromFloat(rate))
			}
			if TotalBuy.Cmp(minTotal) == -1 {
				log.Printf("total %s less than minTotal(%s)", TotalBuy, minTotal)
				return nil, errors.New(" total not enough")
			}
			currentGrid = hs.Grid{
				Id:        i,
				Price:     price,
				AmountBuy: TotalBuy.Div(price).Round(amountPrecision),
				TotalBuy:  TotalBuy,
			}
			grids = append(grids, currentGrid)
		}
	}
	for _, g := range grids {
		log.Printf("%2d - - %s - - %s - - %s - - %s",
			g.Id, g.TotalBuy.StringFixed(amountPrecision+pricePrecision),
			g.Price.StringFixed(pricePrecision),
			g.AmountBuy.StringFixed(amountPrecision), g.AmountSell.StringFixed(amountPrecision))
		if u.Base < g.Id {
			m, _ := g.TotalBuy.Float64()
			arg.NeedMoney += m
		}
	}
	log.Println("需要资金", arg.NeedMoney)
	return grids, nil
}

// RunStrategy 开启策略
func RunStrategy(s *[]hs.Grid, u model.User, symbol *grid.SymbolCategory, arg *grid.Args) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
OuterLoop:
	for {
		select {
		case <-grid.GridDone:
			log.Println("收到消息,暂停策略,exiting ......", u.ObjectId)
			if model.UpdateStatus(u.ID) == 10 {
				u.IsRun = 2
				u.Update()
			}
			break OuterLoop
		default:
		}
		// 执行任务不是一次执行
		if model.UpdateStatus(u.ID) == -1 {
			// log.Println("what 1")
			gr, _ := json.Marshal(&s)
			u.Grids = string(gr)
			u.IsRun = 10
			u.Update()
			for i := 0; i < 1; i++ {
				go grid.Run(ctx, s, u, symbol, arg)
				// log.Println("what is waring  ")
			}
		}
	}
}

// ParseStrategy 解析策略
func ParseStrategy(u model.User) *grid.Args {
	var data = map[string]interface{}{}
	var arg grid.Args
	_ = json.Unmarshal([]byte(u.Strategy), &data)
	if u.BasePrice > 0 {
		arg.Price = u.BasePrice
	} else {
		arg.Price, _ = grid.GetPrice(u.Name).Float64()
		u.BasePrice = arg.Price
		u.Update()
	}
	arg.FirstBuy = model.ParseStringFloat(data["FirstBuy"].(string))
	arg.Rate = model.ParseStringFloat(data["rate"].(string))
	arg.MinBuy = arg.FirstBuy
	arg.AddRate = model.ParseStringFloat(data["growth"].(string))
	arg.Callback = model.ParseStringFloat(data["callback"].(string))
	arg.Reduce = model.ParseStringFloat(data["reduce"].(string))
	arg.Stop = model.ParseStringFloat(data["stop"].(string))
	arg.AddMoney = model.ParseStringFloat(data["add"].(string))
	arg.StrategyType = int64(data["Strategy"].(float64))
	if arg.StrategyType == 1 {
		arg.IsChange = false
	}
	if data["frequency"].(float64) == 2 {
		arg.Crile = true
	}
	arg.Decline = 1.5 // 暂设跌幅
	return &arg
}

// LoadStrategy 从表中加载策略
func LoadStrategy(u model.User) (*[]hs.Grid, bool) {
	var data []hs.Grid
	_ = json.Unmarshal([]byte(u.Grids), &data)
	if float64(len(data)) == u.Number {
		return &data, true
	}
	return &data, false
}
