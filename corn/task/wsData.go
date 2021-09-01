/***************************
@File        : wsData.go
@Time        : 2021/09/01 19:57:06
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : ws 推送数据
****************************/

package job

import (
	"fmt"
	util "zmyjobs/corn/uti"
	"zmyjobs/goex"
)

var (
    BianSpot map[string]*goex.Ticker
    names    []map[string]interface{}
    ws       goex.SpotWsApi
)

func NewBIANWsApi() {
    ws, _ = util.ProxySock().BuildSpotWs(goex.BINANCE)
    ws.TickerCallback(func(ticker []*goex.Ticker) {
        fmt.Println(ticker)
        // BianSpot[ticker.Pair.ToSymbol("")] = ticker
    })
}

func Begin() {
    NewBIANWsApi()
    // model.UserDB.Raw("select `name` from db_task_coin where coin_type = ? and category_id = ?", 0, 2).Scan(&names)
    // fmt.Println(names)

    for _, v := range names {
        fmt.Println("执行", v)
        ws.SubscribeTicker()
    }
    for {
        ;
    }
}
