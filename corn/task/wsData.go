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

func Begin() {
    NewBIANWsApi()
    fmt.Println("开启")
    ws.SubscribeTicker()
    for {
        select {
        case <-Stop:
            fmt.Println("关闭webSocket")
            runtime.Goexit()
        default:
            fmt.Println("hhh")
            time.Sleep(time.Second)
        }
    }
}
