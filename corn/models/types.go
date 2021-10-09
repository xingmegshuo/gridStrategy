package model

import (
	"zmyjobs/goex"

	"github.com/shopspring/decimal"
)

type Operate struct {
	Id float64
	Op int // 1 全部卖出 2 立即补仓 3. 停止买入 4. 恢复买入
}

var OperateCh = make(chan Operate) // 操作管道

// SymbolCategory 交易对参数
type SymbolCategory struct {
	Category        string          // 平台
	Pashare         string          // okex 自定义
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

type CategorySymbols struct {
	BaseCurrency    string          // 交易币种
	QuoteCurrency   string          // 交易基础币种
	AmountPrecision int32           // 精度
	PricePrecision  int32           // 价格精度
	MinTotal        decimal.Decimal // 最小交易钱
	MinAmount       decimal.Decimal // 最小交易数量
	CtVal           float64         // ok 合约张数价值
}

// Args 策略输入参数
type Args struct {
	CoinId       interface{}
	FirstBuy     float64     // 首单
	FirstDouble  bool        // 首单加倍
	Price        float64     // 价格
	IsChange     bool        // 是否为固定加仓金额
	Rate         float64     // 补仓比例
	MinBuy       float64     // 最少买入
	IsAdd        bool        // 是否固定增幅
	AddRate      float64     // 补仓增幅
	NeedMoney    float64     // 需要NeedMoney
	MinPrice     float64     // 最小价格
	Callback     float64     // 回降
	Reduce       float64     // 回调
	Stop         float64     // 止盈
	AddMoney     float64     // 补仓金额
	Decline      float64     // 跌幅比例
	Out          bool        // 出局方式
	OrderType    int64       // 1 限价 2 市价
	IsLimit      bool        // 是否限高
	LimitHigh    float64     // 限高价格
	StrategyType int64       // 1Bi乘方限 2Bi多元限 3Bi乘方市 4Bi多元市 5Bi高频市
	Crile        float64     // 是否循环  1单次,2循环,3做多,4做空,5 单次做多,6单次做空
	OneBuy       bool        // 一键补仓
	AllSell      bool        // 一键平仓
	StopBuy      bool        // 停止买入
	IsHand       bool        // 手动参数 false 自动参数true
	Level        interface{} // 杠杆倍数
	StopFlow     bool        // 停止跟随
	StopEnd      float64     // 止损
}

// 策略预览内容
type Grid struct {
	Id         int
	Price      decimal.Decimal // 价格
	AmountBuy  decimal.Decimal // 买入数量
	Decline    float64         // 跌幅
	AmountSell decimal.Decimal // 卖出数量
	TotalBuy   decimal.Decimal // 投入金额
	Mesure     decimal.Decimal // b本位计量单位张
	Order      string          // 订单id

}

// coin
type Coin struct {
	Id         interface{} `json:"id"`
	CategoryId int         `json:"category_id"` // 平台分类CategoryId
	Name       string      `json:"name"`
	EnName     string      `json:"en_name"`
	CoinName   string      `json:"coin_name"`
	Price      float64     `json:"price"`
	PriceUsd   float64     `json:"price_usd"`
	Raf        float64     `json:"raf"`
	DayAmount  float64     `json:"day_amount"`
	Img        string      `json:"img"`
	Type       string      `json:"type"`
	Enable     int         `json:"enble"`
	CoinType   int         `json:"coin_type"`
	CreateTime int         `json:"create_time"`
}

var (
	a = goex.SWAP_CONTRACT      // 币本位合约
	c = goex.SWAP_USDT_CONTRACT // u 本位合约
)
