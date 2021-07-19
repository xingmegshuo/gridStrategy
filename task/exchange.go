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
	"encoding/json"
	"errors"
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
	for _, u := range *users {
		start := 0
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
				if u.Status == float64(1) && u.IsRun == int64(-1) && Symbols != nil && start == 0 {
					for i := 1; i < 2; i++ {
						symbol, arg, straggly := PrintStrategy(u, Symbols)
						if straggly != nil {
							start = 1
							// runtime.GOMAXPROCS(1)
							log.Println("协程开始----------------用户:", u.ID, "---交易币种:", u.Name, symbol, arg)
							u.IsRun = 10
							u.Update()
							// go RunStrategy(&straggly, u, &symbol, arg)
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

	strategy, e := MakeStrategy(arg, int(u.Number), int32(amountPrecision), int32(pricePrecision), NminAmount, NminToal)

	if e != nil {
		u.Error = e.Error()
		u.Status = float64(0)
		u.IsRun = int64(-2)
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
		// 生成策略后开始跑
		if s, ok := LoadStrategy(u); ok {
			strategy = *s
		}
		return symbolData, arg, strategy
	}
}

// MakeStrategy 网格模型
func MakeStrategy(arg *grid.Args, number int, amountPrecision int32,
	pricePrecision int32, minAmount decimal.Decimal, minTotal decimal.Decimal) ([]hs.Grid, error) {
	// scale := decimal.NewFromFloat(math.Pow(arg.MinPrice/arg.Price, 1.0/float64(number)))
	preTotal := decimal.NewFromFloat(arg.FirstBuy).Div(decimal.NewFromFloat(arg.Price)).Round(amountPrecision)
	currentPrice := decimal.NewFromFloat(arg.Price)
	NowPrice := currentPrice
	var grids []hs.Grid
	// 第一次买入
	currentGrid := hs.Grid{
		Id:        1,
		Price:     currentPrice.Round(pricePrecision),
		AmountBuy: preTotal,
		TotalBuy:  decimal.NewFromFloat(arg.FirstBuy),
	}
	arg.NeedMoney += arg.FirstBuy
	grids = append(grids, currentGrid)
	// 补仓
	for i := 2; i <= number; i++ {
		// 补仓比例 10
		rate := arg.Rate + (arg.AddRate * float64(i-2))
		priceRate := 0.02 + (arg.AddRate * 0.01 * float64(i-2))                                                           // 跌幅 下降2个点买入                                                                                                                   // 跌幅
		currentPrice = decimal.NewFromFloat(float64(1) - priceRate).Mul(NowPrice)                                         // 当前价格
		amountBuy := decimal.NewFromFloat(arg.FirstBuy + arg.FirstBuy*rate*0.01).Div(currentPrice).Round(amountPrecision) // 当前买入
		realTotal := currentPrice.Mul(amountBuy)                                                                          // 当前买入
		if realTotal.Cmp(minTotal) == -1 {
			log.Printf("total %s less than minTotal(%s)", realTotal, minTotal)
			return nil, errors.New(" total not enough")
		}
		arg.NeedMoney += arg.FirstBuy + arg.FirstBuy*rate*0.01
		currentGrid = hs.Grid{
			Id:        i,
			Price:     currentPrice,
			AmountBuy: amountBuy,
			TotalBuy:  realTotal,
		}
		// log.Println(currentGrid)
		grids = append(grids, currentGrid)
	}
	// 打印网格模型
	log.Printf("Id\tTotal\tPrice\tAmountBuy\tAmountSell")
	for _, g := range grids {
		log.Printf("%2d\t%s\t%s\t%s\t%s",
			g.Id, g.TotalBuy.StringFixed(amountPrecision+pricePrecision),
			g.Price.StringFixed(pricePrecision),
			g.AmountBuy.StringFixed(amountPrecision), g.AmountSell.StringFixed(amountPrecision))

	}

	return grids, nil
}

// ParseStringFloat 字符串转浮点型
func ParseStringFloat(str string) float64 {
	floatValue, _ := strconv.ParseFloat(str, 64)
	return floatValue
}

// RunStrategy 开启策略
func RunStrategy(s *[]hs.Grid, u model.User, symbol *grid.SymbolCategory, arg *grid.Args) {
OuterLoop:
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		select {
		case <-grid.GridDone:
			log.Println("收到消息,暂停策略,exiting ......", u.ID)
			break OuterLoop
		default:
		}
		// 执行任务不是一次执行
		if u.IsRun == -1 {
			u.IsRun = 10
			u.Update()
			for i := 0; i < 1; i++ {
				log.Println("协程2开始---------")
				go grid.Run(ctx, s, u, symbol, arg)
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
	arg.FirstBuy = ParseStringFloat(data["FirstBuy"].(string))
	arg.Rate = ParseStringFloat(data["rate"].(string))
	arg.MinBuy = arg.FirstBuy
	arg.AddRate = ParseStringFloat(data["growth"].(string))
	arg.Callback = ParseStringFloat(data["callback"].(string))
	arg.Reduce = ParseStringFloat(data["reduce"].(string))
	arg.Stop = ParseStringFloat(data["stop"].(string))
	arg.AddMoney = ParseStringFloat(data["add"].(string))
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

// SaveStrategy 保存
func SaveStrategy(u model.User, g *[]hs.Grid) {
	s, _ := json.Marshal(&g)
	u.Grids = string(s)
	u.Update()
}
