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
	"os"
	"runtime"
	"time"
	util "zmyjobs/corn/uti"
	"zmyjobs/goex"
)

var (
    BianSpot = map[string]*goex.Ticker{}
    names    []map[string]interface{}
    ws       goex.SpotWsApi
    Stop     = make(chan int)
)

// NewBIANWsApi 新建币安websocket 现货行情
func NewBIANWsApi() {
    os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:1124")
    ws, _ = util.ProxySock().BuildSpotWs(goex.BINANCE)
    ws.TickerCallback(func(ticker []*goex.Ticker) {
        // fmt.Println(ticker)
        time.Sleep(time.Second * 2)
        BianSpot = map[string]*goex.Ticker{}
        for _, t := range ticker {
            BianSpot[t.Pair.ToSymbol("")] = t
        }
    })
}

// Begin 开启websocket 连接更新行情
func Begin() {
    NewBIANWsApi()
    start := time.Now()
    fmt.Println("开启websocket")
    ws.SubscribeTicker()
    for {
        select {
        case <-Stop:
            fmt.Println("关闭webSocket")
            runtime.Goexit()
        default:
            time.Sleep(time.Second)
            if time.Since(start) > time.Minute {
                go Begin()
                Stop <- 1
            }
        }
    }
}