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
	"sync"
	"time"
	model "zmyjobs/corn/models"
	util "zmyjobs/corn/uti"

	"github.com/go-redis/redis/v8"
)

//var job = model.NewJob(model.ConfigMap["jobType1"],"test","@every 5s")
var (
	crawJob  = model.NewJob(model.ConfigMap["jobType1"], "爬取基础数据", "@every 3s")
	crawLock sync.Mutex
	// coins     []map[string]interface{}
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
	coinCache := []*redis.Z{}
	go craw(coinCache)
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

// craw
func craw(coinCache []*redis.Z) {
	start := time.Now()
	coinCache = append(coinCache, xhttpCraw("https://api.huobi.pro/market/tickers", 1)...)
	coinCache = append(coinCache, xhttpCraw("https://api.binance.com/api/v3/ticker/24hr", 2)...)
	coinCache = append(coinCache, xhttpCraw("https://www.okex.com/api/spot/v3/instruments/ticker", 5)...)

	fmt.Println(len(coinCache), coinCount, time.Since(start))
	if len(coinCache) == coinCount {
		fmt.Println("write db")
		model.Del("coins")
		model.AddCache("coins", coinCache...)
		coinCache = []*redis.Z{}
	}
}

// xhttpCraw 不缓存只更新数据   抓取最新的币种价格行情
func xhttpCraw(url string, category int) []*redis.Z {
	client := http.Client{Timeout: 10 * time.Second}
	// client := util.ProxyHttp()
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
		return WriteDB(realData, category)
	} else {
		return []*redis.Z{}
	}
}

func WriteDB(realData []map[string]interface{}, category int) (coinCache []*redis.Z) {
	// start := time.Now()
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
			model.UserDB.Raw("select id from db_task_coin where coin_type = ? and name = ? and category_id = ?", 0, name, category).Scan(&id)
			// fmt.Println(id)
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
					"price_usd":  price,
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
