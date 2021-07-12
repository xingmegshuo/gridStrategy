/***************************
@File        : log.go
@Time        : 2021/07/09 17:37:42
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 交易日志收集
****************************/

package model

import "time"

// TradeLog 交易记录
type TradeLog struct {
	Time         time.Duration // 交易时间
	OrderId      int64         // 订单id
	Symbol       string        // 交易对
	Types        string        // 交易方式
	OrderType    string        // 买入或者卖出
	Price        float64       // 交易价格
	Amount       float64       // 数量
	RealAmount   float64       // 成交量
	RealAvgPrice float64       // 成交均价
	Status       string        // 状态
	Money        float64       // 资金 花费或增加
	Tip          float64       // 交易所扣除小费
}

