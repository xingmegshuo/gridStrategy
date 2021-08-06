package binance

import (
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"
	goex "zmyjobs/goex"
)

var wallet *Wallet

func init() {
	wallet = NewWallet(&goex.APIConfig{
		HttpClient: &http.Client{
			Transport: &http.Transport{
				Proxy: func(req *http.Request) (*url.URL, error) {
					return url.Parse("socks5://127.0.0.1:1123")
					// return nil, nil
				},
				Dial: (&net.Dialer{
					Timeout: 10 * time.Second,
				}).Dial,
			},
			Timeout: 10 * time.Second,
		},
		// HttpClient:   http.DefaultClient,
		ApiKey:       "cIiIZnQ8L77acyTkSAH6je0rDAZGoFcoHSlMHaWYUNDjJhKNtu0Gb8nR9MjLSaws",
		ApiSecretKey: "dnc7LwiB1Vgy6zDOqwuqodIxL8FwltVlxhfEVLvrgmfozezrW9JvnHStpmB4Lymx",
	})
}

func TestWallet_Transfer(t *testing.T) {
	t.Log(wallet.Transfer(goex.TransferParameter{
		Currency: "USDT",
		From:     goex.SPOT,
		To:       goex.SWAP_USDT,
		Amount:   100,
	}))
}
