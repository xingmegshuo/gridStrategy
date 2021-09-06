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
	count    = 0
	port     = "1124"
	readWs   = true
	OpenWs   = false
	startWs  = time.Now()
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
	StopHttp := make(chan int)
	start := time.Now()
	for i := 0; i < 1; i++ {
		go func() {
			// model.UserDB.Raw("select id from db_task_coin").Scan(&coinIds)
			stratHttp := false
			for {
				select {
				case <-StopHttp:
					fmt.Println("退出爬取数据", time.Since(start))
					runtime.Goexit()
				default:
					time.Sleep(time.Millisecond * 200)
					if !stratHttp {
						stratHttp = true
						craw()
						if count%20 == 0 {
							xhttp("https://dapi.binance.com/dapi/v1/ticker/24hr", "ZMYCOINF")
							xhttp("https://fapi.binance.com/fapi/v1/ticker/24hr", "ZMYUSDF")
							crawAccount()
						}
						count++
					}
				}
			}

			// wsCraw(coinCache)
		}()
	}
	for {
		if time.Since(startWs) > time.Hour*8 {
			OpenWs = false
			Stop <- 2
		}
		if time.Since(start) > time.Second*10 {
			fmt.Println("超时退出", time.Since(start))
			StopHttp <- 2

		} else {
			time.Sleep(time.Second)
		}
	}
}

// xhttp 缓存信息
func xhttp(url string, name string) {
	if !model.CheckCache(name) {
		resp, err := util.ProxyHttp("1123").Get(url)
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
func craw() {
	if count%2 == 0 {
		crawHuobiSpot()
		CrawBianSpot()
		CrawOkSpot()
	}
	CrawBianU()
	CrawBianB()
}

func wsCraw() {
	if len(BianSpot) > 0 {
		for k, v := range BianSpot {
			if name := util.ToMySymbol(k); name != "none" {
				var id interface{}
				model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ? and category_id = ?", 0, name, 2).Scan(&id)
				if id != nil {
					price := v.Last
					raf := v.Raf
					dayAmount := fmt.Sprintf("%.2f", v.Vol*price*float64(6.5)/100000000)
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
					model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
					model.ListCacheAddOne("ZMYCOINS", data)
				}
			}
		}
	}
}

func crawHuobiSpot() {
	resp, err := xhttpGet("https://api.huobi.pro/market/tickers", port)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var data = make(map[string]interface{})
		var realData []map[string]interface{}
		_ = json.Unmarshal(content, &data)
		byteData, _ := json.Marshal(data["data"])
		_ = json.Unmarshal(byteData, &realData)
		if len(realData) > 0 {
			for _, s := range realData {
				symbol := s["symbol"].(string)
				var id interface{}
				if name := util.ToMySymbol(symbol); name != "none" && name[len(name)-4:] == "USDT" {
					model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ?  and category_id = ?", 0, name, 1).Scan(&id)
					if id != nil {
						var (
							raf       float64
							dayAmount string
							price     float64
						)
						price = s["close"].(float64)
						raf = (price - s["open"].(float64)) / s["open"].(float64) * 100
						dayAmount = fmt.Sprintf("%.2f", s["amount"].(float64)*price*float64(6.5)/100000000)
						model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
						model.ListCacheAddOne("ZMYCOINS", makePriceInfo(price, dayAmount, raf, name, id))
					}
				}
			}
		} else {
			ChangePort()
		}
	} else {
		ChangePort()
	}
}

func CrawBianSpot() {
	if !readWs {
		resp, err := xhttpGet("https://api.binance.com/api/v3/ticker/24hr", port)
		if err == nil {
			defer resp.Body.Close()
			content, _ := ioutil.ReadAll(resp.Body)
			var realData []map[string]interface{}
			_ = json.Unmarshal(content, &realData)
			if len(realData) > 0 {
				for _, s := range realData {
					symbol := s["symbol"].(string)
					var id interface{}
					if name := util.ToMySymbol(symbol); name != "none" && name[len(name)-4:] == "USDT" {
						model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ?  and category_id = ?", 0, name, 2).Scan(&id)
						if id != nil {
							var (
								raf       float64
								dayAmount string
								price     float64
							)
							price = model.ParseStringFloat(s["lastPrice"].(string))
							raf = (price - model.ParseStringFloat(s["openPrice"].(string))) / model.ParseStringFloat(s["openPrice"].(string)) * 100
							dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["volume"].(string))*price*float64(6.5)/100000000)
							model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
							model.ListCacheAddOne("ZMYCOINS", makePriceInfo(price, dayAmount, raf, name, id))
						}
					}
				}
			} else {
				readWs = true
			}
		} else {
			readWs = true
		}
	} else if !OpenWs {
		go Begin()
		OpenWs = true
	} else if OpenWs {
		wsCraw()
	}
}

