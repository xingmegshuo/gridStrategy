/***************************
@File        : server.go
@Time        : 2021/07/30 09:28:48
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 获取交易所余额持仓信息等接口
****************************/

/**
 * @title        : package xhttp
 * @desc         : http 服务包
 * @auth         : small_ant / time(2021/08/11 10:25:04)
**/
package xhttp

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "strings"
    "time"
    grid "zmyjobs/corn/grid"
    model "zmyjobs/corn/models"
    util "zmyjobs/corn/uti"
    "zmyjobs/goex"

    "github.com/shopspring/decimal"
)

var INFO = "morning"

func Handler(w http.ResponseWriter) http.ResponseWriter {
    w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
    w.Header().Set("content-type", "application/json")
    return w
}

/**
 * @title        : IndexHandler
 * @desc         : 测试连接访问
 * @auth         : small_ant / time(2021/08/11 10:22:15)
 * @param        : / / ``
 * @return       : hello world / / ``
**/
func IndexHandler(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    // fmt.Println("啥也没干")
    fmt.Fprintln(w, "hello world")
}

/*
     @title   :
     @desc    :
     @auth    : small_ant / time(2021/08/12 15:35:50)
     @param   :  / / ``
     @return  :  / / ``
**/
func CheckSymobl(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        res = map[string]interface{}{}
        // data = map[string]interface{}{}
    )
    res["status"] = "error"
    res["msg"] = "交易对参数无效"
    if id := r.FormValue("category_id"); id != "" {
        categoryName := "币安"
        if id == "5" {
            categoryName = "ok"
        }
        quote := "USDT"
        if r.FormValue("usdt") == "false" {
            quote = "USD"
        }
        if name := r.FormValue("name"); name != "" {
            coinType := util.SwitchCoinType(name)
            ex := grid.NewEx(&model.SymbolCategory{Symbol: name, Future: true, Category: categoryName, QuoteCurrency: quote})
            _, err := ex.GetPrice()
            if err == nil {
                res["status"] = "success"
                res["msg"] = "交易对参数有效"
                res["data"] = coinType
            }
        }
    }
    b, _ := json.Marshal(&res)
    fmt.Fprintln(w, string(b))
}

/*
    语言是历史的档案……语言是诗歌的化石
     -  爱献生

     @title   : GetPrice
     @desc    : http : /price 获取交易对价格
     @auth    : small_ant / time(2021/08/11 10:12:38)
     @param   : coin_id / string / `币种id`
     @return  : status,msg,data / string,string,interface{} / `状态,信息介绍,数据`
**/
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

/*
   @title        : GetAccountHandler
   @desc         : http : /account 获取用户交易所账户明细
   @auth         : small_ant / time(2021/08/11 10:12:38)
   @param        : account_id,category / string,string / `用户id,交易所分类`
   @return       : status,msg,data / string,string,interface{} / `状态,信息介绍,数据`
**/
func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
        res      = map[string]interface{}{}
        list     []map[string]interface{}
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

