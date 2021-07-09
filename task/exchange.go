/***************************
@File        : exchange.go.go
@Time        : 2021/7/2 15:08
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 生成策略并开启
****************************/

package job

import (
	"context"
	"errors"

	"math"
	"strconv"
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
	//log.Println("i am working for make strategy and start run ")
	users := LoadUsers()
	for _, u := range users {
		Symbols := LoadSymbols(u.Category)
	OuterLoop:
		for {
			select {
			case v := <-model.Ch:
				u.Status = float64(v.Run)
				u.Update()
				// 停止策略,协程退出
				if u.IsRun == 10 {
					log.Println("向协程1发送退出")
					u.IsRun = 2
					u.Update()
					grid.GridDone <- 1
					break OuterLoop
				}
				//if u.IsRun == -2 {
				//	//Done <- 1
				//	u.IsRun = -10
				//	log.Println("策略运行出错，正在退出...........", u.IsRun)
				//	u.Update()
				//	grid.GridDone <- 1
				//	break OuterLoop
				//}
			default:
				if u.Status == float64(1) && u.IsRun == int64(-1) && Symbols != nil {
					for i := 1; i < 2; i++ {
						symbol, straggly := PrintStrategy(u, Symbols)
						if straggly != nil {
							log.Println("协程开始----------------用户:", u.ID, "---交易币种:", u.Name)
							go RunStrategy(&straggly, u, &symbol)
						} else {
							u.IsRun = -2
							u.Error = "参数配置错误，不能生成策略"
							u.Update()
						}
					}
				}
				break OuterLoop
			}
		}
	}
}

// LoadUsers 获取数据库用户内容
func LoadUsers() []model.User {
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
func PrintStrategy(u model.User, symbol []map[string]interface{}) (grid.SymbolCategory, []hs.Grid) {
	var (
		pricePrecision  float64
		amountPrecision float64
		minAmount       float64
		minToal         float64
		baseCurrency    string
		quoteCurrency   string
	)
	for _, v := range symbol {
		if v["symbol"] == "shibusdt" {
			pricePrecision = v["price-precision"].(float64)
			amountPrecision = v["amount-precision"].(float64)
			minAmount = v["min-order-amt"].(float64)
			minToal = v["min-order-value"].(float64)
			baseCurrency = v["base-currency"].(string)
			quoteCurrency = v["quote-currency"].(string)

		}
	}
	NminAmount := decimal.NewFromFloat(minAmount)
	NminToal := decimal.NewFromFloat(minToal)
	strategy, e := MakeStrategy(ParseStringFloat(u.MaxPrice), ParseStringFloat(u.MinPrice), int(u.Number), ParseStringFloat(u.Total), int32(amountPrecision), int32(pricePrecision), NminAmount, NminToal)

	if e != nil {
		u.Error = e.Error()
		u.Status = float64(0)
		u.IsRun = int64(-2)
		u.Update()
		return grid.SymbolCategory{}, nil
	} else {
		h := model.Host{}
		h.Get(u.Category)
		symbolData := grid.SymbolCategory{
			Category:        u.Category,
			Symbol:          "shibusdt",
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
		// 生成策略后开始跑
		return symbolData, strategy
	}
}

// MakeStrategy 网格模型
func MakeStrategy(maxPrice float64, minPrice float64, number int, total float64, amountPrecision int32,
	pricePrecision int32, minAmount decimal.Decimal, minTotal decimal.Decimal) ([]hs.Grid, error) {
	scale := decimal.NewFromFloat(math.Pow(minPrice/maxPrice, 1.0/float64(number)))
	preTotal := decimal.NewFromFloat(total / float64(number))
	currentPrice := decimal.NewFromFloat(maxPrice)
	var grids []hs.Grid
	currentGrid := hs.Grid{
		Id:    0,
		Price: currentPrice.Round(pricePrecision),
	}
	grids = append(grids, currentGrid)

	for i := 1; i <= number; i++ {
		// log.Info(t.pricePrecision, t.amountPrecision, t.scale, currentPrice)
		currentPrice = currentPrice.Mul(scale).Round(pricePrecision)
		amountBuy := preTotal.Div(currentPrice).Round(amountPrecision)
		if amountBuy.Cmp(minAmount) == -1 {
			log.Printf("amount %s less than minAmount(%s)", amountBuy, minAmount)
			return nil, errors.New("order amount not enough")
		}
		realTotal := currentPrice.Mul(amountBuy)
		if realTotal.Cmp(minTotal) == -1 {
			log.Printf("total %s less than minTotal(%s)", realTotal, minTotal)
			return nil, errors.New(" total not enough")
		}
		currentGrid = hs.Grid{
			Id:        i,
			Price:     currentPrice,
			AmountBuy: amountBuy,
			TotalBuy:  realTotal,
		}
		grids = append(grids, currentGrid)
		grids[i-1].AmountSell = amountBuy
	}
	// 打印网格模型
	//log.Printf("Id\tTotal\tPrice\tAmountBuy\tAmountSell")
	//
	//for _, g := range grids {
	//	log.Printf("%2d\t%s\t%s\t%s\t%s",
	//		g.Id, g.TotalBuy.StringFixed(amountPrecision+pricePrecision),
	//		g.Price.StringFixed(pricePrecision),
	//		g.AmountBuy.StringFixed(amountPrecision), g.AmountSell.StringFixed(amountPrecision))
	//}

	return grids, nil
}

// ParseStringFloat 字符串转浮点型
func ParseStringFloat(str string) float64 {
	floatValue, _ := strconv.ParseFloat(str, 64)
	return floatValue
}

// RunStrategy 开启策略
func RunStrategy(s *[]hs.Grid, u model.User, symbol *grid.SymbolCategory) {
OuterLoop:
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		//grid.Run(s,symbol)
		select {
		case <-grid.GridDone:
			log.Println("收到消息,暂停策略,exiting ......", u.ID)
			//runtime.Goexit()
			break OuterLoop
		default:
		}
		// 执行任务不是一次执行
		if u.IsRun == -1 {
			u.IsRun = 10
			u.Update()
			for i := 0; i < 1; i++ {
				log.Println("协程2开始---------")
				go grid.Run(ctx, s, u, symbol)
			}
		}
	}
}
