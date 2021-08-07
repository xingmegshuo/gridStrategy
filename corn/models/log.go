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

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

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
	HoldNum      float64 // 币种数量
	HoldMoney    float64 // 交易美金
	PayMoney     float64 // 投入金额
	AddRate      float64 // 当前补仓比例
	TransactFee  float64 // 手续费
	Status       string  // 订单状态
	BuyOrSell    string  // 买入还是卖出
	AccountMoney float64 // 账户余额
	Category     string  // 平台
	Custom       uint    // 用户
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

func RebotUpdateBy(orderId string, price decimal.Decimal, hold decimal.Decimal,
	transactFee decimal.Decimal, hold_m decimal.Decimal, m decimal.Decimal, status string, cli string) {
	// log.Println(orderId, "------订单成功")
	money, _ := m.Float64()
	DB.Table("rebot_logs").Where("order_id = ?", orderId).Update("status", status).Update("account_money", money).Update("price", price).
		Update("hold_num", hold).Update("transact_fee", transactFee).Update("hold_money", price.Mul(hold).Sub(transactFee)).
		Update("pay_money", hold_m).Update("order_id", cli)
	var r RebotLog
	DB.Raw("select * from rebot_logs where `order_id` = ?", cli).Scan(&r)
	AddModelLog(&r, money)
}

// asyncData 同步数据
func AsyncData(id interface{}, amount interface{}, price interface{}, money interface{}, num interface{}) {
	//  同步当前task_order
	var data = map[string]interface{}{}
	// UserDB.Raw("select * from db_task_order where `id` = ?", id).Scan(&data)
	log.Println("同步数据:", amount, price, money, num)
	data["hold_num"] = amount
	data["hold_price"] = price
	data["hold_amount"] = money // 持仓金额
	data["current_num"] = num   // 当前单数
	if num == 0 {
		UserDB.Exec("update db_task_order set current_num = 0 where id = ? ", id)
	}
	UpdateOrder(id, data)
}

// PreView 预览策略
func PreView(id interface{}, strategy string) {
	var data = map[string]interface{}{}
	data["grids"] = strategy
	UpdateOrder(id, data)
}

// StrategyError 策略配置错误和运行错误
func StrategyError(id interface{}, e string) {
	var data = map[string]interface{}{}
	data["status"] = 2
	data["error_msg"] = e
	UpdateOrder(id, data)
}

// AddRun 运行次数增加
func AddRun(id interface{}, b interface{}) {
	var data = map[string]interface{}{
		"run_count":   b,
		"current_num": 1,
		"status":      1,
		// "total_profit": b, // 订单盈利增加
	}
	log.Println("修改运行-----")
	UpdateOrder(id, data)
}

// RunOver 运行完成
func RunOver(id float64, b float64) {
	var data = map[string]interface{}{
		"status": 2,
	}
	if b > 0 {
		data["total_profit"] = b
		GotMoney(b, id)
	}
	log.Println("修改盈利-----", id)
	UpdateOrder(id, data)
}

// OneSell 平仓执行
func OneSell(id interface{}) {
	var data = map[string]interface{}{}
	data["one_sell"] = 1
	UpdateOrder(id, data)
}

// OneBuy 补仓执行
func OneBuy(id interface{}) {
	var data = map[string]interface{}{}
	data["one_buy"] = 1
	UpdateOrder(id, data)
}

// 停止交易
func OneStop(id interface{}) {
	var data = map[string]interface{}{}
	data["stop_buy"] = 1
	UpdateOrder(id, data)
}

func UpdateOrder(id interface{}, data map[string]interface{}) {
	UserDB.Table("db_task_order").Where("id = ?", id).Updates(data)
}

