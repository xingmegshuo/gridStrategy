package grid

import (
	"fmt"
	"testing"
	model "zmyjobs/models"
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
	// ex.GetPrice()
}
