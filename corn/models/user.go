/***************************
@File        : user.go
@Time        : 2021/7/2 15:14
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 用户策略从redis读出写入sqlite
****************************/

package model

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// JobChan 管道信息，依据管道发送用户的暂停策略删除策略操作
type JobChan struct {
	Id  uint
	Run int
}

// var cacheNone = map[interface{}]interface{}{}

var Ch = make(chan JobChan)

type User struct {
	gorm.Model
	Status    float64 // 状态
	BasePrice float64 // 基础价格
	Money     float64 // 钱
	Base      int     // 当前第几单
	Type      float64 // 类型
	Strategy  string  // 策略内容
	IsRun     int64   // -1 待开始,10 正在运行, 1 运行完毕, 2 暂停, -2 启动失败,-10 运行策略失败 100 等待重新开始
	Name      string  // 交易对
	ApiKey    string  // 秘钥
	Secret    string  // 秘钥
	Category  string  // 分类
	ObjectId  int32   `gorm:"index"` // 订单id
	MinPrice  string  // 最低价
	MaxPrice  string  // 最高价
	Total     string  // 持仓
	Number    float64 // 单数
	Error     string  // 错误
	Grids     string  // 策略预设
	Custom    float64 // 用户id
	RunCount  int     // 执行次数
	RealGrids string  // 实际执行策略
	Arg       string  // 策略参数
	Symbol    string  // 交易对参数
}

// NewUser 从缓存获取如果数据库不存在就添加
func NewUser() {
	orders := StringMap(GetCache("db_task_order"))
	if GetCache("火币交易对") == "" {
		time.Sleep(time.Second * 2)
	}
	for _, order := range orders {
		// 符合条件的订单
		// if _, Ok := cacheNone[order["id"]]; !Ok {
		b, cate, api, sec := GetApiConfig(order["customer_id"], order["category_id"])
		if b {
			var u User
			// 数据库查找存在与否
			result := DB.Where(&User{ObjectId: int32(order["id"].(float64))}).First(&u)
			// 条件 数据库未找到，订单启用，创建新的任务
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				log.Println("new 数据")
				u = User{
					ObjectId: int32(order["id"].(float64)),
					ApiKey:   api,
					Secret:   sec,
					Category: cate,
					Name:     ParseSymbol(order["task_coin_name"].(string)),
					IsRun:    -1,
					Strategy: parseInput(order),
					// MinPrice: order["price_stop"].(string),
					// MaxPrice: order["price_add"].(string),
					Money:  GetAccount(order["customer_id"].(float64)),
					Number: order["num"].(float64),
					// Total:    order["hold_num"].(string),
					Type:   order["frequency"].(float64),
					Status: 1,
					Base:   0,
					Custom: order["customer_id"].(float64),
				}
				u = UpdateUser(u)
				DB.Create(&u)
			} else if result.Error == nil {
				// status 0 暂停, 1 启用 2 完成 3 删除 缓存与数据不相等
				if order["status"].(float64)+1 != u.Status {
					log.Println("状态改变协程同步之策略协程", order["status"], u.ObjectId)
					u.Status = order["status"].(float64) + 1
					u.Update()
					switch order["status"].(float64) {
					case 0:
						// 发送暂停
						Ch <- JobChan{Id: u.ID, Run: 2}
					case 3:
						// Ch <- JobChan{Id: u.ID, Run: 3}
					case 1:
						u.Status = 2
						if u.IsRun == 2 {
							u.IsRun = -1
						}
						u.Update()
					default:
					}
				} else {
					if u.Status == 2 {
						if u.IsRun == 2 {
							u.IsRun = -1
							u = UpdateUser(u)
							u.Update()
						}
						if u.IsRun == -10 {
							StrategyError(u.ObjectId, u.Error)
						}
					}
				}
				// 更新策略参数
				if u.Strategy != parseInput(order) {
					log.Println("修改参数")
					u.Strategy = parseInput(order)
					if u.IsRun == 10 && order["stop_buy"].(float64) == 1 {
						OperateCh <- Operate{Id: float64(u.ObjectId), Op: 4}
					}
					u = UpdateUser(u)
					u.Update()
				}
			}
		}
	}
}

// StringMap 字符串转map
func StringMap(data string) []map[string]interface{} {
	var res []map[string]interface{}
	_ = json.Unmarshal([]byte(data), &res)
	return res
}

// GetUserJob 获取所有用户
func GetUserJob() *[]User {
	var users []User
	DB.Table("users").Find(&users)
	return &users
}

// Update 更新
func (u *User) Update() {
	DB.Updates(u)
}

// StopUser 停止所有任务
func StopUser() {
	DB.Where(&User{Status: float64(2), IsRun: 10}).UpdateColumns(User{IsRun: int64(2)})
	DB.Where(&User{Status: float64(2), IsRun: 100}).UpdateColumns(User{IsRun: int64(2)})
}

