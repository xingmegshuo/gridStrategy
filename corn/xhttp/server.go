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
    grid "zmyjobs/corn/grid"
    model "zmyjobs/corn/models"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
    // fmt.Println("啥也没干")
    fmt.Fprintln(w, "hello world")
}

// GetAccount 获取用户信息
func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
    // fmt.Println(r.Method, "-----", r.Form, "-----", r.Form.Get("account_id"), "-----", r.FormValue("account_id"))
    if id := r.FormValue("account_id"); id != "" {
        category := r.FormValue("category")
        if category == "" {
            category = "1"
        }
        b, name, key, secret := model.GetApiConfig(model.ParseStringFloat(id), model.ParseStringFloat(category))
        if b {
            c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, Host: "https://api.huobi.de.com"})
            data, err := c.Ex.GetAccount()
            if err == nil {
                var response = map[string]interface{}{}
                for k, v := range data.SubAccounts {
                    if v.Amount > 0 {
                        // s, _ := json.Marshal(&v)
                        response[k.Symbol] = v
                    }
                }
                b, _ := json.Marshal(&response)
                fmt.Fprintln(w, string(b))
            } else {
                fmt.Fprintln(w, "api 出错或超时")
            }
        } else {
            fmt.Fprintln(w, "用户不正确")
        }
    } else {
        fmt.Fprintln(w, "参数不正确")
    }
}

func RunServer() {
    log.Println("服务开启")
    http.HandleFunc("/", IndexHandler)
    http.HandleFunc("/account", GetAccountHandler)
    go http.ListenAndServe(":80", nil)
    // fmt.Println("服务运行")
}
