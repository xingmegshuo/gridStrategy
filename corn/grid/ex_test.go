package grid

import (
	"context"
	"fmt"
	"testing"
	"time"
	model "zmyjobs/corn/models"
	// "github.com/nntaoli-project/goex"
)

func TestGetMoney(t *testing.T) {
	fmt.Println("testing start .....")
	// huobi := model.SymbolCategory{
	// 	// Key:    "405a3c54-5dce3618-7yngd7gh5g-326cc", 不是我的
	// 	// Secret: "4e59ae57-01e1c38c-cb63e660-d273b",
	// 	Key:             "8c892879-hrf5gdfghe-4332a67f-a0851", //我的
	// 	Secret:          "5af10cd5-ca04c723-6a621d2e-32c76",   // 我的
	// 	Host:            "https://api.huobi.de.com",
	// 	BaseCurrency:    "USDT",
	// 	QuoteCurrency:   "ACH",
	// 	AmountPrecision: 4,
	// 	PricePrecision:  6,
	// }

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
	// ex := NewEx(&huobi)
	// b, m, coin := ex.GetAccount()
	// fmt.Println(b, m, coin)
	// // 价格测试
	// price, err := ex.GetPrice()
	// fmt.Println(price, err)
	// 交易测试
	// price = price.Sub(price.Mul(decimal.NewFromFloat(0.001))).Round(ex.symbol.PricePrecision)
	// 买入
	// amount := decimal.NewFromInt(6).Div(price).Round(ex.symbol.AmountPrecision)
	// cliId, orderId, err := ex.Exchanges(amount, price, BuyL)
	// fmt.Println(cliId, err, orderId, amount, price)

	// 查找
	// b, c, o := ex.SearchOrder("335517933772145")
	// fmt.Println(b, c, o)

	// 撤单
	// b = ex.CancelOrder("335517933772145")
	// fmt.Println(b)

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
	u := model.User{
		Symbol: `{"Category":"火币","Symbol":"dogeusdt","AmountPrecision":2,
			"PricePrecision":6,"Label":"goex-火币-dogeusdt-33","Key":"7b5d41cb-8f7e2626-h6n2d4f5gh-c2d91",
			"Secret":"f09fba02-a5c1b946-d2b57c4e-04335","Host":"api.huobi.de.com","MinTotal":"5","MinAmount":"1",
			"BaseCurrency":"doge","QuoteCurrency":"usdt"}`,
		Arg: `{"FirstBuy": 6, "FirstDouble": false, "Price": 0.196562, "IsChange": false, "Rate": 20,
				"MinBuy": 6, "AddRate": 50, "NeedMoney": 0, "MinPrice": 0, "Callback": 0.2, "Reduce": 0.2, "Stop": 5,
				"AddMoney": 0, "Decline": 0.5, "Out": false, "OrderType": 0, "IsLimit": false, "LimitHigh": 0, "StrategyType": 1,
				"Crile": true, "OneBuy": false, "AllSell": false, "StopBuy": false, "IsHand": false}`,
		Strategy: `{"FirstBuy":"6.00000000","NowPrice":"0.00000000","Strategy":1,"Type":2,"add":"0.0000","allSell":1,"callback":"0.20",
			"decline":"0.50","double":1,"frequency":2,"growth":"50.00","high_price":"0.00000000","limit_high":1,"one_buy":1,"order_type":1,
			"rate":"20.00","reduce":"0.20","reset":"20.00","stop":"5.00","stop_buy":1}`,
		Grids: `[{"Id":1,"Price":"0.196562","AmountBuy":"30.52","Decline":0.5,"AmountSell":"0","TotalBuy":"6","Order":0},
			{"Id":2,"Price":"0.195087785","AmountBuy":"36.91","Decline":0.75,"AmountSell":"0","TotalBuy":"7.2","Order":0},
			{"Id":3,"Price":"0.1928764625","AmountBuy":"44.8","Decline":1.125,"AmountSell":"0","TotalBuy":"8.64","Order":0}]`,
		Number: 3,
		Base:   0,
	}
	symbol := model.StringSymobol(u.Symbol)
	cli := NewEx(&symbol)
	for i := 1; i < 11; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		u.ObjectId = int32(i)
		// time.Sleep(time.Second * 2)
		// for i := 1; i < 2; i++ {
		time.Sleep(time.Millisecond * 100)
		go RunEx(ctx, u, cli)
		// }
	}
	for {

	}
}
