/***************************
@File        : server.go
@Time        : 2021/07/30 09:28:48
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 获取交易所余额持仓信息等接口
****************************/

package xhttp

import (
    "fmt"
    "net/http"
)

func IndexHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintln(w, "hello world")
}

// GetAccount 获取用户信息
func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Println(r.Method, r.Form, r.Form.Get("account_id"))
}

func RunServer() {
    http.HandleFunc("/", IndexHandler)
    http.HandleFunc("/account", GetAccountHandler)
    http.ListenAndServe("127.0.0.0:8000", nil)
}