// AddModelLog 增加日志
func AddModelLog(r *RebotLog, m float64) {
	log.Println("add trade-----", r.Price, r.HoldMoney)
	var data = map[string]interface{}{}
	var coin = map[string]interface{}{}
	var mes string
	UserDB.Raw("select id,name from db_task_coin where en_name like ?", r.GetCoin).Scan(&coin)
	data["order_sn"] = r.OrderId // 订单号
	data["category_id"] = 2      // 平台
	data["order_id"] = r.UserID  //机器人
	data["buyer_id"] = r.Custom  // 会员
	if r.BuyOrSell == "买入" {
		data["type"] = 0 // 买入还是卖出
		if r.AddNum+1 == 1 {
			mes = fmt.Sprintf("首单买入,开仓---")
		} else {
			mes = fmt.Sprintf("当前第%d买入,跌幅:%f", r.AddNum+1, r.AddRate) + "%"
		}
	} else {
		data["type"] = 1
		if r.AddNum == 0 {
			mes = fmt.Sprintf("清仓操作----,盈利%f", r.AddRate) + "%"
		} else {
			mes = fmt.Sprintf("当前第%d单卖出,盈利:%f", r.AddNum, r.AddRate) + "%"
		}
	}
	data["remark"] = mes
	data["jy_coin_id"] = 1
	data["js_coin_id"] = 1
	data["coin_name"] = "币币交易"                    // 币种名字
	data["task_coin_id"] = coin["id"]             // 币种id
	data["price"] = decimal.NewFromFloat(r.Price) // 价格
	data["num"] = decimal.NewFromFloat(r.HoldNum) // 持仓数量
	data["amount"] = r.HoldMoney                  // 成交金额
	data["handing_fee"] = r.TransactFee           //手续费
	data["status"] = 1                            // 1启用
	data["left_amount"] = decimal.NewFromFloat(m) // 账户余额
	data["extra"] = ""
	data["create_time"] = time.Now().Unix()
	data["update_time"] = time.Now().Unix()
	UserDB.Table("db_task_order_profit").Create(&data)
	// data["description"] = "手动策略"                  // 策略类型
	// data["order_number"] = r.AddNum                            // 当前第几单
}

