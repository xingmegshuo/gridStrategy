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
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
)

var (
	HUOBI_MIN = decimal.NewFromInt(5)
	BIAN_MIN  = decimal.NewFromFloat(10)
)

// NewSymbol 生成grid.SymbolCategory
func NewSymbol(u User) *SymbolCategory {
	var (
		cateSymol CategorySymbols
	)
	name := ParseSymbol(u.Name)
	if u.Category == "火币" {
		cateSymol = (*HuobiSymbol)[name]
	}
	if u.Category == "币安" {
		if u.Future == 0 {
			cateSymol = (*BianSymbol)[name]
		}
		if u.Future == 1 || u.Future == 3 {
			cateSymol = (*BianFutureU)[name]
		}
		if u.Future == 2 || u.Future == 4 {
			cateSymol = (*BianFutureB)[name]
		}
	}
	if u.Category == "OKex" {
		if u.Future == 0 {
			cateSymol = (*OkexSymbol)[name]
		} else {
			cateSymol = (*OkexFuture)[name]
		}
	}

	symbol := SymbolCategory{
		Category:        u.Category,
		Pashare:         u.Pashare,
		Symbol:          u.Name,
		AmountPrecision: cateSymol.AmountPrecision,
		PricePrecision:  cateSymol.PricePrecision,
		Key:             u.ApiKey,
		Secret:          u.Secret,
		Host:            "",
		MinTotal:        cateSymol.MinTotal,
		MinAmount:       cateSymol.MinAmount,
		BaseCurrency:    cateSymol.BaseCurrency,
		QuoteCurrency:   cateSymol.QuoteCurrency,
		Label:           "goex-" + u.Category + "-" + u.Name + "-" + strconv.Itoa(int(u.ObjectId)),
	}

	if u.Future != 0 {
		symbol.Future = true
	}

	if symbol.Category == "OKex" && u.Future != 0 {
		symbol.MinTotal = decimal.NewFromFloat(cateSymol.CtVal)
	}
	log.Printf("交易对数据:%+v", symbol)
	return &symbol
}

