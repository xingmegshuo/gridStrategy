package grid

import (
	"fmt"
	"testing"
	model "zmyjobs/corn/models"
	// "github.com/nntaoli-project/goex"
)

func TestGetMoney(t *testing.T) {
	fmt.Println("testing start .....")
	// huobi := model.SymbolCategory{
	// 	Key:             "405a3c54-5dce3618-7yngd7gh5g-326cc",
	// 	Secret:          "4e59ae57-01e1c38c-cb63e660-d273b",
	// 	Host:            "https://api.huobi.de.com",
	// 	BaseCurrency:    "USDT",
	// 	QuoteCurrency:   "ETH",
	// 	AmountPrecision: 2,
	// 	PricePrecision:  6,
	// }

	bian := model.SymbolCategory{
		Key:    "l6OXxzbfFUSkCFKTrmAPLw1LYpL0RoKwdDw8ASOVA51qBXUSWgn7WU1kZr8vQ2Qk",
		Secret: "EBkFBeqbkjw9woUGs5QnLg1u2FeK4OjMOk0i4rhOzHYnZbavfj4opULuWt42m3kR",
		// Host:            "https://fapi.binace.com",
		BaseCurrency:    "USDT",
		QuoteCurrency:   "DOGE",
		AmountPrecision: 2,
		PricePrecision:  6,
		Category:        "币安",
	}

	ex := NewEx(&bian)
	ex.GetAccount()
	ex.GetPrice()
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

}
