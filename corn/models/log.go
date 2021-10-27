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
	CreateTime     int64
}

func (r *RebotLog) New() {
	DB.Create(&r)
}

func RebotUpdateBy(orderId string, price decimal.Decimal, hold decimal.Decimal,
	transactFee decimal.Decimal, hold_m decimal.Decimal, m decimal.Decimal, status string, cli string, f int, coin_id interface{}) {
	// log.Println(orderId, "------订单成功")
	money, _ := m.Float64()
	DB.Table("rebot_logs").Where("order_id = ?", orderId).Update("status", status).Update("account_money", money).Update("price", price).
		Update("hold_num", hold).Update("transact_fee", transactFee).Update("hold_money", price.Mul(hold).Sub(transactFee)).
		Update("pay_money", hold_m).Update("order_id", cli)
	time.Sleep(time.Second * 3)
	var r RebotLog
	DB.Raw("select * from rebot_logs where `order_id` = ?", cli).Scan(&r)
	AddModelLog(&r, money, f, coin_id)
}

// asyncData 同步数据
func AsyncData(id interface{}, amount interface{}, price interface{}, money interface{}, num interface{}) {
	//  同步当前task_order
	var data = map[string]interface{}{}
	// UserDB.Raw("select * from db_task_order where `id` = ?", id).Scan(&data)
	log.Printf("同步数据-- 数量:%v;价格:%v;钱:%v;单数:%v", amount, price, money, num)
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
func RunOver(id interface{}, b interface{}, orderId interface{}, from interface{}, isHand bool, status interface{}) {
	// fmt.Println(isHand)
	old := GetOldAmount(orderId)
	var data = map[string]interface{}{}
	data["status"] = status
	// 预充值金额减去分红金额小于5 结束任务
	if GetAccount(id.(float64))-b.(float64)*0.24 < 5 {
		data["status"] = 2 // 循环中不重新开启直接结束
	}
	if b.(float64) > 0 {
		log.Println("修改盈利:", orderId, "盈利金额:", b, "之前盈利金额:", old, "现在盈利金额:", old+b.(float64), isHand)
		data["total_profit"] = b.(float64) + old
		CentsUser(b.(float64), id.(float64), from)
	}
	UpdateOrder(orderId, data)
}

// 获取策略之前盈利
func GetOldAmount(id interface{}) (m float64) {
	UserDB.Raw("select `total_profit` from db_task_order where id = ? ", id).Scan(&m)
	log.Println("之前盈利", m)
	return
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
func AddModelLog(r *RebotLog, m float64, f int, coin_id interface{}) {
	log.Printf("写入日志 价格:%v;持仓:%v;查询币种条件:%v;coin_type:%v", r.Price, r.HoldMoney, r.GetCoin, f)
	var data = map[string]interface{}{}
	// var coin = map[string]interface{}{}
	var mes string
	// UserDB.Raw("select id,name from db_task_coin where en_name like ? and coin_type = ?", r.GetCoin, f).Scan(&coin)
	// log.Printf("获取的coin内容%v", coin)
	data["order_sn"] = r.OrderId // 订单号
	data["category_id"] = 2      // 平台
	data["order_id"] = r.UserID  //机器人
	data["buyer_id"] = r.Custom  // 会员
	if r.BuyOrSell == "买入" {
		data["type"] = 0 // 买入还是卖出
		if r.AddNum+1 == 1 {
			mes = fmt.Sprintf("首单买入,开仓---")
		} else {
			mes = fmt.Sprintf("当前第%d买入,跌幅:%.2f", r.AddNum+1, r.AddRate) + "%"
		}
	} else {
		data["type"] = 1
		if r.AddNum == 0 {
			mes = fmt.Sprintf("清仓---,盈利%.2f", r.AddRate) + "%"
		} else {
			mes = fmt.Sprintf("当前第%d单卖出,盈利:%.2f", r.AddNum, r.AddRate) + "%"
		}
	}
	data["remark"] = mes + fmt.Sprintf(",您的交易所账户可交易余额:%2f", m)
	data["jy_coin_id"] = 1
	data["js_coin_id"] = 1
	data["coin_name"] = "币币交易"                    // 币种名字
	data["task_coin_id"] = coin_id                // 币种id
	data["price"] = decimal.NewFromFloat(r.Price) // 价格
	data["num"] = decimal.NewFromFloat(r.HoldNum) // 持仓数量
	data["amount"] = r.HoldMoney                  // 成交金额
	data["handing_fee"] = r.TransactFee           //手续费
	data["status"] = 1                            // 1启用
	data["left_amount"] = decimal.NewFromFloat(m) // 账户余额
	data["extra"] = "手动策略"                        // 策略类型
	data["create_time"] = time.Now().Unix()
	data["update_time"] = time.Now().Unix()
	UserDB.Table("db_task_order_profit").Create(&data)
	// data["description"] = "手动策略"                  // 策略类型
	// data["order_number"] = r.AddNum                  // 当前第几单
}

// GotMoney 盈利分红
func GotMoney(money float64, uId float64, from interface{}) {
	t := GetAccount(uId)
	realMoney := money * 0.2 // 分红盈利
	log.Println("盈利金额:", money, "账户余额:", t)
	if money > 0 && t > realMoney { // 盈利
		var (
			u = map[string]interface{}{}
		)
		tx := UserDB                                                                                                                                // 使用事务
		tx.Raw("select `id`,`profit_mine_amount`,`team_min_amount`,`level`,`inviter_id`,`team_number` from db_customer where id = ?", uId).Scan(&u) // 获取用户
		log.Println(fmt.Sprintf("分红金额:%v----用户id:%v", realMoney, u["id"]))
		// 修改盈利
		ChangeAmount(money, &u, tx, true)
		// 扣除分红金额，写日志
		ownLog := &AmountLog{
			FlowType:       62,
			CustomerId:     uId,
			FromCustomerId: float64(from.(int64)),
			Direction:      1,
			CoinId:         2,
			Amount:         realMoney,
			BeforeAmount:   t,
			AfterAmount:    t - realMoney,
			Hash:           "000",
			Remark:         "盈利扣款",
			CreateTime:     time.Now().Unix(),
		}
		log.Println(fmt.Sprintf("之前预充值余额:%v----之后预充值余额:%v", ownLog.BeforeAmount, ownLog.AfterAmount))
		ownLog.Write(UserDB)
		tx.Table("db_customer").Where("id = ? ", uId).Update("meal_amount", ownLog.AfterAmount)

		// 直推奖励
		referralReward := ParseStringFloat(fmt.Sprintf("%.8f", realMoney*0.8*0.6))

		// 合伙人
		partnerReward := ParseStringFloat(fmt.Sprintf("%.8f", realMoney*0.8*0.1))

		// 创始人
		founderReward := ParseStringFloat(fmt.Sprintf("%.8f", realMoney*0.8*0.2))

		log.Println("直推分红金额:", referralReward, "合伙人分红:", partnerReward, "创始人:", founderReward, "平台收入:", realMoney*0.2)

		count := 0
		after := realMoney * 0.8
		for {
			var myMoney float64
			if u["inviter_id"].(uint32) > 0 {
				tx.Raw("select `id`,`team_amount`,`profit_mine_amount`,`inviter_id`,`team_number`,`is_meal` from db_customer where id = ?", u["inviter_id"]).Scan(&u) // 获取用户
				log.Printf("用户:%+v", u)
				var thisLog = &AmountLog{
					CoinId:         float64(2),
					Direction:      1,
					Hash:           "",
					FromCustomerId: uId,
					CustomerId:     float64(u["id"].(int32)),
					BeforeAmount:   GetAccountCach(float64(u["id"].(int32))),
					CreateTime:     time.Now().Unix(),
				}
				if count == 0 && u["is_meal"].(uint32) == 1 {
					thisLog.FlowType = float64(63)
					thisLog.Remark = "直推奖励"
					log.Printf("用户%v直推奖励%v", u["id"], referralReward)
					myMoney = referralReward
					count++
				} else if 0 < count && count < 3 && ParseStringFloat(u["team_amount"].(string)) > 50000 && u["is_meal"].(uint32) == 1 {
					thisLog.FlowType = float64(64)
					thisLog.Remark = "合伙人奖励"
					log.Printf("用户%v合伙人奖励%v", u["id"], partnerReward)
					myMoney = partnerReward
					count++
				} else if ParseStringFloat(u["team_amount"].(string)) > 100000 && u["is_meal"].(uint32) == 1 {
					thisLog.FlowType = float64(65)
					thisLog.Remark = "创始人奖励"
					log.Printf("用户%v创始人奖励%v", u["id"], founderReward)
					myMoney = founderReward
					count++
				}
				after -= myMoney
				thisLog.Amount = myMoney
				thisLog.AfterAmount = thisLog.BeforeAmount + myMoney
				thisLog.Write(UserDB)
				UserDB.Exec("update db_coin_amount set amount = ? where customer_id = ? and coin_id = ?", thisLog.AfterAmount, u["id"], 2)
			} else {
				break
			}
			if count >= 4 {
				break
			}
		}
		log.Println(fmt.Sprintf("级差分红剩余金额:%v--- 平台收入:%v", after, after+realMoney*0.2))
	}
}

// ChangeAmount 修改账户业绩
func ChangeAmount(money float64, u *map[string]interface{}, db *gorm.DB, b bool) {
	// fmt.Println(*u)
	var grade = map[string]interface{}{}
	// 修改业绩
	if b && (*u)["profit_mine_amount"] != nil {
		grade["profit_mine_amount"] = ParseStringFloat((*u)["profit_mine_amount"].(string)) + money
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

func ParseFloatString(f float64) string {
	return fmt.Sprintf("%.0f", f)
}

// DeleteRebotLog 删除交易记录
func DeleteRebotLog(orderId string) {
	DB.Exec("delete from rebot_logs where order_id = ? ", orderId)
}

// LogStrategy 卖出盈利日志
func LogStrategy(coin_id interface{}, name interface{}, coin_name interface{}, order interface{},
	member interface{}, amount interface{}, price interface{}, isHand bool, money interface{}, status interface{}) {
	log.Printf("盈利日志,交易所名称:%v,用户:%v,币种名称:%v", name, member, coin_name)
	var (
		data     = map[string]interface{}{}
		categroy = map[string]interface{}{}
		// coin     = map[string]interface{}{}
	)
	UserDB.Raw("select id from db_task_category where `name` like ?", name).Scan(&categroy)
	// UserDB.Raw("select id from db_task_coin where `en_name` like ? and category_id = ?", coin_name, categroy["id"]).Scan(&coin)
	data["category_id"] = categroy["id"]
	data["coin_id"] = coin_id
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
	var id interface{}
	UserDB.Raw("select id from db_task_order_log where create_time = ? and order_id = ? and member_id = ? ", data["create_time"], order, member).Scan(&id)
	m, _ := money.(decimal.Decimal).Float64()
	RunOver(member, m, order, id, isHand, status)
}
