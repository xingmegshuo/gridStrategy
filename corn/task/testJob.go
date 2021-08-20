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
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/uti"
)

//var job = model.NewJob(model.ConfigMap["jobType1"],"test","@every 5s")
var crawJob = model.NewJob(model.ConfigMap["jobType1"], "爬取基础数据", "@every 5s")

var crawLock sync.Mutex

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
	// log.Println("working for data clone ......")
	crawLock.Lock()
	runtime.GOMAXPROCS(runtime.NumCPU())
	h := model.Host{}
	h.Get("火币")
	Hurl := "https://" + h.Url
	go xhttp(Hurl+"/v1/common/symbols", "火币交易对")
	go xhttpCraw(Hurl+"/market/tickers", 1)
	go xhttpCraw("https://api.binance.com/api/v3/ticker/24hr", 2)
	go xhttpCraw("https://www.okex.com/api/spot/v3/instruments/ticker", 5)
	crawLock.Unlock()
}

// xhttp 缓存信息
func xhttp(url string, name string) {
	if !model.CheckCache(name) {
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Get(url)
		if err == nil {
			defer resp.Body.Close()
			content, _ := ioutil.ReadAll(resp.Body)
			var data = make(map[string]interface{})
			_ = json.Unmarshal(content, &data)
			byteData, _ := json.Marshal(data["data"])
			//log.Println(data["data"])
			model.SetCache(name, string(byteData), time.Hour*72)
			crawJob.Count++
			crawJob.UpdateJob()
		}
	}
}

// xhttpCraw 不缓存只更新数据   抓取最新的币种价格行情
func xhttpCraw(url string, category int) {
	client := http.Client{Timeout: 10 * time.Second}
	// client := util.ProxyHttp()
	resp, err := client.Get(url)
	// fmt.Println(err)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var data = make(map[string]interface{})
		var realData []map[string]interface{}
		if category == 1 {
			_ = json.Unmarshal(content, &data)
			byteData, _ := json.Marshal(data["data"])
			_ = json.Unmarshal(byteData, &realData)

			if !model.CheckCache("coin") {
				var coin []map[string]interface{}
				model.UserDB.Raw("select id,en_name from db_coin").Scan(&coin)
				byteData, _ := json.Marshal(coin)
				model.SetCache("coins", string(byteData), time.Second*60)
			}
			coins := model.StringMap(model.GetCache("coins"))
			for _, v := range coins {
				for _, s := range realData {
					if s["symbol"].(string) == model.ParseSymbol(v["en_name"].(string))+"usdt" {
						var coinPrice = map[string]interface{}{}
						model.UserDB.Raw("select * from db_coin_price where coin_id = ? ", v["id"].(float64)).Scan(&coinPrice)
						// log.Println("创建数据:", coinPrice)
						if len(coinPrice) == 0 {
							model.UserDB.Table("db_coin_price").Create(map[string]interface{}{"coin_id": v["id"].(float64)})
							continue
						}
						// l.Println(coinPrice, "------old")
						coinPrice["day_amount"] = s["amount"]          // 成交量
						coinPrice["open_price"] = s["open"]            // 开盘价
						coinPrice["before_price"] = coinPrice["price"] // 直前价格
						coinPrice["price_usd"] = s["close"]            // 当前价格
						raf := (s["close"].(float64) - s["open"].(float64)) / s["open"].(float64) * 100
						base := "+"
						if raf < 0 {
							base = ""
						}
						s := fmt.Sprintf("%.2f", raf) // 涨跌幅
						coinPrice["raf"] = base + s + "%"
						coinPrice["update_time"] = time.Now().Unix()
						// log.Println("更新了----new", v["en_name"], v["id"])
						model.UserDB.Table("db_coin_price").Where(map[string]interface{}{"coin_id": v["id"]}).Updates(&coinPrice)
					}
				}
			}
		}
		if category == 2 || category == 5 {
			_ = json.Unmarshal(content, &realData)
		}
		// fmt.Println(string(content))
		go WriteDB(realData, category)
	}
}

func WriteDB(realData []map[string]interface{}, category int) {
	// start := time.Now()
	var (
		coins []map[string]interface{}
	)
	model.UserDB.Raw("select name,id from db_task_coin where category_id = ? and coin_type = ?", category, 0).Scan(&coins)
	for _, s := range realData {
		// fmt.Println(s)
		// a := time.Now()
		// add := true
		var (
			symbol string
		)
		if category == 5 {
			symbol = s["instrument_id"].(string)
		} else {
			symbol = s["symbol"].(string)
		}
		if name := util.ToMySymbol(symbol); name != "none" {
			for _, coin := range coins {
				if name == coin["name"].(string) {
					// fmt.Println(s)
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

					// base := "+"
					// if raf < 0 {
					// 	base = ""
					// }
					r := fmt.Sprintf("%.2f", raf) // 涨跌幅
					// fmt.Println(r, name)w
					value := map[string]interface{}{
						"price_usd":  price,
						"price":      price * 6.5,
						"day_amount": dayAmount,
						"raf":        r,
					}

					model.UserDB.Table("db_task_coin").Where("id = ?", coin["id"]).Updates(&value)
					// add = false
				}
			}
			// if add {
			// 	var data = map[string]interface{}{}
			// 	data["name"] = name
			// 	data["coin_name"] = name
			// 	data["en_name"] = name[:len(name)-5]
			// 	data["category_id"] = category
			// 	model.UserDB.Table("db_task_coin").Create(&data)
			// }
		}
		// // fmt.Println(time.Since(a))
	}
	// fmt.Println(time.Since(start), "结束")
}
