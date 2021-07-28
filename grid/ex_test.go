package grid

import (
	"fmt"
	"testing"
	model "zmyjobs/models"
)

func TestGetMoney(t *testing.T) {
	fmt.Println("testing start .....")
	symbol := model.SymbolCategory{Key: "7b5d41cb-8f7e2626-h6n2d4f5gh-c2d91", Secret: "f09fba02-a5c1b946-d2b57c4e-04335", Host: "https://api.huobi.de.com"}
	ex := NewEx(&symbol)
	ex.GetAccount()
}
