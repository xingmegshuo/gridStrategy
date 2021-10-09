package grid

import (
	"fmt"
	"testing"
	"time"
	model "zmyjobs/corn/models"

	"github.com/shopspring/decimal"
	// "github.com/nntaoli-project/goex"
)

// bian Test api mhRYW9GrdeEZ6nNKdZsoGYg4S3DlvrHNxxTqNaP60m3SDz09TwXe5rGDw7YrEVyp secretKey lkUxg31nEBExSprEXX3wRgcGIte30UaYxZmfSW0b31sClkVa0hjrtgCEV5AsxJXB
// bian my api cIiIZnQ8L77acyTkSAH6je0rDAZGoFcoHSlMHaWYUNDjJhKNtu0Gb8nR9MjLSaws secretKey dnc7LwiB1Vgy6zDOqwuqodIxL8FwltVlxhfEVLvrgmfozezrW9JvnHStpmB4Lymx

// ok my api 61d888b4-bb69-461e-81e9-3987604ab08b FD78BBB9BCEB06DCD27E8CEFF6D51E74

func TestGetMoney(t *testing.T) {
	// fmt.Println("testing start .....")
	// huobiSymbol := model.SymbolCategory{
	// 	Key:    "405a3c54-5dce3618-7yngd7gh5g-326cc", // 不是我的
	// 	Secret: "4e59ae57-01e1c38c-cb63e660-d273b",

	// Key:    "7b5d41cb-8f7e2626-h6n2d4f5gh-c2d91", // 李彬彬
	// Secret: "f09fba02-a5c1b946-d2b57c4e-04335",

	// Key:             "8c892879-hrf5gdfghe-4332a67f-a0851", //我的
	// Secret:          "5af10cd5-ca04c723-6a621d2e-32c76",   // 我的
	// 	Host:            "api.huobi.de.com",
	// 	BaseCurrency:    "DOGE",
	// 	QuoteCurrency:   "USDT",
	// 	AmountPrecision: 4,
	// 	PricePrecision:  2,
	// }

	bian := model.SymbolCategory{
		Key:             "b0pHDmqinoKMWW4mlAj4mJ6urfwgI3F1OnACcZr3Vocr0RJ1NVHcdYniqAwac6CA",
		Secret:          "UxtPFv116Y4pC6itHH25guzVfTO0lhiKKA5jyFcsSmsgqFcubPmrVQWBrRZP613f",
		BaseCurrency:    "SAND",
		QuoteCurrency:   "USDT",
		AmountPrecision: 8,
		PricePrecision:  8,
		Category:        "ok",
		Label:           "u20",
	}

	// 账户测试
	ex := NewEx(&bian)
	b, m, coin := ex.GetAccount()
	fmt.Println(b, m, coin)
	// 价格测试
	// price, err := ex.GetPrice()
	// fmt.Println(price, err)
	// 交易测试
	// price = price.Sub(price.Mul(decimal.NewFromFloat(0.001))).Round(ex.symbol.PricePrecision)
	// price := decimal.NewFromFloat(30000.11).Round(huobi.PricePrecision)

	// c, _ := huobi.New(huobi.Config{Host: huobiSymbol.Host, Label: huobiSymbol.Label, SecretKey: huobiSymbol.Secret, AccessKey: huobiSymbol.Key})
	// n, err := ex..PlaceOrder("sell-market", "ethusdt", "test", price, amount)
	// fmt.Println(n, err)
	// .Round(ex.symbol.AmountPrecision)
	// fmt.Println(amount)

	// price := decimal.NewFromFloat(0)
	// 买入
	// amount := decimal.NewFromFloat(21)
	// o, err := ex.Exchanges(amount, price, SellM, false)
	// fmt.Println(o, err)

	//
	// cliId := "338280716011966"
	// time.Sleep(time.Second * 10)

	// b, c, o := ex.SearchOrder(cliId)
	// fmt.Println(b, c, o)

	// order, err := ex.Ex.GetOrderHistorys(ex.MakePair())
	// fmt.Println(order, err)

	// 撤单
	// if !c {
	// 	b = ex.CancelOrder(cliId)
	// 	fmt.Println(b)
	// }

	// goex.LimitOrderOptionalParameter
	// c, e := ex.Ex.LimitSell("0.025", "2400.00", ex.MakePair(), 332301627399393)
	// fmt.Println(c, e)

	// b, h := ex.Ex.GetOneOrder("332313548499641", ex.MakePair())
	// fmt.Println(b.Cid, b.Status, h)
	// b, e := ex.Ex.CancelOrder("332301627399393", ex.MakePair())
	// fmt.Println(b, e)
	// ex.GetPrice()

	// b, h := ex.Ex.MarketSell("0.025", "2400.00", ex.MakePair())
	// fmt.Println(b, h)

	// 稳定测试

	// o := false
	// var v = []map[time.Duration]bool{}
	// for i := 1; i < 2; i++ {
	// 	ctx, cancel := context.WithCancel(context.Background())
	// 	defer cancel()
	// 	u.ObjectId = int32(i)
	// 	cli := NewEx(&symbol)
	// 	time.Sleep(time.Millisecond * 500)
	// 	go RunEx(ctx, u, cli)
	// }

	// for {
	// 	if len(v) == 200 {
	// 		o = true
	// 	}
	// 	if o {
	// 		break
	// 	}
	// }

	// 期货
	// fmt.Println(cli.Future.GetExchangeName()) //交易所名字
	// b, err := cli.Future.GetFee()

	//获取深度
	// b, err := cli.Future.GetFutureDepth(goex.BTC_USDT, goex.SWAP_CONTRACT, 1)
	// fmt.Println(b, err)
	// b := goex.NewCurrencyPair2("FIL_USDT")

	// 持仓查询
	// p, err := cli.Future.GetFuturePosition(b, goex.SWAP_USDT_CONTRACT)
	// fmt.Println(fmt.Sprintf("%+v", p), err)

	// 账户信息
	// account, e :=
	// cli.FutureAccount("aa")
	// fmt.Println(account, e)

	// 合约价值
	// p, err := cli.Future.GetFutureTicker(b, goex.SWAP_CONTRACT)
	// fmt.Println(p, err)

	// p, err := cli.Future
}

