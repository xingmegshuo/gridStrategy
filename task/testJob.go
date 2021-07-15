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
	"time"

	model "zmyjobs/models"
)

//var job = model.NewJob(model.ConfigMap["jobType1"],"test","@every 5s")
var crawJob = model.NewJob(model.ConfigMap["jobType1"], "爬取基础数据", "@every 5s")

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
	h := model.Host{}
	h.Get("火币")
	Hurl := "https://" + h.Url
	xhttp(Hurl+"/v1/common/symbols", "火币交易对")
	go xhttpCraw(Hurl + "/market/tickers")
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
func xhttpCraw(url string) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err == nil {
		defer resp.Body.Close()
		content, _ := ioutil.ReadAll(resp.Body)
		var data = make(map[string]interface{})
		_ = json.Unmarshal(content, &data)
		byteData, _ := json.Marshal(data["data"])
		var realData = []map[string]interface{}{}
		// l.Println(data["data"])
		_ = json.Unmarshal(byteData, &realData)
		// l.Println(realData[0])
		// byteData, _ := json.Marshal(data["data"])
		// l.Println(data["data"])
		var coin = []map[string]interface{}{}
		model.UserDB.Raw("select id,en_name from db_coin").Scan(&coin)
		for _, v := range coin {
			for _, s := range realData {
				if s["symbol"].(string) == model.ParseSymbol(v["en_name"].(string))+"usdt" {
					var coinPrice = map[string]interface{}{}
					model.UserDB.Raw("select * from db_coin_price where coin_id = ? ", v["id"].(int32)).Scan(&coinPrice)
					// log.Println("创建数据:", coinPrice)

					if len(coinPrice) == 0 {
						model.UserDB.Table("db_coin_price").Create(map[string]interface{}{"coin_id": v["id"].(int32)})
						continue
					}
					// l.Println(coinPrice, "------old")
					coinPrice["day_amount"] = s["amount"]          // 成交量
					coinPrice["open_price"] = s["open"]            // 开盘价
					coinPrice["before_price"] = coinPrice["price"] // 直前价格
					coinPrice["price"] = s["close"]                // 当前价格
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
}