func CrawBianU() {
	resp, err := xhttpGet("https://fapi.binance.com/fapi/v1/ticker/24hr", port)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var realData []map[string]interface{}
		_ = json.Unmarshal(content, &realData)
		if len(realData) > 0 {
			for _, s := range realData {
				symbol := s["symbol"].(string)
				var id interface{}
				if name := util.ToMySymbol(symbol); name != "none" && name[len(name)-4:] == "USDT" {
					model.UserDB.Raw("select id from db_task_coin where coin_type not in (0,2,4) and name = ?  and category_id = ?", name, 2).Scan(&id)
					if id != nil {
						var (
							raf       float64
							dayAmount string
							price     float64
						)
						price = model.ParseStringFloat(s["lastPrice"].(string))
						raf = (price - model.ParseStringFloat(s["openPrice"].(string))) / model.ParseStringFloat(s["openPrice"].(string)) * 100
						dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["volume"].(string))*price*float64(6.5)/100000000)
						model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
						model.ListCacheAddOne("ZMYCOINS", makePriceInfo(price, dayAmount, raf, name, id))
					}
				}
			}
		} else {
			ChangePort()
		}
	} else {
		ChangePort()
	}
}

func CrawBianB() {
	resp, err := xhttpGet("https://dapi.binance.com/dapi/v1/ticker/24hr", port)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var realData []map[string]interface{}
		_ = json.Unmarshal(content, &realData)
		if len(realData) > 0 {
			for _, s := range realData {
				symbol := s["symbol"].(string)
				var id interface{}
				if name := util.ToMySymbol(symbol); name != "none" {
					model.UserDB.Raw("select id from db_task_coin where coin_type not in (0,1,3) and name = ?  and category_id = ?", name, 2).Scan(&id)
					if id != nil {
						var (
							raf       float64
							dayAmount string
							price     float64
						)
						price = model.ParseStringFloat(s["lastPrice"].(string))
						raf = (price - model.ParseStringFloat(s["openPrice"].(string))) / model.ParseStringFloat(s["openPrice"].(string)) * 100
						dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["volume"].(string))*price*float64(6.5)/100000000)
						model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
						model.ListCacheAddOne("ZMYCOINS", makePriceInfo(price, dayAmount, raf, name, id))
					}
				}
			}
		} else {
			ChangePort()
		}
	} else {
		ChangePort()
	}
}

func ChangePort() {
	if port == "1123" {
		port = "1124"
	} else {
		port = "1123"
	}
	fmt.Println("换服务器", port)
}

func CrawOkSpot() {
	resp, err := xhttpGet("https://www.okex.com/api/spot/v3/instruments/ticker", port)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var realData []map[string]interface{}
		_ = json.Unmarshal(content, &realData)
		if len(realData) > 0 {
			for _, s := range realData {
				symbol := s["instrument_id"].(string)
				var id interface{}
				if name := util.ToMySymbol(symbol); name != "none" && name[len(name)-4:] == "USDT" {
					model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ?  and category_id = ?", 0, name, 5).Scan(&id)
					if id != nil {
						var (
							raf       float64
							dayAmount string
							price     float64
						)
						price = model.ParseStringFloat(s["last"].(string))
						raf = (price - model.ParseStringFloat(s["open_utc8"].(string))) / model.ParseStringFloat(s["open_utc8"].(string)) * 100
						dayAmount = fmt.Sprintf("%.2f", model.ParseStringFloat(s["quote_volume_24h"].(string))*6.5/100000000)
						model.ListCacheRm("ZMYCOINS", model.ParseFloatString(float64(id.(int64))), model.ParseFloatString(float64(id.(int64))))
						model.ListCacheAddOne("ZMYCOINS", makePriceInfo(price, dayAmount, raf, name, id))
					}
				}
			}
		} else {
			ChangePort()
		}
	} else {
		ChangePort()
	}
}

func makePriceInfo(price float64, dayAmount string, raf float64, name string, id interface{}) *redis.Z {
	value := map[string]interface{}{
		"price_usd":  fmt.Sprintf("%f", price),
		"price":      fmt.Sprintf("%.4f", price*6.5),
		"day_amount": dayAmount,
		"raf":        fmt.Sprintf("%.2f", raf),
		"symbol":     name,
	}
	s, _ := json.Marshal(&value)
	return &redis.Z{
		Score:  float64(id.(int64)),
		Member: s,
	}
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
		// fmt.Println(err, value)
		if err == nil {
			for k, v := range value.SubAccounts {
				// fmt.Println("hhh", v, k)
				if v.Amount > 0 {
					one := map[string]interface{}{}
					one["amount"] = decimal.NewFromFloat(v.Amount).Round(8)
					one["symbol"] = k.Symbol
					if k.Symbol != "USDT" {
						var id float64
						model.UserDB.Raw("select id from db_task_coin where category_id = ? and en_name = ? and coin_type = 0 ", cate, k.Symbol).Scan(&id)
						// fmt.Println(id)
						if id > 0 {
							one["money"] = model.GetPrice(model.ParseFloatString(id)).Mul(decimal.NewFromFloat(v.Amount)).Round(8)
						} else {
							one["money"] = 1
						}
					} else {
						one["money"] = decimal.NewFromFloat(v.Amount).Round(8)
					}
					// fmt.Println("00000", one)
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

// 请求数据
func xhttpGet(url string, port string) (*http.Response, error) {
	resp, err := util.ProxyHttp(port).Get(url)

	return resp, err
}
