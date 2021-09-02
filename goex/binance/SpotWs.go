package binance

import (
	json2 "encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
	"zmyjobs/goex/internal/logger"

	"zmyjobs/goex"
)

type req struct {
	Method string   `json:"method"`
	Params []string `json:"params"`
	Id     int      `json:"id"`
}

type resp struct {
	Stream string           `json:"stream"`
	Data   json2.RawMessage `json:"data"`
}

type depthResp struct {
	LastUpdateId int             `json:"lastUpdateId"`
	Bids         [][]interface{} `json:"bids"`
	Asks         [][]interface{} `json:"asks"`
}

type SpotWs struct {
	c         *goex.WsConn
	once      sync.Once
	wsBuilder *goex.WsBuilder

	reqId int

	depthCallFn  func(depth *goex.Depth)
	tickerCallFn func(ticker []*goex.Ticker)
	tradeCallFn  func(trade *goex.Trade)
}

func NewSpotWs() *SpotWs {
	spotWs := &SpotWs{}
	fmt.Printf("proxy url: %s", os.Getenv("HTTPS_PROXY"))
	spotWs.wsBuilder = goex.NewWsBuilder().
		WsUrl("wss://stream.binance.com:9443/stream?streams=depth/miniTicker/ticker/trade").
		ProxyUrl(os.Getenv("HTTPS_PROXY")).
		ProtoHandleFunc(spotWs.handle).AutoReconnect()

	spotWs.reqId = 1

	return spotWs
}

func (s *SpotWs) connect() {
	s.once.Do(func() {
		s.c = s.wsBuilder.Build()
	})
}

func (s *SpotWs) DepthCallback(f func(depth *goex.Depth)) {
	s.depthCallFn = f
}

func (s *SpotWs) TickerCallback(f func(ticker []*goex.Ticker)) {
	s.tickerCallFn = f
}

func (s *SpotWs) TradeCallback(f func(trade *goex.Trade)) {
	s.tradeCallFn = f
}

func (s *SpotWs) SubscribeDepth(pair goex.CurrencyPair) error {
	defer func() {
		s.reqId++
	}()

	s.connect()

	return s.c.Subscribe(req{
		Method: "SUBSCRIBE",
		Params: []string{
			fmt.Sprintf("%s@depth10@100ms", pair.ToLower().ToSymbol("")),
		},
		Id: s.reqId,
	})
}

func (s *SpotWs) SubscribeTicker() error {
	defer func() {
		s.reqId++
	}()
	fmt.Println("连接...")
	s.connect()
	fmt.Println("连接成功...")
	return s.c.Subscribe(req{
		Method: "SUBSCRIBE",
		Params: []string{
			"!ticker@arr",
		},
		Id: s.reqId,
	})
}

func (s *SpotWs) SubscribeTrade(pair goex.CurrencyPair) error {
	panic("implement me")
}

func (s *SpotWs) handle(data []byte) error {
	var r resp
	err := json2.Unmarshal(data, &r)
	// fmt.Println(err, data)
	if err != nil {
		logger.Errorf("json unmarshal ws response error [%s] , response data = %s", err, string(data))
		return err
	}

	if strings.HasSuffix(r.Stream, "@depth10@100ms") {
		return s.depthHandle(r.Data, adaptStreamToCurrencyPair(r.Stream))
	}

	if strings.Contains(r.Stream, "ticker") {
		// fmt.Println("hhhh")
		return s.tickerHandle(r.Data)
	}

	logger.Warn("unknown ws response:", string(data))

	return nil
}

func (s *SpotWs) depthHandle(data json2.RawMessage, pair goex.CurrencyPair) error {
	var (
		depthR depthResp
		dep    goex.Depth
		err    error
	)

	err = json2.Unmarshal(data, &depthR)
	if err != nil {
		logger.Errorf("unmarshal depth response error %s[] , response data = %s", err, string(data))
		return err
	}

	dep.UTime = time.Now()
	dep.Pair = pair

	for _, bid := range depthR.Bids {
		dep.BidList = append(dep.BidList, goex.DepthRecord{
			Price:  goex.ToFloat64(bid[0]),
			Amount: goex.ToFloat64(bid[1]),
		})
	}

	for _, ask := range depthR.Asks {
		dep.AskList = append(dep.AskList, goex.DepthRecord{
			Price:  goex.ToFloat64(ask[0]),
			Amount: goex.ToFloat64(ask[1]),
		})
	}

	sort.Sort(sort.Reverse(dep.AskList))

	s.depthCallFn(&dep)

	return nil
}

func (s *SpotWs) tickerHandle(data json2.RawMessage) error {
	var (
		tickerDatas = []map[string]interface{}{}
		tickerData  = map[string]interface{}{}
		tickers     []*goex.Ticker
	)

	err := json2.Unmarshal(data, &tickerDatas)
	// fmt.Println("hhhh")
	if err != nil {
		logger.Errorf("unmarshal ticker response data error [%s] , data = %s", err, string(data))
		return err
	}

	for _, tickerData = range tickerDatas {
		// fmt.Println(tickerData["s"])
		str := tickerData["s"].(string)
		if str[len(str)-4:] == "USDT" {
			ticker := goex.Ticker{}
			ticker.Pair = goex.NewCurrencyPair2(str[:len(str)-4] + "/" + "USDT")
			ticker.Vol = goex.ToFloat64(tickerData["v"])
			ticker.Last = goex.ToFloat64(tickerData["c"])
			ticker.Sell = goex.ToFloat64(tickerData["a"])
			ticker.Buy = goex.ToFloat64(tickerData["b"])
			ticker.High = goex.ToFloat64(tickerData["h"])
			ticker.Low = goex.ToFloat64(tickerData["l"])
			ticker.Date = goex.ToUint64(tickerData["E"])
			ticker.Raf = goex.ToFloat64(tickerData["P"])
			tickers = append(tickers, &ticker)
		}
	}
	s.tickerCallFn(tickers)
	return nil
}
