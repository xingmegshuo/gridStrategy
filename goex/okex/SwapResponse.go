package okex

/*
 OKEX api result definition
 @author Lingting Fu
 @date 2018-12-27
 @version 1.0.0
*/

type SwapPositionHolding struct {
	LiquidationPrice float64 `json:"liquidation_price , string"`
	Position         float64 `json:"position,string"`
	AvailPosition    float64 `json:"avail_position,string"`
	AvgCost          float64 `json:"avg_cost,string"`
	SettlementPrice  float64 `json:"settlement_price,string"`
	InstrumentId     string  `json:"instId"`
	Leverage         string  `json:"lever"`
	RealizedPnl      float64 `json:"realized_pnl,string"`
	Side             string  `json:"posSide"`
	Timestamp        string  `json:"timestamp"`
	Margin           string  `json:"margin";default:""`
	Amount           string  `json:"pos"`
	Upl              string  `json:"upl"`
}

type BizWarmTips struct {
	Code    string `json:"code"`
	Message string `json:"msg"`
}

type SwapPosition struct {
	BizWarmTips
	Holding []SwapPositionHolding `json:"data"`
}

type SwapAccountInfo struct {
	InstrumentId      string  `json:"instrument_id"`
	Timestamp         string  `json:"timestamp"`
	MarginFrozen      float64 `json:"margin_frozen,string"`
	TotalAvailBalance float64 `json:"total_avail_balance,string"`
	MarginRatio       float64 `json:"margin_ratio,string"`
	RealizedPnl       float64 `json:"realized_pnl,string"`
	UnrealizedPnl     float64 `json:"unrealized_pnl,string"`
	FixedBalance      float64 `json:"fixed_balance,string"`
	Equity            float64 `json:"equity,string"`
	Margin            float64 `json:"margin,string"`
	MarginMode        string  `json:"margin_mode"`
}

type SwapAccounts struct {
	BizWarmTips
	Info []SwapAccountInfo `json:"info"`
}

type BaseSwapOrderResult struct {
	OrderId      string `json:"order_id"`
	ClientOid    string `json:"client_oid"`
	ErrorMessage string `json:"error_message"`
	ErrorCode    string `json:"error_code"`
	Result       string `json:"result"`
}

type SwapOrderResult struct {
	BaseSwapOrderResult
	BizWarmTips
}

type SwapOrdersResult struct {
	BizWarmTips
	OrderInfo []BaseSwapOrderResult `json:"order_info"`
}

type SwapCancelOrderResult struct {
	ErrorMessage string `json:"error_message"`
	ErrorCode    string `json:"error_code"`
	OrderId      string `json:"order_id"`
	Result       bool   `json:"result,string"`
}

type SwapBatchCancelOrderResult struct {
	BizWarmTips
	InstrumentId string   `json:"instrument_id"`
	Ids          []string `json:"ids"`
	Result       string   `json:"result"`
}

type BaseOrderInfo struct {
	InstrumentId string  `json:"instId"`
	OrderId      string  `json:"ordId"`
	ClientOid    string  `json:"clOrdId"`
	Timestamp    string  `json:"fillTime"`
	Price        float64 `json:"px,string"`
	PriceAvg     float64 `json:"avgPx,string"`
	Size         float64 `json:"accFillSz,string"`
	Fee          float64 `json:"fee,string"`
	OrdType      string  `json:"ordType"`
	Slide        string  `json:"side"`
	State        string  `json:"state"`
	PostSlde     string  `json:"posSide"`
	Pnl          float64 `json:"pnl,string"`
	FilledQty    float64 `json:"lever,string"`
	ContractVal  string
	Type         int
	Status       int
}

type SwapOrdersInfo struct {
	BizWarmTips
	OrderInfo []BaseOrderInfo `json:"order_info"`
}

type BaseFillInfo struct {
	InstrumentId string `json:"instrument_id"`
	OrderQty     string `json:"order_qty"`
	TradeId      string `json:"trade_id"`
	Fee          string `json:"fee"`
	OrderId      string `json:"order_id"`
	Timestamp    string `json:"timestamp"`
	Price        string `json:"price"`
	Side         string `json:"side"`
	ExecType     string `json:"exec_type"`
}

type SwapFillsInfo struct {
	BizWarmTips
	FillInfo []*BaseFillInfo `json:"fill_info"`
}

type SwapAccountsSetting struct {
	BizWarmTips
	InstrumentId  string `json:"instrument_id"`
	LongLeverage  string `json:"long_leverage"`
	ShortLeverage string `json:"short_leverage"`
	MarginMode    string `json:"margin_mode"`
}

