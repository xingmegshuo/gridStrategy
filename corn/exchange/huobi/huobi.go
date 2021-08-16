package huobi

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
	logs "zmyjobs/corn/logs"

	"github.com/huobirdcenter/huobi_golang/pkg/client"
	"github.com/huobirdcenter/huobi_golang/pkg/client/orderwebsocketclient"
	"github.com/huobirdcenter/huobi_golang/pkg/client/websocketclientbase"
	"github.com/huobirdcenter/huobi_golang/pkg/model/account"
	"github.com/huobirdcenter/huobi_golang/pkg/model/auth"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"github.com/huobirdcenter/huobi_golang/pkg/model/order"
	"github.com/shopspring/decimal"
	"github.com/xyths/hs/convert"
)

var log = logs.Log
var connectLock1 sync.Mutex

type Config struct {
	Label        string
	AccessKey    string
	SecretKey    string
	CurrencyList []string
	Host         string
}

const (
	DefaultHost         = "api.huobi.pro"
	OrderTypeSellLimit  = "sell-limit"  // 限价卖出
	OrderTypeBuyLimit   = "buy-limit"   // 限价买入
	OrderTypeBuyMarket  = "buy-market"  // 市价买入
	OrderTypeSellMarket = "sell-market" // 市价卖出
)

type Client struct {
	Config        Config
	Accounts      map[string]account.AccountInfo
	CurrencyMap   map[string]bool
	SpotAccountId int64

	orderSubscriber *orderwebsocketclient.SubscribeOrderWebSocketV2Client
}

type PlaceOrderRequest struct {
	AccountId     string `json:"account-id"`
	Symbol        string `json:"symbol"`
	Type          string `json:"type"`
	Amount        string `json:"amount"`
	Price         string `json:"price,omitempty"`
	Source        string `json:"source,omitempty"`
	ClientOrderId string `json:"client-order-id,omitempty"`
	StopPrice     string `json:"stop-price,omitempty"`
	Operator      string `json:"operator,omitempty"`
}