func TestFutureAccount(t *testing.T) {
	// u := model.User{
	// 	Symbol: `{"Category":"币安","Symbol":"ETH/USDT","AmountPrecision":2,
	// 		"PricePrecision":6,"Label":"goex-币安-dogeusdt-33","Key":"DwKk9rNVVPQuDpvU3Z2grTPZBxgWqZQt5GRtVbJp7cswsRv1UqjxAUxONWJNwycF",
	// 		"Secret":"pwWvUZiC8iu7KHIM1e2ai99om20yALMkwzrWGJ3CqFp9j28E2bRIlsVfwb8E0ioL","Host":"","MinTotal":"5","MinAmount":"1",
	// 		"BaseCurrency":"ETH","QuoteCurrency:"USDT","future":true,"lever":20}`,
	// 	Arg: `{"FirstBuy": 6, "FirstDouble": false, "Price": 0.196562, "IsChange": false, "Rate": 20,
	// 			"MinBuy": 6, "AddRate": 50, "NeedMoney": 0, "MinPrice": 0, "Callback": 0.2, "Reduce": 0.2, "Stop": 5,
	// 			"AddMoney": 0, "Decline": 0.5, "Out": false, "OrderType": 0, "IsLimit": false, "LimitHigh": 0, "StrategyType": 1,
	// 			"Crile": true, "OneBuy": false, "AllSell": false, "StopBuy": false, "IsHand": false}`,
	// 	Strategy: `{"FirstBuy":"6.00000000","NowPrice":"0.00000000","Strategy":1,"Type":2,"add":"0.0000","allSell":1,"callback":"0.20",
	// 		"decline":"0.50","double":1,"frequency":2,"growth":"50.00","high_price":"0.00000000","limit_high":1,"one_buy":1,"order_type":1,
	// 		"rate":"20.00","reduce":"0.20","reset":"20.00","stop":"5.00","stop_buy":1}`,
	// 	Grids: `[{"Id":1,"Price":"0.196562","AmountBuy":"30.52","Decline":0.5,"AmountSell":"0","TotalBuy":"6","Order":0},
	// 		{"Id":2,"Price":"0.195087785","AmountBuy":"36.91","Decline":0.75,"AmountSell":"0","TotalBuy":"7.2","Order":0},
	// 		{"Id":3,"Price":"0.1928764625","AmountBuy":"44.8","Decline":1.125,"AmountSell":"0","TotalBuy":"8.64","Order":0}]`,
	// 	Number: 3,
	// 	Base:   0,
	// 	Custom: 2,
	// }
	// symbol := model.StringSymobol(u.Symbol)

	// ok 模拟盘 : b3f3f6e2-7243-4602-bdaf-2de3fee7564f   CF67295C5D9450ED625206EB04285D52
	//    ok 实盘 : eb861d6c-711c-4e16-a656-48839b5b1dd1  90775A06A926AC9AFDAA657C2AF06ED1
	bian := model.SymbolCategory{
		Key:    "b3f3f6e2-7243-4602-bdaf-2de3fee7564f",
		Secret: "CF67295C5D9450ED625206EB04285D52",
		Symbol: "ETH/USDT",
		// Host:            "https://fapi.binace.com",
		BaseCurrency:    "ETH",
		QuoteCurrency:   "USDT",
		AmountPrecision: 2,
		PricePrecision:  6,
		Category:        "ok",
		Label:           "u20",
		Future:          true,
		Lever:           60,
	}
	// fmt.Println(bian)
	cli := NewEx(&bian)

	// b := goex.NewCurrencyPair2(bian.Symbol)
	// fmt.Println("获取账户信息:")
	// fmt.Println(cli.GetAccount())

	// fmt.Println(b.String())
	// fmt.Println("修改交易对杠杆倍数:")
	// fmt.Println(cli.Future.ChangeLever(b, goex.SWAP_USDT_CONTRACT))

	// fmt.Println(cli.Currency)
	// cl := util.Config{Name: "币安"}
	// p, err := cl.GetPrice("ETH/USD", true)
	// fmt.Println(p, err)
	// p, err := cli.Future.GetFuturePosition(b, goex.SWAP_CONTRACT)

	// p, err := cli.Future.GetFuturePosition(b, goex.SWAP_CONTRACT)
	// fmt.Println(fmt.Sprintf("持仓数据:%+v", p), err)
	// o, err := cli.Future.MarketFuturesOrder(b, goex.SWAP_CONTRACT, "1", 1)
	// fmt.Println(o, err)

	order, result := cli.Exchanges(decimal.NewFromFloat(100), decimal.Decimal{}, OpenLL, false)
	fmt.Println("下单数据", order, result)

	fmt.Println("查找订单数据:")
	// orderId := "363283586224631809"
	orderId := order.OrderId
	time.Sleep(time.Second * 5)
	n, r, o := cli.SearchOrder(orderId)
	fmt.Println(fmt.Sprintf("返回结果%+v", o), n, r)

	// ordierId, clientId, err := cli.Exchanges(decimal.NewFromFloat(0.001), decimal.NewFromFloat(39500), OpenDL, true)
	// fmt.Println(ordierId, clientId, err)
	// orderId := "8389765506715143168"
	// h, over, o := cli.SearchOrder(orderId)
	// fmt.Println(h, over, o)

	// ordierId, clientId, err := cli.Exchanges(decimal.NewFromFloat(0.001), decimal.NewFromFloat(39500), OpenDL, true)
	// fmt.Println(ordierId, clientId, err)
	// orderId := "8389765504833933312"
	// h, over, o := cli.SearchOrder(orderId)
	// fmt.Println(h, over, o)

	// b := cli.CancelOrder(orderId)
	// fmt.Println(b)
}

func TestOk(t *testing.T) {
	ok := model.SymbolCategory{
		Key:    "76141669-f4cf-4f45-ad6d-82522539749b", //我的
		Secret: "02AE5E8DAA50D40ABC0EB4E5882D8EB6",     // 我的
		// Host:            "api.huobi.de.com",
		Category:        "ok",
		BaseCurrency:    "LTC",
		QuoteCurrency:   "USDT",
		AmountPrecision: 4,
		PricePrecision:  2,
	}
	ex := NewEx(&ok)
	b, m, coin := ex.GetAccount()
	fmt.Println(b, m, coin)
}
