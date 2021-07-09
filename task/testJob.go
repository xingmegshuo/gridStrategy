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
	"io/ioutil"
	"net/http"
	"time"
	model"zmyjobs/models"
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
