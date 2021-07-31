package gdax

import (
	"errors"
	"fmt"
	"net/http"
	"sort"

	"zmyjobs/goex"
	"zmyjobs/goex/internal/logger"
)

//www.coinbase.com or www.gdax.com

type Gdax struct {
	httpClient *http.Client
	baseUrl,
	accessKey,
	secretKey string
}

func New(client *http.Client, accesskey, secretkey string) *Gdax {
	return &Gdax{client, "https://api.gdax.com", accesskey, secretkey}
}

func (g *Gdax) LimitBuy(amount, price string, currency goex.CurrencyPair, opt ...goex.LimitOrderOptionalParameter) (*goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) LimitSell(amount, price string, currency goex.CurrencyPair, opt ...goex.LimitOrderOptionalParameter) (*goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) MarketBuy(amount, price string, currency goex.CurrencyPair) (*goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) MarketSell(amount, price string, currency goex.CurrencyPair) (*goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) CancelOrder(orderId string, currency goex.CurrencyPair) (bool, error) {
	panic("not implement")
}
func (g *Gdax) GetOneOrder(orderId string, currency goex.CurrencyPair) (*goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) GetUnfinishOrders(currency goex.CurrencyPair) ([]goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) GetOrderHistorys(currency goex.CurrencyPair, optional ...goex.OptionalParameter) ([]goex.Order, error) {
	panic("not implement")
}
func (g *Gdax) GetAccount() (*goex.Account, error) {
	panic("not implement")
}

func (g *Gdax) GetTicker(currency goex.CurrencyPair) (*goex.Ticker, error) {
	resp, err := goex.HttpGet(g.httpClient, fmt.Sprintf("%s/products/%s/ticker", g.baseUrl, currency.ToSymbol("-")))
	if err != nil {
		errCode := goex.HTTP_ERR_CODE
		errCode.OriginErrMsg = err.Error()
		return nil, errCode
	}

	return &goex.Ticker{
		Last: goex.ToFloat64(resp["price"]),
		Sell: goex.ToFloat64(resp["ask"]),
		Buy:  goex.ToFloat64(resp["bid"]),
		Vol:  goex.ToFloat64(resp["volume"]),
	}, nil
}

func (g *Gdax) Get24HStats(pair goex.CurrencyPair) (*goex.Ticker, error) {
	resp, err := goex.HttpGet(g.httpClient, fmt.Sprintf("%s/products/%s/stats", g.baseUrl, pair.ToSymbol("-")))
	if err != nil {
		errCode := goex.HTTP_ERR_CODE
		errCode.OriginErrMsg = err.Error()
		return nil, errCode
	}
	return &goex.Ticker{
		High: goex.ToFloat64(resp["high"]),
		Low:  goex.ToFloat64(resp["low"]),
		Vol:  goex.ToFloat64(resp["volmue"]),
		Last: goex.ToFloat64(resp["last"]),
	}, nil
}

func (g *Gdax) GetDepth(size int, currency goex.CurrencyPair) (*goex.Depth, error) {
	var level int = 2
	if size == 1 {
		level = 1
	}

	resp, err := goex.HttpGet(g.httpClient, fmt.Sprintf("%s/products/%s/book?level=%d", g.baseUrl, currency.ToSymbol("-"), level))
	if err != nil {
		errCode := goex.HTTP_ERR_CODE
		errCode.OriginErrMsg = err.Error()
		return nil, errCode
	}

	bids, _ := resp["bids"].([]interface{})
	asks, _ := resp["asks"].([]interface{})

	dep := new(goex.Depth)

	for _, v := range bids {
		r := v.([]interface{})
		dep.BidList = append(dep.BidList, goex.DepthRecord{goex.ToFloat64(r[0]), goex.ToFloat64(r[1])})
	}

	for _, v := range asks {
		r := v.([]interface{})
		dep.AskList = append(dep.AskList, goex.DepthRecord{goex.ToFloat64(r[0]), goex.ToFloat64(r[1])})
	}

	sort.Sort(sort.Reverse(dep.AskList))

	return dep, nil
}

func (g *Gdax) GetKlineRecords(currency goex.CurrencyPair, period goex.KlinePeriod, size int, opt ...goex.OptionalParameter) ([]goex.Kline, error) {
	urlpath := fmt.Sprintf("%s/products/%s/candles", g.baseUrl, currency.AdaptUsdtToUsd().ToSymbol("-"))
	granularity := -1
	switch period {
	case goex.KLINE_PERIOD_1MIN:
		granularity = 60
	case goex.KLINE_PERIOD_5MIN:
		granularity = 300
	case goex.KLINE_PERIOD_15MIN:
		granularity = 900
	case goex.KLINE_PERIOD_1H, goex.KLINE_PERIOD_60MIN:
		granularity = 3600
	case goex.KLINE_PERIOD_6H:
		granularity = 21600
	case goex.KLINE_PERIOD_1DAY:
		granularity = 86400
	default:
		return nil, errors.New("unsupport the kline period")
	}
	urlpath += fmt.Sprintf("?granularity=%d", granularity)
	resp, err := goex.HttpGet3(g.httpClient, urlpath, map[string]string{})
	if err != nil {
		errCode := goex.HTTP_ERR_CODE
		errCode.OriginErrMsg = err.Error()
		return nil, errCode
	}

	var klines []goex.Kline
	for i := 0; i < len(resp); i++ {
		k, is := resp[i].([]interface{})
		if !is {
			logger.Error("data format err data =", resp[i])
			continue
		}
		klines = append(klines, goex.Kline{
			Pair:      currency,
			Timestamp: goex.ToInt64(k[0]),
			Low:       goex.ToFloat64(k[1]),
			High:      goex.ToFloat64(k[2]),
			Open:      goex.ToFloat64(k[3]),
			Close:     goex.ToFloat64(k[4]),
			Vol:       goex.ToFloat64(k[5]),
		})
	}

	return klines, nil
}

//非个人，整个交易所的交易记录
func (g *Gdax) GetTrades(currencyPair goex.CurrencyPair, since int64) ([]goex.Trade, error) {
	panic("not implement")
}

func (g *Gdax) GetExchangeName() string {
	return goex.GDAX
}