/*

     @title   : FutureAccount
     @desc    :
     @auth    : small_ant / time(2021/08/17 13:43:11)
     @param   :  / / ``
     @return  :  / / ``
**/
func GetFuture(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
        // res      = map[string]interface{}{}
        // list     = []map[string]interface{}{}
        // sumMoney decimal.Decimal
    )
    response["status"] = "success"
    response["msg"] = "获取用户合约持仓"
    // response["data"] = "none"
    if id := r.FormValue("account_id"); id != "" {
        category := r.FormValue("category")
        if category == "" {
            category = "2"
        }
        // fmt.Println(id,category)
        b, name, key, secret := model.GetApiConfig(model.ParseStringFloat(id), model.ParseStringFloat(category))

        // fmt.Println(b, name, key)
        if b {
            c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Future: true})
            // fmt.Println(fmt.Sprintf("%+v", c))
            data, err := c.Future.GetFuturePosition(goex.UNKNOWN_PAIR, goex.SWAP_USDT_CONTRACT)
            if err == nil {
                var u = []map[string]interface{}{}
                for _, v := range data {
                    var one = map[string]interface{}{}
                    one["amount"] = v.BuyAmount
                    one["symbol"] = v.Symbol.String()
                    one["unprofit"] = v.BuyProfitReal
                    one["level"] = v.LeverRate
                    if v.ContractType == "LNOG" {
                        one["slide"] = "做多"
                    } else {
                        one["slide"] = "做空"
                    }
                    u = append(u, one)
                }
                response["U"] = u
            } else {
                response["status"] = "error"
                response["msg"] = err.Error()
            }
            Bdata, Berr := c.Future.GetFuturePosition(goex.UNKNOWN_PAIR, goex.SWAP_CONTRACT)
            if Berr == nil {
                var u = []map[string]interface{}{}
                for _, v := range Bdata {
                    var one = map[string]interface{}{}
                    one["amount"] = v.BuyAmount
                    one["symbol"] = v.Symbol.String()
                    one["unprofit"] = v.BuyProfitReal
                    one["level"] = v.LeverRate
                    if v.ContractType == "LNOG" {
                        one["slide"] = "做多"
                    } else {
                        one["slide"] = "做空"
                    }
                    u = append(u, one)
                }
                response["B"] = u
            } else {
                response["status"] = "error"
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

// 获取自动策略列表
func GetStrategy(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
        result   []map[string]interface{}
    )
    response["status"] = "success"
    response["msg"] = "获取自动策略列表"
    t := r.FormValue("type")
    cate := r.FormValue("category_id")
    account := r.FormValue("account_id")
    var strategy []map[string]interface{}
    model.UserDB.Raw("select * from db_task_strategy where category_id = ?", cate).Scan(&strategy)
    for _, s := range strategy {
        var coin_type = map[string]interface{}{}
        condintaion := 0
        model.UserDB.Raw("select coin_type from db_task_coin where id = ?", s["task_coin_id"]).Scan(&coin_type)
        if t == "0" {
            response["msg"] = "获取现货自动策略"
        }
        if t == "1" {
            response["msg"] = "获取u本位自动策略"
            condintaion = 1
        }
        if t == "2" {
            response["msg"] = "获取b本位自动策略"
            condintaion = 2
        }
        if coin_type["coin_type"].(int8) == int8(condintaion) {
            fmt.Println(s["id"])
            var num int
            model.UserDB.Raw("select count(*) from db_task_order where task_strategy_id = ? and status != 3 and customer_id = ? ", s["id"], account).Scan(&num)
            fmt.Println(num)
            if num > 0 {
                s["can"] = false
            } else {
                s["can"] = true
            }
            result = append(result, s)
        }
    }
    response["data"] = result
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

// 获取u交易对
func GetFutureU(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
    )

    url := "https://fapi.binance.com/fapi/v1/ticker/24hr"
    data := util.HttpGet(url)
    response["status"] = "success"
    response["msg"] = "获取币安市场u本位合约交易对"
    if id := r.FormValue("db"); id == "true" {
        response["msg"] = "获取自选u本位合约交易对"
    }
    if data == nil {
        response["status"] = "error"
    } else {
        var coins []model.Coin
        for _, v := range data.([]interface{}) {
            var coin model.Coin
            one := v.(map[string]interface{})
            coin.CategoryId = 2
            coin.Name = util.ToMySymbol(one["symbol"].(string))
            coin.PriceUsd = model.ParseStringFloat(one["lastPrice"].(string))
            coin.Price = coin.PriceUsd * 6.5
            coin.DayAmount = model.ParseStringFloat(one["quoteVolume"].(string)) * 6.5 / 100000000
            coin.Raf = model.ParseStringFloat(one["priceChangePercent"].(string))
            coin.CoinType = util.SwitchCoinType(coin.Name)
            coin.EnName = coin.Name
            coin.CoinName = coin.Name
            coin.CreateTime = int(time.Now().Unix())
            if id := r.FormValue("db"); id == "true" {
                var coinDB []map[string]interface{}
                model.UserDB.Raw("select name from db_task_coin where coin_type = ? or coin_type = ?", 1, 3).Scan(&coinDB)
                for _, c := range coinDB {
                    if coin.Name == c["name"] {
                        coins = append(coins, coin)
                    }
                }
            } else {
                coins = append(coins, coin)
            }

        }
        response["data"] = coins
    }
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

// 获取b交易对
func GetFutureB(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
    )
    url := "https://dapi.binance.com/dapi/v1/ticker/24hr"
    data := util.HttpGet(url)
    response["status"] = "success"
    response["msg"] = "获取b本位合约交易对"
    if id := r.FormValue("db"); id == "true" {
        response["msg"] = "获取自选b本位合约交易对"
    }
    if data == nil {
        response["status"] = "error"
    } else {
        var coins []model.Coin
        for _, v := range data.([]interface{}) {
            var coin model.Coin
            one := v.(map[string]interface{})
            coin.CategoryId = 2
            coin.Name = util.ToMySymbol(one["symbol"].(string))
            coin.PriceUsd = model.ParseStringFloat(one["lastPrice"].(string))
            coin.Price = coin.PriceUsd * 6.5
            coin.DayAmount = model.ParseStringFloat(one["baseVolume"].(string)) * coin.Price / 100000000
            coin.Raf = model.ParseStringFloat(one["priceChangePercent"].(string))
            coin.CoinType = util.SwitchCoinType(coin.Name)
            coin.CreateTime = int(time.Now().Unix())
            coin.EnName = coin.Name
            coin.CoinName = coin.Name
            if id := r.FormValue("db"); id == "true" {
                var coinDB []map[string]interface{}
                model.UserDB.Raw("select name from db_task_coin where coin_type = ? or coin_type = ?", 2, 4).Scan(&coinDB)
                for _, c := range coinDB {
                    if coin.Name == c["name"] {
                        coins = append(coins, coin)
                    }
                }
            } else {
                coins = append(coins, coin)
            }
        }
        response["data"] = coins
    }
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

/**
 *@title        : RunServer
 *@desc         : 开启一个http服务端接收请求
 *@auth         : small_ant / time(2021/08/11 10:10:40)
 *@param        : / / ``
 *@return       : / / ``
 */
func RunServer() {
    log.Println("服务开启")

    http.HandleFunc("/", IndexHandler)
    http.HandleFunc("/account", GetAccountHandler)
    http.HandleFunc("/price", GetPrice)
    http.HandleFunc("/symbol", CheckSymobl)
    http.HandleFunc("/future", GetFuture)
    http.HandleFunc("/strategy", GetStrategy)
    http.HandleFunc("/u", GetFutureU)
    http.HandleFunc("/b", GetFutureB)

    go http.ListenAndServe(":80", nil)

    // fmt.Println("服务运行")
}

/**
 *@title        : SplitString
 *@desc         : 分割交易对
 *@auth         : small_ant / time(2021/08/11 10:11:12)
 *@param        : name / string / `交易对`
 *@return       : base,quote / string,string / `交易币种，基础币种`
 */
func SplitString(name string) (base string, quote string) {
    stringSlince := strings.Split(name, "/")
    if len(stringSlince) > 0 {
        return stringSlince[0], stringSlince[1]
    }
    return "", ""
}
