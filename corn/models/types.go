package model

import (
	"zmyjobs/goex"

	"github.com/shopspring/decimal"
)

type Operate struct {
	Id float64
	Op int // 1 全部卖出 2 立即补仓 3. 停止买入 4. 恢复买入
}

var OperateCh = make(chan Operate) // 立即补仓

// SymbolCategory 交易对参数
type SymbolCategory struct {
	Category        string          // 平台
	Symbol          string          // 交易对
	AmountPrecision int32           // 精度
	PricePrecision  int32           // 价格精度
	Label           string          // label
	Key             string          // 秘钥
	Secret          string          // 秘钥
	Host            string          // 连接
	MinTotal        decimal.Decimal // 最小交易钱
	MinAmount       decimal.Decimal // 最小交易数量
	BaseCurrency    string          // 交易币种
	QuoteCurrency   string          // 交易基础币种
	Future          bool            // 现货/期货
	Lever           float64         // 杠杆倍数
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
	OrderType    int64   // 1 限价 2 市价
	IsLimit      bool    // 是否限高
	LimitHigh    float64 // 限高价格
	StrategyType int64   // 1Bi乘方限 2Bi多元限 3Bi乘方市 4Bi多元市 5Bi高频市
	Crile        bool    // 是否循环
	OneBuy       bool    // 一键补仓
	AllSell      bool    // 一键平仓
	StopBuy      bool    // 停止买入
	IsHand       bool    // 手动参数 false 自动参数true
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

// 合约
type Future struct {
	// TODO : 合约类型
}

var (
	a = goex.SWAP_CONTRACT // 永续合约
	b = goex.SWAP_USDT     // u本位
	c = goex.SWAP_USDT_CONTRACT
)
