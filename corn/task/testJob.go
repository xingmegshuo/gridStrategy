/***************************
@File        : testJob.go
@Time        : 2021/7/1 18:09
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package job

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime"
	"sync"
	"time"
	grid "zmyjobs/corn/grid"
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/uti"
	"zmyjobs/goex"

	"github.com/go-redis/redis/v8"
	"github.com/shopspring/decimal"
)

//var job = model.NewJob(model.ConfigMap["jobType1"],"test","@every 5s")
var (
	crawJob  = model.NewJob(model.ConfigMap["jobType1"], "爬取基础数据", "@every 3s")
	crawLock sync.Mutex
)

func InitJob(j model.Job, f func()) {
	err := C.AddFunc(j.Spec, f)
	if err == nil {
		j.Status = model.ConfigMap["jobStatus1"]
	}
	j.UpdateJob()
	Wg.Add(1)
}

func JobExit(job model.Job) {
	job.Status = model.ConfigMap["jobStatus2"]
	job.UpdateJob()
	Wg.Done()
}

func CrawRun() {
	start := time.Now()
	// fmt.Println("开始jjjjj")
	coinCache := []*redis.Z{}
	go craw(coinCache)
	go xhttp("https://dapi.binance.com/dapi/v1/ticker/24hr", "ZMYCOINF")
	go xhttp("https://fapi.binance.com/fapi/v1/ticker/24hr", "ZMYUSDF")
	go crawAccount()
	// fmt.Println("结束")
	if time.Since(start) > time.Second*10 {
		fmt.Println("超时退出", time.Since(start))
		runtime.Goexit()
	}
}

// xhttp 缓存信息
func xhttp(url string, name string) {
	if !model.CheckCache(name) {
		client := makeClient()
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			content, _ := ioutil.ReadAll(resp.Body)
			var realData []map[string]interface{}
			_ = json.Unmarshal(content, &realData)
			var coins []model.Coin
			for _, one := range realData {
				// fmt.Printf("%+v", one)
				var coin model.Coin
				coin.CategoryId = 2
				coin.Name = util.ToMySymbol(one["symbol"].(string))
				coin.PriceUsd = model.ParseStringFloat(one["lastPrice"].(string))
				coin.Price = coin.PriceUsd * 6.5
				if one["quoteVolume"] == nil {
					coin.DayAmount = model.ParseStringFloat(one["volume"].(string)) * 6.5 / 100000000
				} else {
					coin.DayAmount = model.ParseStringFloat(one["quoteVolume"].(string)) * 6.5 / 100000000
				}
				coin.Raf = model.ParseStringFloat(one["priceChangePercent"].(string))
				coin.CoinType = util.SwitchCoinType(coin.Name)
				coin.EnName = coin.Name
				coin.CoinName = coin.Name
				coin.CreateTime = int(time.Now().Unix())
				coins = append(coins, coin)
			}
			data, _ := json.Marshal(&coins)
			model.Del(name)
			// fmt.Println("写入数据", name)
			model.SetCache(name, string(data), time.Hour)
		}
	}
}

// craw
func craw(coinCache []*redis.Z) {
	// start := time.Now()
	model.UserDB.Raw("select count(*) from db_task_coin").Scan(&coinCount)
	coinCache = append(coinCache, xhttpCraw("https://api.huobi.pro/market/tickers", 1, 0)...)
	coinCache = append(coinCache, xhttpCraw("https://api.binance.com/api/v3/ticker/24hr", 2, 0)...)
	coinCache = append(coinCache, xhttpCraw("https://www.okex.com/api/spot/v3/instruments/ticker", 5, 0)...)
	coinCache = append(coinCache, xhttpCraw("https://dpi.binance.com/dapi/v1/ticker/24hr", 2, 2)...)
	coinCache = append(coinCache, xhttpCraw("https://fpi.binance.com/fapi/v1/ticker/24hr", 2, 2)...)
	// fmt.Println(len(coinCache), coinCount, time.Since(start))
	if len(coinCache) == coinCount {
		fmt.Println("write coins db")
		model.Del("ZMYCOINS")
		model.AddCache("ZMYCOINS", coinCache...)
		coinCache = []*redis.Z{}
	}
}

// xhttpCraw 不缓存只更新数据   抓取最新的币种价格行情
func xhttpCraw(url string, category int, coinType int) []*redis.Z {
	client := makeClient()
	resp, err := client.Get(url)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var data = make(map[string]interface{})
		var realData []map[string]interface{}
		if category == 1 {
			_ = json.Unmarshal(content, &data)
			byteData, _ := json.Marshal(data["data"])
			_ = json.Unmarshal(byteData, &realData)
		}
		if category == 2 || category == 5 {
			_ = json.Unmarshal(content, &realData)
		}
		return WriteDB(realData, category, coinType)
	} else {
		fmt.Println(err)
		return []*redis.Z{}
	}
}

// WriteDB 解析平台返回数据
func WriteDB(realData []map[string]interface{}, category int, coinType int) (coinCache []*redis.Z) {
	for _, s := range realData {
		var (
			symbol string
		)
		if category == 5 {
			symbol = s["instrument_id"].(string)
		} else {
			symbol = s["symbol"].(string)
		}
		if name := util.ToMySymbol(symbol); name != "none" {
			var id interface{}
			if coinType == 0 {
				model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ? and category_id = ?", 0, name, category).Scan(&id)
			} else {
				model.UserDB.Raw("select id from db_task_coin where coin_type != ? and name = ? and category_id = ?", 0, name, category).Scan(&id)
			}
			if id != nil {
				var (
					raf       float64
					dayAmount string
					price     float64
				)
				if category == 1 {
					price = s["close"].(float64)
					raf = (price - s["open"].(float64)) / s["open"].(float64) * 100
					dayAmount = fmt.Sprintf("%.2f", s["amount"].(float64)*price*float64(6.5)/100000000)
				}
				if category == 2 {
					price = model.ParseStringFloat(s["lastPrice"].(string))
					raf = (price - model.ParseStringFloat(s["openPrice"].(string))) / model.ParseStringFloat(s["openPrice"].(string)) * 100
					dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["volume"].(string))*price*float64(6.5)/100000000)
				}
				if category == 5 {
					price = model.ParseStringFloat(s["last"].(string))
					raf = (price - model.ParseStringFloat(s["open_utc8"].(string))) / model.ParseStringFloat(s["open_utc8"].(string)) * 100
					dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["quote_volume_24h"].(string))*6.5/100000000)
				}
				r := fmt.Sprintf("%.2f", raf) // 涨跌幅
				value := map[string]interface{}{
					"price_usd":  fmt.Sprintf("%f", price),
					"price":      fmt.Sprintf("%.4f", price*6.5),
					"day_amount": dayAmount,
					"raf":        r,
				}
				s, _ := json.Marshal(&value)
				data := &redis.Z{
					Score:  float64(id.(int64)),
					Member: s,
				}
				coinCache = append(coinCache, data)
			}
		}
	}
	return
}

func makeClient() http.Client {
	return http.Client{Timeout: 3 * time.Second}
	// util.ProxyHttp()
	// return *util.ProxyHttp()
}

// CrawAccount 缓存用户持仓数据
func crawAccount() {
	var (
		users = []*redis.Z{}
		ids   []float64
	)
	model.UserDB.Raw("select id from db_customer").Scan(&ids)
	// fmt.Println(ids)
	for _, id := range ids {
		data := map[string]interface{}{
			"1": map[string][]map[string]interface{}{
				"spot": GetUserHold(id, 1, 0),
			},
			"2": map[string][]map[string]interface{}{
				"spot": GetUserHold(id, 2, 0),
				"B":    GetUserHold(id, 2, 1),
				"U":    GetUserHold(id, 2, 2),
			},
		}
		str, _ := json.Marshal(&data)
		// fmt.Println(string(str), id)
		users = append(users, &redis.Z{
			Score:  id,
			Member: string(str),
		})
	}
	if len(users) == len(ids) {
		model.Del("ZMYUSERS")
		model.AddCache("ZMYUSERS", users...)
	}
}

func GetUserHold(id float64, cate float64, t float64) (data []map[string]interface{}) {
	b, name, key, secret := model.GetApiConfig(id, cate)
	if b && t == 0 {
		c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8})
		value, err := c.Ex.GetAccount()
		// fmt.Println(err)
		if err == nil {
			for k, v := range value.SubAccounts {
				if v.Amount > 0 {
					one := map[string]interface{}{}
					one["amount"] = decimal.NewFromFloat(v.Amount).Round(8)
					one["symbol"] = k.Symbol
					data = append(data, one)
				}
			}
		}
		return
	} else if t == 1 && b {
		c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Future: true})
		Bdata, Berr := c.Future.GetFuturePosition(goex.UNKNOWN_PAIR, goex.SWAP_CONTRACT)
		if Berr == nil {
			for _, v := range Bdata {
				var one = map[string]interface{}{}
				one["amount"] = v.BuyAmount
				one["symbol"] = v.Symbol.String()
				one["unprofit"] = v.BuyProfitReal
				one["level"] = v.LeverRate
				if v.ContractType == "LNOG" {
					one["slide"] = "做多"
				} else {
					one["slide"] = "做空"
				}
				data = append(data, one)
			}
		}
		return
	} else if t == 2 && b {
		c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Future: true})
		value, err := c.Future.GetFuturePosition(goex.UNKNOWN_PAIR, goex.SWAP_USDT_CONTRACT)
		if err == nil {
			for _, v := range value {
				var one = map[string]interface{}{}
				one["amount"] = v.BuyAmount
				one["symbol"] = v.Symbol.String()
				one["unprofit"] = v.BuyProfitReal
				one["level"] = v.LeverRate
				if v.ContractType == "LNOG" {
					one["slide"] = "做多"
				} else {
					one["slide"] = "做空"
				}
				data = append(data, one)
			}
		}
		return
	}
	return
}
