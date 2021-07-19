/***************************
@File        : t.go
@Time        : 2021/07/19 13:28:21
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package main

import (
	"context"
	"fmt"
	"log"
	"time"
	"zmyjobs/exchange/huobi"

	"github.com/huobirdcenter/huobi_golang/pkg/model/order"
)

func web(ctx context.Context) {
	log.Println("i am start")
	run := 0
	symbol := "btcusdt"
	h, _ := huobi.New(huobi.Config{Host: "api.huobi.de.com", Label: "none", SecretKey: "f09fba02-a5c1b946-d2b57c4e-04335", AccessKey: "7b5d41cb-8f7e2626-h6n2d4f5gh-c2d91"})
	clientId := fmt.Sprintf("%d", time.Now().Unix())

	for {
		select {
		case <-ctx.Done():
			log.Println("协程2中的websocket被暂停")
			return
		default:
		}
		for i := 0; i < 1; i++ {
			if run == 0 {
				log.Println("开始.....")
				go h.SubscribeOrder(ctx, symbol, clientId, OrderUpdateHandler)
				go h.SubscribeTradeClear(ctx, symbol, clientId, TradeClearHandler)
				run = 1
			}
		}
	}

}

func main() {
	c := make(chan int)
	b := 0
	go run(c)

	for {
		time.Sleep(time.Minute * 60)
		if b == 0 {
			c <- 1
			log.Println("main 退出")
		}
	}
}

func run(ch chan int) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	run := 0
	c := 0
OuterLoop:
	for {
		time.Sleep(time.Second * 5)
		c++
		select {
		case v := <-ch:
			log.Println(v)
			log.Println("结束了")
			break OuterLoop
		default:
			log.Println("i am working ", c)
		}

		if run == 0 {
			log.Println("begin")
			for i := 0; i < 1; i++ {
				run = 1
				go web(ctx)
			}
		}
	}
}

func OrderUpdateHandler(response interface{}) {
	subOrderResponse, ok := response.(order.SubscribeOrderV2Response)
	if !ok {
		log.Printf("Received unknown response: %v", response)
	}
	//log.Printf("subOrderResponse = %#v", subOrderResponse)
	if subOrderResponse.Action == "sub" {
		if subOrderResponse.IsSuccess() {
			// o := subOrderResponse.Data
			log.Printf("Subscription topic %s successfully", subOrderResponse.Ch)
			// model.RebotUpdateBy(o.ClientOrderId, "成功")
			// model.AsyncData(t.u.ObjectId, t.amount, GetPrice(t.symbol.Symbol), t.hold)
			// if o.OrderId == int64(t.SellOrder) {
			// 	t.over = true
			// }
		} else {
			log.Fatalf("Subscription topic %s error, code: %d, message: %s",
				subOrderResponse.Ch, subOrderResponse.Code, subOrderResponse.Message)
		}
	} else if subOrderResponse.Action == "push" {
		if subOrderResponse.Data == nil {
			log.Printf("SubscribeOrderV2Response has no data: %#v", subOrderResponse)
			return
		}
		o := subOrderResponse.Data
		log.Printf("Order update, event: %s, symbol: %s, type: %s, id: %d, clientId: %s, status: %s",
			o.EventType, o.Symbol, o.Type, o.OrderId, o.ClientOrderId, o.OrderStatus)
		switch o.EventType {
		case "creation":
			log.Printf("order created, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)

		case "cancellation":
			log.Printf("order cancelled, orderId: %d, clientOrderId: %s", o.OrderId, o.ClientOrderId)
		case "trade":
			log.Printf("交易成功 order filled, orderId: %d, clientOrderId: %s, fill type: %s", o.OrderId, o.ClientOrderId, o.OrderStatus)

		default:
			log.Printf("unknown eventType, should never happen, orderId: %d, clientOrderId: %s, eventType: %s",
				o.OrderId, o.ClientOrderId, o.EventType)
		}
	}
}

func TradeClearHandler(response interface{}) {
	subResponse, ok := response.(order.SubscribeTradeClearResponse)
	if ok {
		if subResponse.Action == "sub" {
			log.Println("aaa")
		} else if subResponse.Action == "push" {
			if subResponse.Data == nil {
				log.Printf("SubscribeOrderV2Response has no data: %#v", subResponse)
				return
			}
			o := subResponse.Data
			log.Printf("Order update, symbol: %s, order id: %d, price: %s, volume: %s",
				o.Symbol, o.OrderId, o.TradePrice, o.TradeVolume)
		}
	} else {
		log.Printf("Received unknown response: %v", response)
	}
}