// parseSymbol 解析交易对
func ParseSymbol(s string) string {
	s = strings.Replace(s, "/", "", 1)
	return strings.ToLower(s)
}

// parseInput 策略输入处理
func parseInput(order map[string]interface{}) string {
	var strategy = map[string]interface{}{}
	strategy["FirstBuy"] = order["price"]          // 首单数量
	strategy["rate"] = order["price_rate"]         // 补仓比例
	strategy["growth"] = order["price_growth"]     // 补仓增幅比例
	strategy["callback"] = order["price_callback"] // 回调比例
	strategy["reduce"] = order["price_reduce"]     // 回降比例
	strategy["Type"] = order["frequency"]          // 类型，等于1为单次，等于2为循环执行
	strategy["stop"] = order["price_stop"]         // 止盈比例
	strategy["Strategy"] = order["strategy_id"]    // 策略分类
	strategy["NowPrice"] = order["now_price"]      // 当前价格
	strategy["add"] = order["price_add"]           // 加仓金额
	strategy["reset"] = order["price_repair"]      // 补仓复位
	strategy["frequency"] = order["frequency"]     // 是否为循环策略
	strategy["decline"] = order["decline"]         // 跌幅比例
	strategy["allSell"] = order["one_sell"]        // 一键平仓
	strategy["one_buy"] = order["one_buy"]         // 一键补仓
	strategy["double"] = order["double_first"]     // 首单加倍
	strategy["limit_high"] = order["limit_high"]   // 限高
	strategy["high_price"] = order["high_price"]   // 限高价格
	strategy["stop_buy"] = order["stop_buy"]       // 停止买入
	strategy["order_type"] = order["order_type"]   // 手动自动
	return ToStringJson(strategy)
}

// GetApiConfig 获取用户设置的平台分类及秘钥
func GetApiConfig(memberid interface{}, category interface{}) (bool, string, string, string) {
	var (
		name   string
		apiKey string
		secret string
	)
	// fmt.Println(category, memberid, "---")
	api := StringMap(GetCache("db_task_api"))
	Category := StringMap(GetCache("db_task_category"))
	// fmt.Println(category, memberid, "---")

	for _, value := range Category {
		// fmt.Println(value["id"] == category, value["id"], category, reflect.DeepEqual(value["id"], category), fmt.Sprintf("%T %T", value["id"], category))
		if value["id"] == category {
			name = value["name"].(string)
		}
	}
	for _, a := range api {
		if a["category_id"] == category && a["member_id"] == memberid {
			apiKey = a["apikey"].(string)
			secret = a["secretkey"].(string)
		}
	}
	// fmt.Println(name, apiKey, secret)
	if name != "" && apiKey != "" && secret != "" {
		return true, name, apiKey, secret
	} else {
		return false, "", "", ""
	}
}

// GetAccount 获取用户预充值余额
func GetAccount(uId float64) float64 {
	var amount = map[string]interface{}{}
	UserDB.Raw("select `meal_amount` from db_customer where id = ?", uId).Scan(&amount)
	log.Println(amount, "-------预充值金额")
	if amount["meal_amount"] != nil {
		return ParseStringFloat(amount["meal_amount"].(string))
	}
	return 0
}

// UpdateStatus 刷新状态
func UpdateStatus(id uint) (res int64) {
	DB.Raw("select is_run from users where id = ?", id).Scan(&res)
	// log.Println("我的状态", res)
	return
}

// LoadSymbols 获取交易对
func LoadSymbols(name string) []map[string]interface{} {
	switch name {
	case "火币":
		// fmt.Println(GetCache("火币交易对"))
		return StringMap(GetCache("火币交易对"))
	default:
		return StringMap(GetCache("火币交易对"))
	}
}

// SourceStrategy 生成并写入表
func SourceStrategy(u User, load bool) (*[]Grid, error) {
	if data, ok := LoadStrategy(u); ok && !load {
		return data, nil
	} else {
		if strategy, e := MakeStrategy(u); e == nil {
			return strategy, nil
		} else {
			return nil, e
		}
	}
}

// LoadStrategy 从表中加载策略
func LoadStrategy(u User) (*[]Grid, bool) {
	var data []Grid
	_ = json.Unmarshal([]byte(u.Grids), &data)
	if float64(len(data)) == u.Number {
		return &data, true
	}
	return &data, false
}

// UpdateUser  更新u
func UpdateUser(u User) User {
	u.Arg = ToStringJson(ParseStrategy(u))

	u.Symbol = ToStringJson(NewSymbol(u))

	if grid, e := SourceStrategy(u, true); e == nil {

		s, _ := json.Marshal(grid)
		u.Grids = string(s)
		PreView(u.ObjectId, u.Grids)
	} else {
		u.IsRun = -10
		u.Status = 1
		u.Error = e.Error()
		StrategyError(u.ObjectId, e.Error())
	}
	return u
}
