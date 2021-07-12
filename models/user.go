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

var Ch = make(chan JobChan)

type User struct {
	gorm.Model
	Status   float64 //
	Strategy string
	IsRun    int64  // -1 待开始,10 正在运行, 1 运行完毕, 2 暂停, -2 启动失败,-10 运行策略失败
	Name     string // 交易对
	ApiKey   string
	Secret   string
	Category string
	ObjectId int32
	MinPrice string
	MaxPrice string
	Total    string
	Number   float64
	Error    string
}

// NewUser 从缓存获取如果数据库不存在就添加
func NewUser() {
	//log.Println("i am working for parse data")
	category := StringMap(GetCache("db_task_category"))
	orders := StringMap(GetCache("db_task_order"))
	api := StringMap(GetCache("db_task_api"))

	var Category = make(map[interface{}]interface{})
	for _, value := range category {
		Category[value["id"]] = value["name"]
	}

	var NewApi = make(map[interface{}]map[interface{}]map[string]interface{})
	var info = make(map[string]interface{})
	var useApi = make(map[interface{}]map[string]interface{})
	for _, v := range api {
		info["apikey"] = v["apikey"]
		info["secret"] = v["secretkey"]
		info["category"] = Category[v["category_id"]]
		useApi[v["member_id"]] = info
		NewApi[v["category_id"]] = useApi
	}
	//log.Println(NewApi)
	for _, order := range orders {
		// 符合条件的订单
		if NewApi[order["category_id"]] != nil && NewApi[order["category_id"]][order["customer_id"]] != nil {
			var u User
			// 数据库查找存在与否
			result := DB.Where(&User{ObjectId: int32(order["id"].(float64))}).First(&u)
			// 条件 数据库未找到，订单启用，创建新的任务
			if errors.Is(result.Error, gorm.ErrRecordNotFound) && order["status"].(float64) == 1 {
				u = User{
					ObjectId: int32(order["id"].(float64)),
					ApiKey:   NewApi[order["category_id"]][order["customer_id"]]["apikey"].(string),
					Secret:   NewApi[order["category_id"]][order["customer_id"]]["secret"].(string),
					Category: NewApi[order["category_id"]][order["customer_id"]]["category"].(string),
					Name:     parseSymbol(order["task_coin_name"].(string)),
					IsRun:    -1,
					MinPrice: order["price_stop"].(string),
					MaxPrice: order["price_add"].(string),
					Number:   order["num"].(float64),
					Total:    order["hold_num"].(string),
					Status:   1,
				}
				DB.Create(&u)
			}
			// 启用

			if u.IsRun == -2 {
				log.Println("检测到策略执行出错.................")
				Ch <- JobChan{u.ID, 000}
			}
			// status 0 禁用, 1 启用 2 暂停 3 删除 缓存与数据不相等

			if order["status"].(float64) != u.Status {
				log.Println("状态改变协程同步之策略协程", order["status"])
				switch order["status"].(float64) {
				case 2:
					// 发送暂停
					log.Println("发送给暂停")
					Ch <- JobChan{Id: u.ID, Run: 2}
				case 0:
					// 发送禁用
					log.Println("禁用任务")
					if u.Status != -1 {
						Ch <- JobChan{Id: u.ID, Run: -1}
					}
				case 3:
					// 发送删除
					log.Println("删除任务")
					Ch <- JobChan{Id: u.ID, Run: 3}
				default:
					// 1
					u.Status = 1
					if u.IsRun == 2 {
						u.IsRun = -1
					}
					u.Update()
				}
			} else {
				if u.Status == 1 {
					if u.IsRun == 2 {
						u.IsRun = -1
					}
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
func GetUserJob() []User {
	var users []User
	DB.Table("users").Find(&users)
	return users
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
func parseSymbol(s string) string {
	s = strings.Replace(s, "/", "", 1)
	return strings.ToLower(s)
}
