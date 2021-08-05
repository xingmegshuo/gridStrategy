/***************************
@File        : server.go
@Time        : 2021/07/30 09:28:48
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 获取交易所余额持仓信息等接口
****************************/

package xhttp

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    grid "zmyjobs/corn/grid"
    model "zmyjobs/corn/models"

    "github.com/shopspring/decimal"
)

func Handler(w http.ResponseWriter) http.ResponseWriter {
    w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
    w.Header().Set("content-type", "application/json")
    return w
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    // fmt.Println("啥也没干")
    fmt.Fprintln(w, "hello world")
}

// GetPrice 当前价格
func GetPrice(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)

    var (
        res  = map[string]interface{}{}
        data = map[string]interface{}{}
    )
    res["status"] = "error"
    res["msg"] = "获取当前价格"
    res["data"] = "none"

    if id := r.FormValue("coin_id"); id != "" {
        model.UserDB.Raw("select name,category_id from db_task_coin where id = ?", id).Scan(&data)
        if data["name"] != nil {
            // 目前获取火币价格
            symbol := model.SymbolCategory{}
            symbol.BaseCurrency, symbol.QuoteCurrency = SplitString(data["name"].(string))
            ex := grid.NewEx(&symbol)
            price, err := ex.GetPrice()
            if err == nil {
                res["status"] = "success"
                res["data"] = price
            } else {
                res["msg"] = err.Error()
            }
        } else {
            res["msg"] = "获取信息出错"
        }
    } else {
        res["msg"] = "参数解析出错"
    }
    b, _ := json.Marshal(&res)
    fmt.Fprintln(w, string(b))
}

// GetAccount 获取用户信息
func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)

    var (
        response = map[string]interface{}{}
        res      = map[string]interface{}{}
        list     = []map[string]interface{}{}
        sumMoney decimal.Decimal
    )
    response["status"] = "error"
    response["msg"] = "获取用户账户资金"
    response["data"] = "none"

    if id := r.FormValue("account_id"); id != "" {
        category := r.FormValue("category")
        if category == "" {
            category = "1"
        }
        b, name, key, secret := model.GetApiConfig(model.ParseStringFloat(id), model.ParseStringFloat(category))
        // fmt.Println(b, name, key)
        if b {
            c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Host: "https://api.huobi.de.com"})
            data, err := c.Ex.GetAccount()
            if err == nil {
                response["status"] = "success"
                for k, v := range data.SubAccounts {
                    if v.Amount > 0 {
                        one := map[string]interface{}{}
                        one["amount"] = decimal.NewFromFloat(v.Amount).Round(8)
                        symbol := model.SymbolCategory{BaseCurrency: k.Symbol, QuoteCurrency: "USDT", Category: name, PricePrecision: 8, AmountPrecision: 8, Host: "https://api.huobi.de.com"}
                        cli := grid.NewEx(&symbol)
                        price, _ := cli.GetPrice()
                        // fmt.Println(price, err)
                        if k.Symbol == "USDT" {
                            one["money"] = decimal.NewFromFloat(v.Amount).Round(8)
                        } else {
                            one["money"] = price.Mul(decimal.NewFromFloat(v.Amount)).Round(8)
                        }
                        one["symbol"] = k.Symbol
                        list = append(list, one)
                    }
                }
                for _, v := range list {
                    sumMoney = sumMoney.Add(v["money"].(decimal.Decimal))
                }
                // fmt.Println(sumMoney)
                for _, v := range list {
                    // fmt.Println(fmt.Sprintf("%T", v["money"]))
                    rate := decimal.NewFromInt(100)
                    m := v["money"].(decimal.Decimal)
                    value := m.Div(sumMoney).Mul(rate)
                    v["position"] = value.Round(4)
                }
                res["list"] = list
                res["sum"] = sumMoney
                response["data"] = res
            } else {
                response["msg"] = err.Error()
            }
        } else {
            response["msg"] = "获取信息出错"
        }
    } else {
        response["msg"] = "参数解析出错"
    }
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

func RunServer() {
    log.Println("服务开启")

    http.HandleFunc("/", IndexHandler)
    http.HandleFunc("/account", GetAccountHandler)
    http.HandleFunc("/price", GetPrice)
    go http.ListenAndServe(":80", nil)

    // fmt.Println("服务运行")
}

// SplitString 交易对
func SplitString(name string) (base string, quote string) {
    stringSlince := strings.Split(name, "/")
    return stringSlince[0], stringSlince[1]
}
