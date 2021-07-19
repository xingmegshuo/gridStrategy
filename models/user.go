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

	"gorm.io/gorm"
)

// JobChan 管道信息，依据管道发送用户的暂停策略删除策略操作
type JobChan struct {
	Id  uint
	Run int
}

var cacheNone = map[interface{}]interface{}{}

var Ch = make(chan JobChan)

type User struct {
	gorm.Model
	Status    float64 // 状态
	BasePrice float64 // 基础价格
	Money     float64 // 钱
	Base      int     // 当前第几单
	Type      float64 // 类型
	Strategy  string  // 策略内容
	IsRun     int64   // -1 待开始,10 正在运行, 1 运行完毕, 2 暂停, -2 启动失败,-10 运行策略失败
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
	Grids     string  // 策略
	Custom    float64 // 用户id
}

// NewUser 从缓存获取如果数据库不存在就添加
func NewUser() {
	// log.Println("i am working for parse data")
	orders := StringMap(GetCache("db_task_order"))
	// log.Println(NewApi)
	for _, order := range orders {
		// 	// 符合条件的订单
		if _, Ok := cacheNone[order["id"]]; !Ok {
			b, cate, api, sec := GetApiConfig(order["customer_id"], order["category_id"])
			if b {
				var u User
				// 数据库查找存在与否
				result := DB.Where(&User{ObjectId: int32(order["id"].(float64))}).First(&u)
				// 			// 条件 数据库未找到，订单启用，创建新的任务
				if errors.Is(result.Error, gorm.ErrRecordNotFound) && order["status"].(float64) == 1 {
					// log.Println(NewApi[order["customer_id"]])
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
						Money:  order["money"].(float64),
						Number: order["num"].(float64),
						// Total:    order["hold_num"].(string),
						Type:   order["frequency"].(float64),
						Status: 1,
						Base:   0,
						Custom: order["customer_id"].(float64),
					}
					DB.Create(&u)
				} else {
					cacheNone[order["id"]] = order["status"]
				}
				//
				// if u.IsRun == -2 {
				// 	// log.Println("检测到策略执行出错.................")
				// 	// Ch <- JobChan{u.ID, 000}
				// }
				// status 0 禁用, 1 启用 2 暂停 3 删除 缓存与数据不相等
				var p User
				res := DB.Where(&User{ObjectId: int32(order["id"].(float64))}).First(&p)
				if !errors.Is(res.Error, gorm.ErrRecordNotFound) {
					if order["status"].(float64) != p.Status {
						log.Println("状态改变协程同步之策略协程", order["status"])
						switch order["status"].(float64) {
						case 2:
							// 发送暂停
							log.Println("发送给暂停")
							Ch <- JobChan{Id: u.ID, Run: 2}
						case 0:
							// 发送禁用
							log.Println("禁用任务")
							if p.Status != -1 {
								Ch <- JobChan{Id: u.ID, Run: -1}
							}
						// case 3:
						// 	// 发送删除
						// 	u.Status = float64(3)
						// 	log.Println("删除任务")
						// 	Ch <- JobChan{Id: u.ID, Run: 3}
						// 	u.Update()
						// 	UserDB.Delete(&u)
						default:
							// 1
							p.Status = 1
							if p.IsRun == 2 {
								p.IsRun = -1
							}
							p.Update()
						}
					} else {
						if p.Status == 1 {
							if p.IsRun == 2 {
								p.IsRun = -1
							}
							p.Update()
						}
					}
				}
			} else {
				cacheNone[order["id"]] = order["status"]
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
	DB.Updates(&u)
}

// StopUser 停止所有任务
func StopUser() {
	DB.Where(&User{Status: float64(1), IsRun: 10}).UpdateColumns(User{IsRun: int64(2)})
}

// parseSymbol 解析交易对
func ParseSymbol(s string) string {
	s = strings.Replace(s, "/", "", 1)
	return strings.ToLower(s)
}

// parseInput 策略输入处理
func parseInput(order map[string]interface{}) string {
	var strategy = map[string]interface{}{}
	strategy["FirstBuy"] = order["price"] // 首单数量
	// strategy["BuyRate"] = order["price"].(float64) * order["price_rate"].(float64) // 补仓数量
	strategy["rate"] = order["price_rate"]     // 补仓比例
	strategy["growth"] = order["price_growth"] // 补仓增幅比例
	// strategy["BuyGrowth"] = order["price"].(float64) * order["price_growth"].(float64) // 补仓增幅数量
	strategy["callback"] = order["price_callback"] // 回调比例
	strategy["reduce"] = order["price_reduce"]     // 回降比例
	strategy["Type"] = order["frequency"]          // 类型，等于1为单次，等于2为循环执行
	strategy["stop"] = order["price_stop"]         // 止盈比例
	strategy["Strategy"] = order["strategy_id"]    // 策略分类
	strategy["NowPrice"] = order["now_price"]      // 当前价格
	strategy["add"] = order["price_add"]           // 加仓金额
	strategy["types"] = order["strategy_id"]       // 分类
	strategy["reset"] = order["price_repair"]      // 补仓复位
	str, _ := json.Marshal(&strategy)
	return string(str)
}

func GetApiConfig(memberid interface{}, category interface{}) (bool, string, string, string) {
	var (
		name   string
		apiKey string
		secret string
	)
	api := StringMap(GetCache("db_task_api"))
	Category := StringMap(GetCache("db_task_category"))

	for _, value := range Category {
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
	if name != "" && apiKey != "" && secret != "" {
		return true, name, apiKey, secret
	} else {
		return false, "", "", ""
	}
}
