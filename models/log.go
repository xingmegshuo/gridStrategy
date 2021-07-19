/***************************
@File        : log.go
@Time        : 2021/07/09 17:37:42
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 交易日志收集
****************************/

package model

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TradeLog 交易记录
//type TradeLog struct {
//	gorm.Model
//	Time         time.Duration // 交易时间
//	OrderId      int64         // 订单id
//	Symbol       string        // 交易对
//	Types        string        // 交易方式
//	OrderType    string        // 买入或者卖出
//	Price        float64       // 交易价格
//	Amount       float64       // 数量
//	RealAmount   float64       // 成交量
//	RealAvgPrice float64       // 成交均价
//	Status       string        // 状态
//	Money        float64       // 资金  花费或增加
//	Tip          float64       // 交易所扣除小费
//}

// RebotLog 机器人日志
type RebotLog struct {
	gorm.Model
	OrderId      string  `gorm:"index"` // orderid
	BaseCoin     string  // 交易货币
	GetCoin      string  // 买入货币
	Price        float64 // 当前交易价格
	Types        string  // 市价或限价
	UserID       uint    // 哪个任务
	AddNum       int     // 当前补单数量
	HoldNum      float64 // 当前持有币种数量
	HoldMoney    float64 // 当前价值美金
	PayMoney     float64 // 当前投入金额
	AddRate      float64 // 当前补仓比例
	Status       string  // 订单状态
	BuyOrSell    string  // 买入还是卖出
	AccountMoney float64 // 账户余额
}

func (r *RebotLog) New() {
	DB.Create(&r)
}

func RebotUpdateBy(orderId string, m decimal.Decimal, status string) {
	log.Println(orderId)
	// order := strconv.FormatInt(orderId, 10)
	money, _ := m.Float64()
	var r RebotLog
	DB.Raw("select * from rebot_logs where `order_id` = ?", orderId).Scan(&r)
	DB.Table("rebot_logs").Where("id = ?", r.ID).Update("status", status).Update("account_money", money)
	AddModelLog(&r)
}

// Message 发通知
func Message(mes string) {

}

// asyncData 同步数据
func AsyncData(id interface{}, amount interface{}, price interface{}, money interface{}) {
	//  同步当前task_order
	var data = map[string]interface{}{}
	UserDB.Raw("select * from db_task_order where `id` = ?", id).Scan(&data)
	log.Println("同步数据:", amount, price, money)
	data["hold_num"] = amount
	data["hold_price"] = price
	data["hold_amount"] = money
	UserDB.Table("db_task_order").Where("id = ?", id).Updates(&data)
}

// AddModelLog 增加日志
func AddModelLog(r *RebotLog) {
	log.Println("add trade-----", r.Price, r.HoldMoney)
	var data = map[string]interface{}{}
	data["order_sn"] = r.OrderId
	data["category_id"] = 2
	data["order_id"] = r.UserID
	if r.BuyOrSell == "买入" {
		data["type"] = 1
	} else {
		data["type"] = 1
	}
	data["coin_name"] = "币币交易"
	data["task_coin_id"] = 1
	data["remark"] = "none"
	data["jy_coin_id"] = 1
	data["js_coin_id"] = 1
	data["order_type"] = 1
	data["extra"] = "none"
	data["price"] = decimal.NewFromFloat(r.Price)
	data["num"] = decimal.NewFromFloat(r.HoldNum)
	data["amount"] = decimal.NewFromFloat(r.HoldMoney)
	data["left_amount"] = decimal.NewFromFloat(r.AccountMoney)
	data["status"] = 1
	data["create_time"] = time.Now().Unix()
	data["update_time"] = time.Now().Unix()
	UserDB.Table("db_task_order_profit").Create(&data)
}


// GotMoney 盈利分红
func GotMoney(){
	
}