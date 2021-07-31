package grid

import (
	"fmt"
	"testing"
	model "zmyjobs/models"
	// "github.com/nntaoli-project/goex"
)

func TestGetMoney(t *testing.T) {
	fmt.Println("testing start .....")
	symbol := model.SymbolCategory{
		Key:             "405a3c54-5dce3618-7yngd7gh5g-326cc",
		Secret:          "4e59ae57-01e1c38c-cb63e660-d273b",
		Host:            "https://api.huobi.de.com",
		BaseCurrency:    "USDT",
		QuoteCurrency:   "ETH",
		AmountPrecision: 2,
		PricePrecision:  6,
	}
	ex := NewEx(&symbol)
	ex.GetAccount()
	// goex.LimitOrderOptionalParameter
	// c, e := ex.Ex.LimitSell("0.025", "2400.00", ex.MakePair(), 332301627399393)
	// fmt.Println(c, e)

	b, h := ex.Ex.GetOneOrder("332313548499641", ex.MakePair())
	fmt.Println(b.Cid, b.Status, h)
	// b, e := ex.Ex.CancelOrder("332301627399393", ex.MakePair())
	// fmt.Println(b, e)
	// ex.GetPrice()

	// b, h := ex.Ex.MarketSell("0.025", "2400.00", ex.MakePair())
	// fmt.Println(b, h)

}