type BaseLedgerInfo struct {
	InstrumentId string `json:"instrument_id"`
	Fee          string `json:"fee"`
	Timestamp    string `json:"timestamp"`
	Amount       string `json:"amount"`
	LedgerId     string `json:"ledger_id"`
	Type         string `json:"type"`
}

type SwapAccountsLedgerList []BaseLedgerInfo

type BaseInstrumentInfo struct {
	InstrumentId    string `json:"instrument_id"`
	QuoteCurrency   string `json:"quote_currency"`
	TickSize        string `json:"tick_size"`
	ContractVal     string `json:"contract_val"`
	Listing         string `json:"listing"`
	UnderlyingIndex string `json:"underlying_index"`
	Delivery        string `json:"delivery"`
	Coin            string `json:"coin"`
	SizeIncrement   string `json:"size_increment"`
}

type SwapInstrumentList []BaseInstrumentInfo

type BaesDepthInfo []interface{}
type SwapInstrumentDepth struct {
	BizWarmTips
	Timestamp string          `json:"time"`
	Bids      []BaesDepthInfo `json:"bids"`
	Asks      []BaesDepthInfo `json:"asks"`
}

type BaseTickerInfo struct {
	InstrumentId string  `json:"instrument_id"`
	Last         float64 `json:"last,string"`
	Timestamp    string  `json:"timestamp"`
	High24h      float64 `json:"high_24h,string"`
	Volume24h    float64 `json:"volume_24h,string"`
	Low24h       float64 `json:"low_24h,string"`
	BestBid      float64 `json:"best_bid,string"`
	BestAsk      float64 `json:"best_ask,string"`
}

type SwapTickerList []BaseTickerInfo

type BaseTradeInfo struct {
	Timestamp string `json:"timestamp"`
	TradeId   string `json:"trade_id"`
	Side      string `json:"side"`
	Price     string `json:"price"`
	Size      string `json:"size"`
}

type SwapTradeList []BaseTradeInfo

type BaseCandleInfo []interface{}
type SwapCandleList []BaseCandleInfo

type SwapIndexInfo struct {
	BizWarmTips
	InstrumentId string `json:"instrument_id"`
	Index        string `json:"index"`
	Timestamp    string `json:"timestamp"`
}

type SwapRate struct {
	InstrumentId string `json:"instrument_id"`
	Timestamp    string `json:"timestamp"`
	Rate         string `json:"rate"`
}

type BaseInstrumentAmount struct {
	BizWarmTips
	InstrumentId string `json:"instrument_id"`
	Timestamp    string `json:"timestamp"`
	Amount       string `json:"amount"`
}

type SwapOpenInterest BaseInstrumentAmount

type SwapPriceLimit struct {
	BizWarmTips
	InstrumentId string `json:"instrument_id"`
	Lowest       string `json:"lowest"`
	Highest      string `json:"highest"`
	Timestamp    string `json:"timestamp"`
}

type BaseLiquidationInfo struct {
	InstrumentId string `json:"instrument_id"`
	Loss         string `json:"loss"`
	CreatedAt    string `json:"created_at"`
	Type         string `json:"type"`
	Price        string `json:"price"`
	Size         string `json:"size"`
}

type SwapLiquidationList []BaseLiquidationInfo

type SwapAccountHolds BaseInstrumentAmount

type SwapFundingTime struct {
	BizWarmTips
	InstrumentId string `json:"instrument_id"`
	FundingTime  string `json:"funding_time"`
}

type SwapMarkPrice struct {
	BizWarmTips
	InstrumentId string `json:"instrument_id"`
	MarkPrice    string `json:"mark_price"`
	Timestamp    string `json:"timestamp"`
}

type BaseHistoricalFundingRate struct {
	InstrumentId string `json:"instrument_id"`
	InterestRate string `json:"interest_rate"`
	FundingRate  string `json:"funding_rate"`
	FundingTime  string `json:"funding_time"`
	RealizedRate string `json:"realized_rate"`
}

type SwapHistoricalFundingRateList []BaseHistoricalFundingRate

type SwapAccountMoney struct {
	Currency string `json:"ccy"`     // ??????
	CashBal  string `json:"cashBal"` // ??????
	AvailEq  string `json:"availEq"` // ????????????
}

type SwapAccountDetail struct {
	Details []SwapAccountMoney `json:"details"`
}

type SwapAccountBalance struct {
	BizWarmTips
	Info []SwapAccountDetail `json:"data"`
}

type SwapOrderInfo struct {
	ClientId string `json:"clOrdId"` // ??????????????????
	OrdId    string `json:"ordId"`   // ??????????????????
}
