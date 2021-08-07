package grid

import (
	"fmt"
	"testing"
	model "zmyjobs/corn/models"
	// "github.com/nntaoli-project/goex"
)

// bian Test api mhRYW9GrdeEZ6nNKdZsoGYg4S3DlvrHNxxTqNaP60m3SDz09TwXe5rGDw7YrEVyp secretKey lkUxg31nEBExSprEXX3wRgcGIte30UaYxZmfSW0b31sClkVa0hjrtgCEV5AsxJXB
// bian my api cIiIZnQ8L77acyTkSAH6je0rDAZGoFcoHSlMHaWYUNDjJhKNtu0Gb8nR9MjLSaws secretKey dnc7LwiB1Vgy6zDOqwuqodIxL8FwltVlxhfEVLvrgmfozezrW9JvnHStpmB4Lymx

func TestGetMoney(t *testing.T) {
	fmt.Println("testing start .....")
	huobiSymbol := model.SymbolCategory{
		// Key:    "405a3c54-5dce3618-7yngd7gh5g-326cc", // 不是我的
		// Secret: "4e59ae57-01e1c38c-cb63e660-d273b",

		Key:    "7b5d41cb-8f7e2626-h6n2d4f5gh-c2d91", // 李彬彬
		Secret: "f09fba02-a5c1b946-d2b57c4e-04335",

		// Key:             "8c892879-hrf5gdfghe-4332a67f-a0851", //我的
		// Secret:          "5af10cd5-ca04c723-6a621d2e-32c76",   // 我的
		Host:            "api.huobi.de.com",
		BaseCurrency:    "ETH",
		QuoteCurrency:   "USDT",
		AmountPrecision: 4,
		PricePrecision:  2,
	}

	// bian := model.SymbolCategory{
	// 	Key:    "l6OXxzbfFUSkCFKTrmAPLw1LYpL0RoKwdDw8ASOVA51qBXUSWgn7WU1kZr8vQ2Qk",
	// 	Secret: "EBkFBeqbkjw9woUGs5QnLg1u2FeK4OjMOk0i4rhOzHYnZbavfj4opULuWt42m3kR",
	// 	// Host:            "https://fapi.binace.com",
	// 	BaseCurrency:    "USDT",
	// 	QuoteCurrency:   "DOGE",
	// 	AmountPrecision: 2,
	// 	PricePrecision:  6,
	// 	Category:        "币安",
	// 	Label:           "u20",
	// }

	// 账户测试
	ex := NewEx(&huobiSymbol)
	b, m, coin := ex.GetAccount()
	fmt.Println(b, m, coin)
	// // 价格测试
	// price, err := ex.GetPrice()
	// fmt.Println(price, err)
	// 交易测试
	// price = price.Sub(price.Mul(decimal.NewFromFloat(0.001))).Round(ex.symbol.PricePrecision)
	// price := decimal.NewFromFloat(30000.11).Round(huobi.PricePrecision)
	// price := decimal.NewFromFloat(32001.11)
	// 买入
	// amount := decimal.NewFromFloat(0.0032)

	// c, _ := huobi.New(huobi.Config{Host: huobiSymbol.Host, Label: huobiSymbol.Label, SecretKey: huobiSymbol.Secret, AccessKey: huobiSymbol.Key})
	// n, err := c.PlaceOrder("sell-market", "ethusdt", "test", price, amount)
	// fmt.Println(n, err)
	// .Round(ex.symbol.AmountPrecision)
	// fmt.Println(amount)
	// cliId, orderId, err := ex.Exchanges(amount, price, SellL)
	// fmt.Println(cliId, err, orderId, amount, price)

	// 查找
	// b, c, o := ex.SearchOrder("338257219234466")
	// fmt.Println(b, c, o)

	// 撤单
	b = ex.CancelOrder("338257303905840")
	fmt.Println(b)

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
	// u := model.User{
	// 	Symbol: `{"Category":"币安","Symbol":"dogeusdt","AmountPrecision":2,
	// 		"PricePrecision":6,"Label":"goex-币安-dogeusdt-33","Key":"cIiIZnQ8L77acyTkSAH6je0rDAZGoFcoHSlMHaWYUNDjJhKNtu0Gb8nR9MjLSaws",
	// 		"Secret":"dnc7LwiB1Vgy6zDOqwuqodIxL8FwltVlxhfEVLvrgmfozezrW9JvnHStpmB4Lymx","Host":"","MinTotal":"5","MinAmount":"1",
	// 		"BaseCurrency":"doge","QuoteCurrency":"usdt","future":true}`,
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
	// cli := NewEx(&symbol)
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
