package binance

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
	goex "zmyjobs/goex"
	"zmyjobs/goex/internal/logger"
)

var baDapi = NewBinanceFutures(&goex.APIConfig{
	HttpClient: &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse("socks5://127.0.0.1:1123")
				// return nil, nil
			},
			Dial: (&net.Dialer{
				Timeout: 10 * time.Second,
			}).Dial,
		},
		Timeout: 10 * time.Second,
	},
	// HttpClient:   http.DefaultClient,
	ApiKey:       "cIiIZnQ8L77acyTkSAH6je0rDAZGoFcoHSlMHaWYUNDjJhKNtu0Gb8nR9MjLSaws",
	ApiSecretKey: "dnc7LwiB1Vgy6zDOqwuqodIxL8FwltVlxhfEVLvrgmfozezrW9JvnHStpmB4Lymx",
})

func init() {
	logger.SetLevel(logger.DEBUG)
}

func TestBinanceFutures_GetFutureDepth(t *testing.T) {
	t.Log(baDapi.GetFutureDepth(goex.ETH_USD, goex.QUARTER_CONTRACT, 10))
}

func TestBinanceSwap_GetFutureTicker(t *testing.T) {
	// 价格 季度和永续
	ticker, err := baDapi.GetFutureTicker(goex.LTC_USD, goex.QUARTER_CONTRACT)
	t.Log(err)
	t.Logf("%+v", ticker)
}

func TestBinance_GetExchangeInfo(t *testing.T) {
	//
	baDapi.GetExchangeInfo()
}

func TestBinanceFutures_GetFutureUserinfo(t *testing.T) {
	// 资产
	c, err := baDapi.GetFutureUserinfo(goex.BCD_BTC)
	fmt.Println(fmt.Sprintf("%+v", c), err)
	// t.Log()
}

func TestBinanceFutures_PlaceFutureOrder(t *testing.T) {
	//1044675677
	// 交易
	t.Log(baDapi.PlaceFutureOrder(goex.BTC_USD, goex.QUARTER_CONTRACT, "19990", "2", goex.OPEN_SELL, 0, 10))
}

func TestBinanceFutures_LimitFuturesOrder(t *testing.T) {
	// 限价开空
	t.Log(baDapi.LimitFuturesOrder(goex.BTC_USD, goex.QUARTER_CONTRACT, "20001", "2", goex.OPEN_SELL))
}

func TestBinanceFutures_MarketFuturesOrder(t *testing.T) {
	// 市价开空
	t.Log(baDapi.MarketFuturesOrder(goex.BTC_USD, goex.QUARTER_CONTRACT, "2", goex.OPEN_SELL))
}

func TestBinanceFutures_GetFutureOrder(t *testing.T) {
	// 查询订单
	t.Log(baDapi.GetFutureOrder("1045208666", goex.BTC_USD, goex.QUARTER_CONTRACT))
}

func TestBinanceFutures_FutureCancelOrder(t *testing.T) {
	// 撤单
	t.Log(baDapi.FutureCancelOrder(goex.BTC_USD, goex.QUARTER_CONTRACT, "1045328328"))
}

func TestBinanceFutures_GetFuturePosition(t *testing.T) {
	// 持仓风险
	curr := goex.NewCurrencyPair2("ETH_USD")

	t.Log(baDapi.GetFuturePosition(curr, goex.QUARTER_CONTRACT))


}

func TestBinanceFutures_GetUnfinishFutureOrders(t *testing.T) {
	t.Log(baDapi.GetUnfinishFutureOrders(goex.BTC_USD, goex.QUARTER_CONTRACT))
}
