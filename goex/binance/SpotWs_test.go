package binance

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"
	goex "zmyjobs/goex"
)

var spotWs *SpotWs

func createSpotWs() {
	os.Setenv("HTTPS_PROXY", "socks5://127.0.0.1:1124")
	spotWs = NewSpotWs()
	spotWs.DepthCallback(func(depth *goex.Depth) {
		log.Println(depth)
	})
	spotWs.TickerCallback(func(ticker []*goex.Ticker) {
		fmt.Println(ticker)
	})
}

func TestSpotWs_DepthCallback(t *testing.T) {
	createSpotWs()
	spotWs.SubscribeDepth(goex.BTC_USDT)
	spotWs.SubscribeTicker()
	time.Sleep(11 * time.Minute)
}

func TestSpotWs_SubscribeTicker(t *testing.T) {
	createSpotWs()
	spotWs.SubscribeTicker()
	time.Sleep(30 * time.Minute)
}
