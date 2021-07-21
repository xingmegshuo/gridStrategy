/***************************
@File        : log.go
@Time        : 2021/07/09 17:37:42
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 交易日志收集
****************************/

package model

import (
	"fmt"
	"strconv"
	"time"

	l "log"

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

// AmountLog 余额流水日志
type AmountLog struct {
	FlowType       float64 // 日志类型
	CoinId         float64 // 币种
	FromCustomerId float64 // 来自哪个用户
	Direction      int64   // 支出or 收入
	Amount         float64 // 金额
	BeforeAmount   float64 // 操作前余额
	AfterAmount    float64 // 操作后余额
	Hash           string  // 默认值
	Remark         string  // 说明
	CustomerId     float64 // 账号

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
	AddModelLog(&r, money)
}

// Message 发通知
// func Message(mes string) {

// }

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
func AddModelLog(r *RebotLog, m float64) {
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
	data["left_amount"] = decimal.NewFromFloat(m)
	data["status"] = 1
	data["create_time"] = time.Now().Unix()
	data["update_time"] = time.Now().Unix()
	UserDB.Table("db_task_order_profit").Create(&data)
}

// GotMoney 盈利分红
func GotMoney(money float64, uId float64) {
	if money > 0 && GetAccount(uId) > money { // 盈利
		var (
			u         = map[string]interface{}{}
			thisLevel uint8
		)
		tx := UserDB                                                                                                                         // 使用事务
		realMoney := money * 0.2                                                                                                             // 分红盈利
		tx.Raw("select `id`,`team_amount`,`team_min_amount`,`level`,`inviter_id`,`team_number` from db_customer where id = ?", uId).Scan(&u) // 获取用户

		baseLevel := u["level"].(uint8)
		// 修改业绩
		ChangeAmount(money, &u, tx, false)

		// 扣除分红金额，写日志
		ownLog := &AmountLog{
			FlowType:       58,
			CustomerId:     uId,
			FromCustomerId: uId,
			Direction:      1,
			CoinId:         2,
			Amount:         realMoney,
			BeforeAmount:   GetAccount(uId),
			AfterAmount:    GetAccount(uId) - realMoney,
			Hash:           "000",
			Remark:         "盈利扣款",
		}

		ownLog.Write(UserDB)
		tx.Table("db_coin_amount").Where("customer_id = ? and coin_id = 2", uId).Update("amount", ownLog.AfterAmount)
		// 合伙人
		_ = ParseStringFloat(fmt.Sprintf("%.2f", realMoney*0.8*0.2))

		// 级差分红
		levelMoney := ParseStringFloat(fmt.Sprintf("%.2f", realMoney*0.8*0.8))
		for {
			if u["inviter_id"].(uint32) > 0 {
				time.Sleep(time.Second * 2)
				tx.Raw("select `id`,`team_amount`,`team_min_amount`,`level`,`inviter_id`,`team_number` from db_customer where id = ?", u["inviter_id"]).Scan(&u) // 获取用户
				ChangeAmount(money, &u, tx, true)
				thisLevel = u["level"].(uint8)
				var thisLog = &AmountLog{
					FlowType:       float64(59),
					CoinId:         float64(2),
					Direction:      2,
					Hash:           "",
					Remark:         "级差分红",
					FromCustomerId: uId,
					CustomerId:     float64(u["id"].(int32)),
					BeforeAmount:   GetAccount(float64(u["id"].(int32))),
				}
				if thisLevel >= baseLevel && levelMoney > 0 {
					l.Println("给我分红")
					baseLevel = thisLevel
					var myMoney float64
					if thisLevel == 1 {
						l.Println("我分25%")
						myMoney = levelMoney * 0.25
					}
					if thisLevel == 2 {
						l.Println("我要30%")
						myMoney = levelMoney * 0.3
					}
					if thisLevel == 3 {
						l.Println("我要40%")
						myMoney = levelMoney * 0.4
					}
					if thisLevel == 4 {
						l.Println("我要50%")
						myMoney = levelMoney * 0.5
					}
					if thisLevel == 5 {
						l.Println("我要60%")
						myMoney = levelMoney * 0.6
					}
					levelMoney -= myMoney
					thisLog.Amount = myMoney
					thisLog.AfterAmount = thisLog.BeforeAmount + myMoney
					thisLog.Write(UserDB)
					tx.Table("db_coin_amount").Where("customer_id = ? and coin_id = 2", thisLog.CustomerId).Update("amount", thisLog.AfterAmount)
				}
			} else {
				break
			}
		}
	}
}

// ChangeAmount 修改账户业绩
func ChangeAmount(money float64, u *map[string]interface{}, db *gorm.DB, b bool) {
	var grade = map[string]interface{}{}
	// 修改业绩

	if b {
		grade["team_amount"] = ParseStringFloat((*u)["team_amount"].(string)) + money
	} else {
		grade["team_min_amount"] = money + ParseStringFloat((*u)["team_min_amount"].(string))
	}
	// 购买数量
	t := ParseStringFloat((*u)["team_number"].(string))
	if t >= float64(50) {
		grade["level"] = 1
	}
	if t >= float64(100) {
		grade["level"] = 2
	}
	if t >= float64(500) {
		grade["level"] = 3
	}
	if t >= float64(1000) {
		grade["level"] = 4
	}
	if t >= float64(3000) {
		grade["level"] = 5
	}
	db.Table("db_customer").Where("id = ?", (*u)["id"]).Updates(&grade)
}

// Write 余额变动日志写入
func (amount *AmountLog) Write(db *gorm.DB) {
	c := db.Table("db_flow_coin_amount").Create(&amount)
	l.Println(c)
}

// ParseStringFloat 字符串转浮点型
func ParseStringFloat(str string) float64 {
	floatValue, _ := strconv.ParseFloat(str, 64)
	return floatValue
}
