package huobi

import (
	"time"

	"github.com/shopspring/decimal"
)

var (
	PricePrecision = map[string]int{
		"btcusdt": 2,
	}
	AmountPrecision = map[string]int{
		"btcusdt": 5,
	}
	MinAmount = map[string]float64{
		"btcusdt": 0.0001,
	}
	MinTotal = map[string]int{
		"btcusdt": 5,
	}
)

type HuobiBalance struct {
	Label    string
	BTC      float64 `gorm:"Column:btc"`
	USDT     float64 `gorm:"Column:usdt"`
	HT       float64 `gorm:"Column:ht"`
	BTCPrice float64 `gorm:"Column:btc_price"`
	HTPrice  float64 `gorm:"Column:ht_price"`
	Time     time.Time
}

type Trade struct {
	Symbol    string
	OrderId   int64
	OrderSide string
	OrderType string
	Aggressor bool

	Id     int64           // trade id
	Time   int64           // trade time
	Price  decimal.Decimal // trade price
	Volume decimal.Decimal // trade volume

	TransactFee       decimal.Decimal //手续费
	FeeDeduct         decimal.Decimal
	FeeDeductCurrency string

	ClientOrder string
}