func New(config Config) (*Client, error) {
	c := &Client{
		Config: config,
	}
	if config.Host == "" {
		c.Config.Host = DefaultHost
	}
	c.CurrencyMap = make(map[string]bool)
	for _, currency := range c.Config.CurrencyList {
		c.CurrencyMap[currency] = true
	}
	c.Accounts = make(map[string]account.AccountInfo)
	if err := c.GetAccountInfo(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) ExchangeName() string {
	return "huobi"
}

func (c *Client) Label() string {
	return c.Config.Label
}

func (c *Client) GetSpotBalance() (map[string]decimal.Decimal, error) {
	hb := new(client.AccountClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	accountBalance, err := hb.GetAccountBalance(fmt.Sprintf("%d", c.SpotAccountId))
	if err != nil {
		return nil, err
	}
	balance := make(map[string]decimal.Decimal)
	for _, b := range accountBalance.List {
		nb, err := decimal.NewFromString(b.Balance)
		if err != nil {
			log.Printf("error when parse balance: %s", err)
			continue
		}
		if nb.IsZero() {
			continue
		}
		if ob, ok := balance[b.Currency]; ok {
			balance[b.Currency] = ob.Add(nb)
		} else {
			balance[b.Currency] = nb
		}
	}
	return balance, nil
}

func (c *Client) GetPrice(symbol string) (decimal.Decimal, error) {
	hb := new(client.MarketClient).Init(c.Config.Host)

	optionalRequest := market.GetCandlestickOptionalRequest{Period: "1min", Size: 1}
	candlesticks, err := hb.GetCandlestick(symbol, optionalRequest)
	if err != nil {
		log.Println(err)
		return decimal.NewFromFloat(0), err
	}
	for _, candlestick := range candlesticks {
		return candlestick.Close, nil
	}
	return decimal.NewFromFloat(0), nil
}

func GetPriceSymbol(symbol string) (decimal.Decimal, error) {
	hb := new(client.MarketClient).Init("api.huobi.de.com")
	optionalRequest := market.GetCandlestickOptionalRequest{Period: "1min", Size: 1}
	candlesticks, err := hb.GetCandlestick(symbol, optionalRequest)
	if err != nil {
		return decimal.NewFromFloat(0), err
	}
	for _, candlestick := range candlesticks {
		return candlestick.Close, nil
	}

	return decimal.NewFromFloat(0), nil
}

func (c *Client) GetTimestamp() (int, error) {
	hb := new(client.CommonClient).Init(c.Config.Host)
	return hb.GetTimestamp()
}

func (c *Client) GetAccountInfo() error {
	hb := new(client.AccountClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	accounts, err := hb.GetAccountInfo()

	if err != nil {
		log.Printf("error when get account info: %s", err)
		return err
	}
	for _, acc := range accounts {
		c.Accounts[acc.Type] = acc
		c.SpotAccountId = acc.Id
	}

	return nil
}

func (c *Client) Balances() (map[string]float64, error) {
	balances := make(map[string]float64)
	hb := new(client.AccountClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	for _, acc := range c.Accounts {
		ab, err := hb.GetAccountBalance(fmt.Sprintf("%d", acc.Id))
		if err != nil {
			log.Printf("[ERROR] error when get account %d balance: %s", acc.Id, err)
			return balances, err
		}
		for _, b := range ab.List {
			if !c.CurrencyMap[b.Currency] {
				continue
			}
			realBalance := convert.StrToFloat64(b.Balance)
			balances[b.Currency] += realBalance
		}
	}
	return balances, nil
}

func (c *Client) LastPrice(symbol string) (float64, error) {
	hb := new(client.MarketClient).Init(c.Config.Host)
	optionalRequest := market.GetCandlestickOptionalRequest{Period: market.MIN1, Size: 1}
	candlesticks, err := hb.GetCandlestick(symbol, optionalRequest)
	if err != nil {
		return 0, err
	}
	for _, cs := range candlesticks {
		price, _ := cs.Close.Float64()
		return price, nil
	}
	return 0, nil
}

func (c *Client) Snapshot(ctx context.Context, result interface{}) error {
	huobiBalance, ok := result.(*HuobiBalance)
	if !ok {
		return errors.New("bad result type, should be *HuobiBalance")
	}
	balanceMap, err := c.Balances()
	if err != nil {
		return err
	}
	huobiBalance.Label = c.Config.Label
	huobiBalance.BTC = balanceMap["btc"]
	huobiBalance.USDT = balanceMap["usdt"]
	huobiBalance.HT = balanceMap["ht"]
	btcPrice, err := c.LastPrice("btcusdt")
	if err != nil {
		return err
	}
	huobiBalance.BTCPrice = btcPrice
	htPrice, err := c.LastPrice("htusdt")
	if err != nil {
		return err
	}
	huobiBalance.HTPrice = htPrice
	huobiBalance.Time = time.Now()
	return nil
}

func (c *Client) SubscribeOrders(clientId string, responseHandler websocketclientbase.ResponseHandler) error {
	//hb := new(orderwebsocketclient.SubscribeOrderWebSocketV2Client).Init(c.Config.AccessKey, c.Config.SecretKey, Host)
	//hb.SetHandler(
	//	// Authentication response handler
	//	func(resp *auth.WebSocketV2AuthenticationResponse) {
	//		if resp.IsAuth() {
	//			err := hb.Subscribe("1", clientId)
	//			if err != nil {
	//				log.Printf("Subscribe error: %s\n", err)
	//			} else {
	//				log.Println("Sent subscription")
	//			}
	//		} else {
	//			log.Printf("Authentication error: %d\n", resp.Code)
	//		}
	//	},
	//	responseHandler)
	//return hb.Connect(true)
	return nil
}

func (c *Client) SubscribeOrder(ctx context.Context, symbol, clientId string,
	responseHandler websocketclientbase.ResponseHandler) {
	hb := new(orderwebsocketclient.SubscribeOrderWebSocketV2Client).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	Connect := 0
	var thisLock sync.Mutex
	hb.SetHandler(
		// Connected handler
		func(resp *auth.WebSocketV2AuthenticationResponse) {
			if resp.IsSuccess() {
				// Subscribe if authentication passed
				hb.Subscribe(symbol, clientId)
			} else {
				log.Fatalf("Authentication error, code: %d, message:%s", resp.Code, resp.Message)
			}
		},
		responseHandler)
	for {
		select {
		case <-ctx.Done():
			log.Println("close websocket 1 ----------")
			hb.UnSubscribe(symbol, clientId)
			hb.Close()
			return
		default:
		}
		if Connect == 0 {
			thisLock.Lock()
			Connect = 1
			thisLock.Unlock()
			log.Println("开启websocket 1 ----------")
			hb.Connect(true)
		}
	}
}

func (c *Client) PlaceOrders(orderType, symbol string, price, amount float64) (uint64, error) {
	hb := new(client.OrderClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)

	strPrice := fmt.Sprintf("%."+strconv.Itoa(PricePrecision[symbol])+"f", price)
	strAmount := fmt.Sprintf("%."+strconv.Itoa(AmountPrecision[symbol])+"f", amount)
	request := order.PlaceOrderRequest{
		AccountId: fmt.Sprintf("%d", c.Accounts["spot"].Id),
		Type:      orderType,
		Source:    "api",
		Symbol:    symbol,
		Price:     strPrice,
		Amount:    strAmount,
	}
	resp, err := hb.PlaceOrder(&request)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	switch resp.Status {
	case "ok":
		log.Printf("Place order successfully, order id: %s\n", resp.Data)
		return convert.StrToUint64(resp.Data), nil
	case "error":
		log.Printf("Place order error: %s\n", resp.ErrorMessage)
		if resp.ErrorCode == "account-frozen-balance-insufficient-error" {
			return 0, nil
		}
		return 0, errors.New(resp.ErrorMessage)
	}

	return 0, errors.New("unknown status")
}

//
//func (c *Client) Sell(symbol string, price, amount float64) (orderId uint64, err error) {
//	return c.PlaceOrder("sell-limit", symbol, price, amount)
//}

//func (c *Client) Buy(symbol string, price, amount float64) (orderId uint64, err error) {
//	return c.PlaceOrder("buy-limit", symbol, price, amount)
//}

func (c *Client) SubscribeBalanceUpdate(clientId string, responseHandler websocketclientbase.ResponseHandler) error {
	//hb := new(accountwebsocketclient.SubscribeAccountWebSocketV2Client).Init(c.Config.AccessKey, c.Config.SecretKey, Host)
	//hb.SetHandler(
	//	// Authentication response handler
	//	func(resp *auth.WebSocketV2AuthenticationResponse) {
	//		if resp.IsAuth() {
	//			err := hb.Subscribe("1", clientId)
	//			if err != nil {
	//				log.Printf("Subscribe error: %s\n", err)
	//			} else {
	//				log.Println("Sent subscription")
	//			}
	//		} else {
	//			log.Printf("Authentication error: %d\n", resp.Code)
	//		}
	//	},
	//	responseHandler)
	//return hb.Connect(true)
	return nil
}

func (c *Client) SubscribeTradeClear(ctx context.Context, symbol, clientId string,
	responseHandler websocketclientbase.ResponseHandler) {
	hb := new(orderwebsocketclient.SubscribeTradeClearWebSocketV2Client).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	hb.SetHandler(
		// Connected handler
		func(resp *auth.WebSocketV2AuthenticationResponse) {
			if resp.IsSuccess() {
				// Subscribe if authentication passed
				hb.Subscribe(symbol, clientId)
			} else {
				log.Printf("Authentication error, code: %d, message:%s", resp.Code, resp.Message)
			}
		},
		responseHandler)
	Connect := 0
	for {
		select {
		case <-ctx.Done():
			hb.UnSubscribe(symbol, clientId)
			time.Sleep(time.Second * 5)
			hb.Close()
			connectLock1.Unlock()

			// log.Printf("UnSubscribed, symbol = %s, clientId = %s", symbol, clientId)
			return
		default:

		}
		if Connect == 0 {
			connectLock1.Lock()
			Connect = 1
			hb.Connect(true)
		}
	}
}

func (c *Client) CancelOrders(orderId uint64) error {
	hb := new(client.OrderClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	resp, err := hb.CancelOrderById("1")
	if err != nil {
		log.Println(err)
		return err
	}
	switch resp.Status {
	case "ok":
		log.Printf("Cancel order successfully, order id: %s\n", resp.Data)
		return nil
	case "error":
		log.Printf("Cancel order error: %s\n", resp.ErrorMessage)
		return errors.New(resp.ErrorMessage)
	}

	return nil
}

func (c *Client) CancelOrder(orderId uint64) (int, error) {
	hb := new(client.OrderClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	resp, err := hb.CancelOrderById(fmt.Sprintf("%d", orderId))
	if err != nil {
		return 0, err
	}
	if resp == nil {
		return 0, nil
	}
	errorCode, err := strconv.Atoi(resp.ErrorCode)
	if err != nil {
		return 0, nil
	}
	return errorCode, errors.New(resp.ErrorMessage)
}

func (c *Client) PlaceOrder(orderType, symbol, clientOrderId string, price, amount decimal.Decimal) (uint64, error) {
	hb := new(client.OrderClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	request := order.PlaceOrderRequest{
		AccountId:     fmt.Sprintf("%d", c.SpotAccountId),
		Type:          orderType,
		Source:        "spot-api",
		Symbol:        symbol,
		Price:         price.String(),
		Amount:        amount.String(),
		ClientOrderId: clientOrderId,
	}
	resp, err := hb.PlaceOrder(&request)
	if err != nil {
		log.Println(err)
		return 0, err
	}
	switch resp.Status {
	case "ok":
		log.Printf("Place order successfully, order id: %s, clientOrderId: %s\n", resp.Data, clientOrderId)
		return convert.StrToUint64(resp.Data), nil
	case "error":
		log.Printf("Place order error: %s\n", resp.ErrorMessage)
		if resp.ErrorCode == "account-frozen-balance-insufficient-error" {
			return 0, nil
		}
		return 0, errors.New(resp.ErrorMessage)
	}
	return 0, errors.New("unknown status")
}

func (c *Client) SearchOrder(order string) (map[string]string, bool, error) {
	var data = map[string]string{}

	hb := new(client.OrderClient).Init(c.Config.AccessKey, c.Config.SecretKey, c.Config.Host)
	response, err := hb.GetOrderById(order)
	// fmt.Println(resp, err)
	if err != nil {
		return nil, false, err
	} else if response != nil {
		if response.Data != nil {
			if response.Data.State == "filled" {
				data["amount"] = response.Data.FilledAmount
				data["price"] = response.Data.Price
				data["fee"] = response.Data.FilledFees
				return data, true, nil
			} else if response.Data.State == "submitted" {
				return nil, true, errors.New("等一会")
			}
		}
	}
	return data, false, nil
}
