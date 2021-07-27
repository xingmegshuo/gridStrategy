package model

import (
    "encoding/json"

    "github.com/shopspring/decimal"
)

var BuyCh = make(chan int)  // 立即补仓
var SellCh = make(chan int) // 立即平仓

// SymbolCategory 交易对参数
type SymbolCategory struct {
    Category        string
    Symbol          string
    AmountPrecision int32
    PricePrecision  int32
    Label           string
    Key             string
    Secret          string
    Host            string
    MinTotal        decimal.Decimal
    MinAmount       decimal.Decimal
    BaseCurrency    string
    QuoteCurrency   string
}

// Args 策略输入参数
type Args struct {
    FirstBuy     float64 // 首单
    FirstDouble  bool    // 首单加倍
    Price        float64 // 价格
    IsChange     bool    // 是否为固定加仓金额
    Rate         float64 // 补仓比例
    MinBuy       float64 // 最少买入
    AddRate      float64 // 补仓增幅
    NeedMoney    float64 // 需要NeedMoney
    MinPrice     float64 // 最小价格
    Callback     float64 // 回降
    Reduce       float64 // 回调
    Stop         float64 // 止盈
    AddMoney     float64 // 补仓金额
    Decline      float64 // 跌幅比例
    Out          bool    // 出局方式
    OrderType    int64   // 市价/限价
    IsLimit      bool    // 是否限高
    LimitHigh    float64 // 限高价格
    StrategyType int64   // 1Bi乘方限 2Bi多元限 3Bi乘方市 4Bi多元市 5Bi高频市
    Crile        bool    // 是否循环
    OneBuy       bool    // 一键补仓
    AllSell      bool    // 一键平仓
    StopBuy      bool    // 停止买入
}

// 策略预览内容
type Grid struct {
    Id         int
    Price      decimal.Decimal // 价格
    AmountBuy  decimal.Decimal // 买入数量
    Decline    float64         // 跌幅
    AmountSell decimal.Decimal // 卖出数量
    TotalBuy   decimal.Decimal // money
    Order      uint64          // 订单id
}

func StringArg(data string) (a Args) {
    _ = json.Unmarshal([]byte(data), &a)
    return
}

func ArgString(a *Args) string {
    s, _ := json.Marshal(&a)
    return string(s)
}

func StringSymobol(data string) (a SymbolCategory) {
    _ = json.Unmarshal([]byte(data), &a)
    return
}