// GotMoney 盈利分红
func GotMoney(money float64, uId float64) {
	realMoney := money * 0.2 // 分红盈利
	log.Println("盈利金额:", money, "账户余额:", GetAccount(uId))
	if money > 0 && GetAccount(uId) > realMoney { // 盈利
		var (
			u         = map[string]interface{}{}
			thisLevel uint8
		)
		tx := UserDB                                                                                                                               // 使用事务
		tx.Raw("select `id`,`profit_min_amount`,`team_min_amount`,`level`,`inviter_id`,`team_number` from db_customer where id = ?", uId).Scan(&u) // 获取用户
		log.Println(fmt.Sprintf("分红金额:%v----用户id:%v", realMoney, u["id"]))
		baseLevel := u["level"].(uint8)
		// 修改盈利
		ChangeAmount(money, &u, tx, true)

		// 扣除分红金额，写日志
		ownLog := &AmountLog{
			FlowType:       62,
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
		log.Println(fmt.Sprintf("之前账户余额:%v----之后账户余额:%v---vip等级:%v", ownLog.BeforeAmount, ownLog.AfterAmount, baseLevel))
		ownLog.Write(UserDB)
		tx.Table("db_customer").Where("id = ? ", uId).Update("meal_amount", ownLog.AfterAmount)

		// 合伙人

		friends := ParseStringFloat(fmt.Sprintf("%.2f", realMoney*0.8*0.2))

		// 级差分红
		levelMoney := ParseStringFloat(fmt.Sprintf("%.2f", realMoney*0.8*0.8))
		log.Println("级差分红金额:", levelMoney, "合伙人分红:", friends, "平台收入:", realMoney*0.2)

		log.Println("--------------")
		f := true
		after := levelMoney
		for {
			var myMoney float64
			if u["inviter_id"].(uint32) > 0 {
				time.Sleep(time.Second)
				tx.Raw("select `id`,`team_amount`,`team_min_amount`,`level`,`inviter_id`,`team_number` from db_customer where id = ?", u["inviter_id"]).Scan(&u) // 获取用户
				// ChangeAmount(money, &u, tx, true)
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

				if thisLevel > baseLevel && after > 0 {
					// baseLevel = thisLevel // 上级的vip等级，下次分红vip必须大于此等级
					if thisLevel == 2 {
						// l.Println("我分25%")
						myMoney = levelMoney * 0.25
					}
					if thisLevel == 3 {
						// l.Println("我要30%")
						if thisLevel-baseLevel == 1 {
							myMoney = levelMoney * 0.05
						} else {
							myMoney = levelMoney * 0.3
						}
					}
					if thisLevel == 4 {
						// l.Println("我要40%")
						if thisLevel-baseLevel == 1 {
							myMoney = levelMoney * (0.1)
						} else if thisLevel-baseLevel == 2 {
							myMoney = levelMoney * 0.15
						} else {
							myMoney = levelMoney * 0.4
						}
					}
					if thisLevel == 5 {
						// l.Println("我要50%")
						if thisLevel-baseLevel == 1 {
							myMoney = levelMoney * (0.1)
						} else if thisLevel-baseLevel == 2 {
							myMoney = levelMoney * 0.2
						} else if thisLevel-baseLevel == 3 {
							myMoney = levelMoney * 0.25
						} else {
							myMoney = levelMoney * 0.5
						}
					}
					if thisLevel == 6 {
						log.Println(thisLevel, baseLevel)
						// l.Println("我要60%")
						if f {
							if thisLevel-baseLevel == 1 {
								myMoney = levelMoney * (0.1)
							} else if thisLevel-baseLevel == 2 {
								myMoney = levelMoney * 0.2
							} else if thisLevel-baseLevel == 3 {
								myMoney = levelMoney * 0.3
							} else if thisLevel-baseLevel == 4 {
								myMoney = levelMoney * 0.35
							} else {
								myMoney = levelMoney * 0.6
							}
							f = false
						} else {
							myMoney = levelMoney * 0.06 //平级
						}
					}
					log.Println(fmt.Sprintf("分红金额:%2f---用户:%v---我的vip:%v", myMoney, u["id"], thisLevel))

					// levelMoney -= myMoney
					thisLog.Amount = myMoney
					thisLog.AfterAmount = thisLog.BeforeAmount + myMoney
					log.Println(fmt.Sprintf("之前余额:%v---之后余额:%v----用户:%v", thisLog.BeforeAmount, thisLog.AfterAmount, u["id"]))
					log.Println("--------------")
					// thisLog.Write(UserDB)
					// tx.Table("db_coin_amount").Where("customer_id = ? and coin_id = 2", thisLog.CustomerId).Update("amount", thisLog.AfterAmount)
					if myMoney > 0 {
						baseLevel = thisLevel
						after -= myMoney
					}
				}
			} else {
				break
			}
		}
		log.Println(fmt.Sprintf("级差分红剩余金额:%v--- 合伙人分红剩余:%v---- 平台收入:%2f", after, friends, realMoney*0.2+after+friends))
	}
}

// ChangeAmount 修改账户业绩
func ChangeAmount(money float64, u *map[string]interface{}, db *gorm.DB, b bool) {
	var grade = map[string]interface{}{}
	// 修改业绩
	if b {
		grade["profit_min_amount"] = ParseStringFloat((*u)["profit_min_amount"].(string)) + money
	}
	log.Println(fmt.Sprintf("业绩更新:%+v,用户:%v", grade, (*u)["id"]))
	db.Table("db_customer").Where("id = ?", (*u)["id"]).Updates(&grade)
}

// Write 余额变动日志写入
func (amount *AmountLog) Write(db *gorm.DB) {
	db.Table("db_flow_coin_amount").Create(&amount)
}

// ParseStringFloat 字符串转浮点型
func ParseStringFloat(str string) float64 {
	floatValue, _ := strconv.ParseFloat(str, 64)
	return floatValue
}

// DeleteRebotLog 删除交易记录
func DeleteRebotLog(orderId string) {
	DB.Exec("delete from rebot_logs where order_id = ? ", orderId)
}

// LogStrategy 卖出盈利日志
func LogStrategy(name interface{}, coin_name interface{}, order interface{}, member interface{}, amount interface{}, price interface{}, isHand bool, money interface{}) {
	log.Println("盈利日志", name, member)
	var (
		data     = map[string]interface{}{}
		categroy = map[string]interface{}{}
		coin     = map[string]interface{}{}
	)
	UserDB.Raw("select id from db_task_category where `name` like ?", "火币").Scan(&categroy)
	UserDB.Raw("select id from db_task_coin where `en_name` like ?", "doge").Scan(&coin)
	// log.Println(categroy, coin, c, d)
	data["category_id"] = categroy["id"]
	data["coin_id"] = coin["id"]
	data["order_id"] = order
	data["member_id"] = member
	data["coin_name"] = coin_name
	data["order_amount"] = amount
	data["order_price"] = price
	data["av_amount"] = money
	data["type"] = 2
	data["member_name"] = ""
	data["description"] = "手动策略"
	if isHand {
		data["description"] = "自动策略"
	}
	data["status"] = 1
	data["create_time"] = time.Now().Unix()
	data["update_time"] = time.Now().Unix()
	UserDB.Table("db_task_order_log").Create(&data)
}