// MakeStrategy 网格模型
func MakeStrategy(u User) (*[]Grid, error) {
	// scale := decimal.NewFromFloat(math.Pow(arg.MinPrice/arg.Price, 1.0/float64(number)))
	arg := StringArg(u.Arg)
	symbol := StringSymobol(u.Symbol)
	currentPrice := decimal.NewFromFloat(arg.Price).Round(symbol.PricePrecision) // 当前价格
	var (
		preTotal decimal.Decimal
		totalBuy decimal.Decimal
	)
	if u.Future == 2 || u.Future == 4 {
		preTotal = decimal.NewFromFloat(arg.FirstBuy)
		totalBuy = preTotal
	} else {
		preTotal = decimal.NewFromFloat(arg.FirstBuy).Div(currentPrice).Round(symbol.AmountPrecision)
		totalBuy = decimal.NewFromFloat(arg.FirstBuy).Round(symbol.PricePrecision + symbol.AmountPrecision)
	}
	if CheckMinToal(u.Category, decimal.NewFromFloat(arg.FirstBuy)) != nil {
		if u.Future == 0 {
			if decimal.NewFromFloat(arg.FirstBuy).Cmp(symbol.MinTotal) < 0 {
				log.Printf("total %s less than minTotal(%s)", decimal.NewFromFloat(arg.FirstBuy), symbol.MinTotal)
				return nil, errors.New(" total not enough")
			}
		}
	}

	var grids []Grid
	currentGrid := Grid{
		Id:        1,
		Price:     currentPrice,
		Decline:   arg.Decline,
		AmountBuy: preTotal,
		TotalBuy:  totalBuy,
	}
	if symbol.Category == "OKex" && u.Future == 1 {
		if symbol.MinTotal.Cmp(currentGrid.AmountBuy) != -1 {
			currentGrid.AmountBuy = symbol.MinAmount.Mul(symbol.MinTotal)
			currentGrid.Mesure = symbol.MinAmount
		} else {
			currentGrid.AmountBuy = currentGrid.AmountBuy.Mul(symbol.MinAmount.Div(symbol.MinTotal)).Mul(symbol.MinTotal).Round(symbol.AmountPrecision)
			currentGrid.Mesure = currentGrid.AmountBuy.Mul(symbol.MinAmount.Div(symbol.MinTotal)).Round(symbol.AmountPrecision)
		}
	} else if u.Future == 2 {
		currentGrid.Mesure = currentGrid.AmountBuy
	}

	grids = append(grids, currentGrid)
	// 补仓
	for i := 2; i <= int(u.Number); i++ {
		// 补仓比例 10
		decline := grids[i-2].Decline + arg.AddRate*0.01*grids[i-2].Decline // 比例跌幅
		if arg.IsAdd {
			decline = grids[i-2].Decline + arg.AddRate // 固定跌幅
		}
		price := grids[i-2].Price.Sub(currentPrice.Mul(decimal.NewFromFloat(decline * 0.01)))
		var TotalBuy decimal.Decimal
		rate := arg.Rate*0.01 + 1 // 补仓比例
		if arg.IsChange {
			// 固定金额
			TotalBuy = grids[i-2].TotalBuy.Add(decimal.NewFromFloat(arg.AddMoney))
		} else {
			TotalBuy = grids[i-2].TotalBuy.Mul(decimal.NewFromFloat(rate))
		}
		currentGrid = Grid{
			Id:        i,
			Price:     price,
			Decline:   decline,
			AmountBuy: TotalBuy.Div(price).Round(symbol.AmountPrecision),
			TotalBuy:  TotalBuy.Round(symbol.PricePrecision + symbol.AmountPrecision),
		}
		if u.Future == 2 || u.Future == 4 {
			if arg.IsChange {
				currentGrid.AmountBuy = grids[i-2].AmountBuy.Add(decimal.NewFromFloat(arg.AddMoney)).Round(0)
			} else {
				currentGrid.AmountBuy = grids[i-2].AmountBuy.Mul(decimal.NewFromFloat(rate)).Round(0)
			}
			currentGrid.TotalBuy = currentGrid.AmountBuy
		}
		if symbol.Category == "OKex" && u.Future == 1 {
			if symbol.MinTotal.Cmp(currentGrid.AmountBuy) != 1 {
				currentGrid.AmountBuy = symbol.MinAmount.Mul(symbol.MinTotal)
				currentGrid.Mesure = symbol.MinAmount
			} else {
				currentGrid.AmountBuy = currentGrid.AmountBuy.Mul(symbol.MinAmount.Div(symbol.MinTotal)).Mul(symbol.MinTotal).Round(symbol.AmountPrecision)
				currentGrid.Mesure = currentGrid.AmountBuy.Mul(symbol.MinAmount.Div(symbol.MinTotal)).Round(symbol.AmountPrecision)
			}
		} else if u.Future == 2 {
			currentGrid.Mesure = currentGrid.AmountBuy
		}
		grids = append(grids, currentGrid)
	}

	if arg.FirstDouble {
		grids[0].TotalBuy = grids[0].TotalBuy.Add(grids[0].TotalBuy).Round(symbol.PricePrecision + symbol.AmountPrecision)
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
	// log.Println("arg 解析")

	var data = map[string]interface{}{}
	var arg Args
	_ = json.Unmarshal([]byte(u.Strategy), &data)
	if data["coin_id"] != nil {
		arg.Price, _ = GetPrice(ParseFloatString(data["coin_id"].(float64))).Float64()
	}
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
	arg.Level = data["leverage"]
	arg.CoinId = data["coin_id"]
	// log.Println("用户修改:", u.ObjectId, u.IsRun)
	if arg.StrategyType == 1 {
		arg.IsChange = false
	}
	arg.StopEnd = ParseStringFloat(data["end"].(string))
	arg.Crile = data["frequency"].(float64)

	arg.Decline = ParseStringFloat(data["decline"].(string)) // 暂设跌幅
	if data["allSell"].(float64) == 2 {
		arg.AllSell = true
	}
	if data["allSell"].(float64) == 3 {
		// arg.AllSell = true
		arg.StopFlow = true
	}
	if data["one_buy"].(float64) == 2 {
		arg.OneBuy = true
	}
	if data["add_type"] != nil && data["add_type"].(float64) == 1 {
		arg.IsAdd = true
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
	if data["stop_buy"].(float64) == 2 {
		arg.StopBuy = true
	} else {
		arg.StopBuy = false
	}
	if data["order_type"].(float64) == 2 {
		arg.IsHand = true
	}
	if arg.StrategyType == 3 || arg.StrategyType == 4 {
		arg.OrderType = 2
	}
	// log.Printf("arg数据:%+v", arg)
	arg.Callback = 0
	arg.Reduce = 0
	return &arg
}

// ListenU 监听变化的字段
func ListenU(u User, arg *Args) *Args {
	var data = map[string]interface{}{}
	_ = json.Unmarshal([]byte(u.Strategy), &data)
	if data["allSell"].(float64) == 2 {
		arg.AllSell = true
	}
	if data["allSell"].(float64) == 3 {
		// arg.AllSell = true
		arg.StopFlow = true
	}
	if data["one_buy"].(float64) == 2 {
		arg.OneBuy = true
	}

	if data["stop_buy"].(float64) == 2 {
		arg.StopBuy = true
	} else {
		arg.StopBuy = false
	}
	if data["Crile"] != nil {
		arg.Crile = data["Crile"].(float64)
	}
	if data["limit_high"].(float64) == 2 {
		arg.IsLimit = true
	} else {
		arg.IsLimit = false
	}
	return arg
}

// GetPrice 从缓存获取价格
func GetPrice(coin string) decimal.Decimal {
	v, _ := decimal.NewFromString(CachePrice(coin))
	return v
}

// ToStringJson 转换为字符串类型
func ToStringJson(v interface{}) string {
	s, _ := json.Marshal(v)
	return string(s)
}

// CheckMinToal 比较是否下单小于最小下单
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

// StringArg 字符串转arg
func StringArg(data string) (a Args) {
	_ = json.Unmarshal([]byte(data), &a)
	return
}

// ArgString arg 转字符串
func ArgString(a *Args) string {
	s, _ := json.Marshal(&a)
	return string(s)
}

// StringSymobol 字符串转symbol
func StringSymobol(data string) (a SymbolCategory) {
	_ = json.Unmarshal([]byte(data), &a)
	return
}

// 获取交易对参数
func GetSymbols(name string) *map[string]CategorySymbols {
	var data = []map[string]interface{}{}
	switch name {
	case "火币":
		data = LoadJson("huobiSpot.json")
	case "币安":
		data = LoadJson("bianSpot.json")
	case "币安u":
		data = LoadJson("bianFutureU.json")
	case "币安b":
		data = LoadJson("bianFutureB.json")
	case "ok":
		data = LoadJson("okSpot.json")
	case "okSwap":
		data = LoadJson("okSwap.json")
	}
	return ParseMapCategorySymobls(data, name)
}

// 从json文件中加载交易对参数
func LoadJson(name string) (res []map[string]interface{}) {
	var data = map[string][]map[string]interface{}{}
	if dir, err := os.Getwd(); err == nil {
		bytes, _ := ioutil.ReadFile(dir + "/symbolInfo/" + name)
		err = json.Unmarshal(bytes, &data)
	}
	res = data["data"]
	return
}

/*

	 @title   : ParseMapCategorySymobls
	 @desc    : 解析交易对参数获取下单要求
	 @auth    : small_ant / time(2021/08/14 09:36:19)
	 @param   : v,name / []map[string]interface{} string / `从json中加载的数据,平台名称`
	 @return  : res / *[]map[string]CategorySymbols / `交易对切片指针`
**/
func ParseMapCategorySymobls(v []map[string]interface{}, name string) *map[string]CategorySymbols {
	var (
		c   CategorySymbols
		res = map[string]CategorySymbols{}
	)
	if name == "火币" {
		for _, data := range v {
			c.PricePrecision = int32(data["price-precision"].(float64))
			c.AmountPrecision = int32(data["amount-precision"].(float64))
			c.MinAmount = decimal.NewFromFloat(data["min-order-amt"].(float64))
			c.MinTotal = decimal.NewFromFloat(data["min-order-value"].(float64))
			c.BaseCurrency = data["base-currency"].(string)
			c.QuoteCurrency = data["quote-currency"].(string)
			res[strings.ToUpper(data["symbol"].(string))] = c
		}
	}
	if name == "币安" {
		for _, data := range v {
			filter := data["filters"].([]interface{})
			for _, v := range filter {
				d := v.(map[string]interface{})
				if d["filterType"].(string) == "LOT_SIZE" {
					c.MinAmount, _ = decimal.NewFromString(d["minQty"].(string))
					c.AmountPrecision = int32(floatLen(d["minQty"].(string)))
				}
				if d["filterType"] != nil && d["filterType"].(string) == "MIN_NOTIONAL" {
					c.MinTotal, _ = decimal.NewFromString(d["minNotional"].(string))
					c.PricePrecision = int32(8)
				}
			}

			// c.MinAmount = decimal.NewFromFloat(0)
			// c.MinTotal = decimal.NewFromFloat(10)
			c.BaseCurrency = data["baseAsset"].(string)
			c.QuoteCurrency = data["quoteAsset"].(string)
			res[data["symbol"].(string)] = c
		}
	}
	if name == "币安b" || name == "币安u" {
		for _, data := range v {
			filter := data["filters"].([]interface{})
			for _, v := range filter {
				// if v["filterType"].(string) == "PRICE_FILTER" {
				// 	v["minPrice"].(float64)
				// }
				d := v.(map[string]interface{})
				if d["filterType"].(string) == "LOT_SIZE" {
					c.MinAmount, _ = decimal.NewFromString(d["minQty"].(string))
				}
				if d["filterType"] != nil && d["filterType"].(string) == "MIN_NOTIONAL" {
					c.MinTotal, _ = decimal.NewFromString(d["notional"].(string))
				}
			}
			c.PricePrecision = int32(data["pricePrecision"].(float64))
			if name == "币安b" {
				c.AmountPrecision = int32(data["equalQtyPrecision"].(float64))
				c.BaseCurrency = data["quoteAsset"].(string)
				c.QuoteCurrency = data["baseAsset"].(string)
			}
			if name == "币安u" {
				c.AmountPrecision = int32(data["quantityPrecision"].(float64))
				c.BaseCurrency = data["baseAsset"].(string)
				c.QuoteCurrency = data["quoteAsset"].(string)
			}
			// c.MinAmount = decimal.NewFromFloat(0)
			// c.MinTotal = decimal.NewFromFloat(10)

			res[data["symbol"].(string)] = c
		}
	}

	if name == "ok" {
		for _, data := range v {
			c.PricePrecision = int32(floatLen(data["tickSz"].(string)))
			c.AmountPrecision = int32(floatLen(data["lotSz"].(string)))
			c.MinAmount, _ = decimal.NewFromString(data["minSz"].(string))
			// c.MinTotal = decimal.NewFromFloat(data["min-order-value"].(float64))
			c.BaseCurrency = data["baseCcy"].(string)
			c.QuoteCurrency = data["quoteCcy"].(string)
			res[strings.ToUpper(strings.Replace(data["instId"].(string), "-", "", 1))] = c
		}
	}

	if name == "okSwap" {
		for _, data := range v {
			c.PricePrecision = int32(floatLen(data["tickSz"].(string)))
			c.AmountPrecision = int32(floatLen(data["lotSz"].(string)))
			c.MinAmount, _ = decimal.NewFromString(data["minSz"].(string))
			// c.MinTotal = decimal.NewFromFloat(data["min-order-value"].(float64))
			c.BaseCurrency = data["ctValCcy"].(string)
			c.QuoteCurrency = data["settleCcy"].(string)
			c.CtVal = ParseStringFloat(data["ctVal"].(string))
			res[strings.ToUpper(
				strings.Replace(strings.Replace(data["instId"].(string), "-", "", 1), "-SWAP", "", 1))] = c
		}
	}

	return &res
}

// CachePrice 从缓存中读取价格
func CachePrice(b string) string {
	var (
		res  = map[string]string{}
		data []string
	)
	op := &redis.ZRangeBy{
		Min: b,
		Max: b,
	}
	for {
		data, _ = ListCacheGet("ZMYCOINS", op).Result()
		if len(data) > 0 {
			break
		} else {
			time.Sleep(time.Millisecond * 300)
		}
	}
	json.Unmarshal([]byte(data[0]), &res)
	return res["price_usd"]
}

// floatLen 计算精度
func floatLen(d string) int {
	tmp := strings.Split(d, ".")
	if len(tmp) <= 1 {
		return 0
	}
	for i := len(tmp[1]); i > 0; i-- {
		if tmp[1][i-1] != '0' {
			return i
		}
	}
	return 0
}

// UpdateBase 更新持仓单数为0
func UpdateBase(objectId interface{}) {
	DB.Exec("update users set base=0 where object_id = ?", objectId)
}

// UpdateLimit 限价单执行后关闭
func UpdateLimit(objectId interface{}) {
	UserDB.Exec("update db_task_order set limit_high = 1 where id = ?", objectId)
}
