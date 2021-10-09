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
    "database/sql"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "strings"
    "time"
    grid "zmyjobs/corn/grid"
    model "zmyjobs/corn/models"
    util "zmyjobs/corn/uti"
    "zmyjobs/goex"

    "github.com/gorilla/mux"
    "gorm.io/gorm"

    "github.com/shopspring/decimal"
)

var INFO = "morning"

func Handler(w http.ResponseWriter) http.ResponseWriter {
    w.Header().Set("Access-Control-Allow-Origin", "*")             //允许访问所有域
    w.Header().Add("Access-Control-Allow-Headers", "Content-Type") //header的类型
    w.Header().Set("content-type", "application/json")
    w.Header().Set("referer-policy", "strict-origin-when-cross-origin")
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
        f    = false
        name string
    )
    res["status"] = "error"
    res["msg"] = "获取当前价格"
    res["data"] = "none"

    if id := r.FormValue("coin_id"); id != "" {
        model.UserDB.Raw("select name,category_id,coin_type from db_task_coin where id = ?", id).Scan(&data)
        if data["category_id"].(uint32) == 1 {
            name = "火币"
        }
        if data["category_id"].(uint32) == 2 {
            name = "币安"
        }
        if data["coin_type"].(int8) != 0 {
            f = true
        }
        if data["name"] != nil {
            // 目前获取火币价格
            ex := &util.Config{Name: name}
            price, err := ex.GetPrice(data["name"].(string), f)
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
        b, name, key, secret, pashare := model.GetApiConfig(model.ParseStringFloat(id), model.ParseStringFloat(category))
        // fmt.Println(b, name, key)
        if b {
            c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Host: "https://api.huobi.de.com", Pashare: pashare})
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
        b, name, key, secret, pashare := model.GetApiConfig(model.ParseStringFloat(id), model.ParseStringFloat(category))

        // fmt.Println(b, name, key)
        if b {
            c := grid.NewEx(&model.SymbolCategory{Category: name, Key: key, Secret: secret, PricePrecision: 8, AmountPrecision: 8, Pashare: pashare, Future: true})
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
    search := r.FormValue("search")
    var strategy []map[string]interface{}
    model.UserDB.Raw("select * from db_task_strategy where category_id = ? and status < 3", cate).Scan(&strategy)

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
            // fmt.Println(s["id"])
            var num int
            model.UserDB.Raw("select count(*) from db_task_order where task_strategy_id = ? and status < 3 and customer_id = ? ", s["id"], account).Scan(&num)
            if num > 0 {
                s["can"] = false
            } else {
                s["can"] = true
                if strings.Contains(s["name"].(string), util.UpString(search)) {
                    result = append(result, s)
                }
            }
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
    data := util.HttpGet(url, util.ProxyHttp("1124"))
    response["status"] = "success"
    response["msg"] = "获取币安市场u本位合约交易对"
    search := r.FormValue("search")
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
                model.UserDB.Raw("select name,id from db_task_coin where coin_type = ? or coin_type = ?", 1, 3).Scan(&coinDB)
                for _, c := range coinDB {
                    if coin.Name == c["name"] && coin.Name[len(coin.Name)-4:] == "USDT" {
                        coin.Id = c["id"]
                        if strings.Contains(coin.Name, util.UpString(search)) {
                            coins = append(coins, coin)
                        }
                    }
                }
            } else {
                if strings.Contains(coin.Name, util.UpString(search)) && coin.Name[len(coin.Name)-4:] == "USDT" {
                    coins = append(coins, coin)
                }
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
    data := util.HttpGet(url, util.ProxyHttp("1124"))
    response["status"] = "success"
    response["msg"] = "获取b本位合约交易对"
    search := r.FormValue("search")
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
                model.UserDB.Raw("select name,id from db_task_coin where coin_type = ? or coin_type = ?", 2, 4).Scan(&coinDB)
                for _, c := range coinDB {
                    if coin.Name == c["name"] && coin.Name[len(coin.Name)-3:] == "USD" {
                        coin.Id = c["id"]
                        if strings.Contains(coin.Name, util.UpString(search)) {
                            coins = append(coins, coin)
                        }
                    }
                }
            } else {
                if strings.Contains(coin.Name, util.UpString(search)) && coin.Name[len(coin.Name)-3:] == "USD" {
                    coins = append(coins, coin)
                }
            }
        }
        response["data"] = coins
    }
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

// 获取任务列表
func GetTask(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    var (
        response = map[string]interface{}{}
        result   = map[string]interface{}{}
        info     = map[string]interface{}{}
        all      = "0"
        category = "0"
    )
    response["status"] = "success"
    response["msg"] = "获取用户任务列表"
    search := r.FormValue("search")
    status := r.FormValue("status")
    category = r.FormValue("category_id")
    all = r.FormValue("order_type")
    task_id := r.FormValue("id")
    id := r.FormValue("account_id")
    if id != "" && status != "" || task_id != "" {
        var (
            res         []map[string]interface{}
            task        []map[string]interface{}
            status0     sql.NullInt64
            status1     sql.NullInt64
            status2     sql.NullInt64
            total_sum   sql.NullFloat64
            total_today sql.NullFloat64
        )
        if task_id != "" {
            response["msg"] = "获取单个策略"
            model.UserDB.Raw("select * from db_task_order where id = ? ", task_id).Scan(&task)
        } else {
            model.UserDB.Raw("select * from db_task_order where customer_id = ? and status = ? ", id, status).Scan(&task)
        }
        if all != "" && category != "" && all != "0" && category != "0" {
            model.UserDB.Raw("select * from db_task_order where customer_id = ? and status = ? and category_id = ? and order_type = ?", id, status, category, all).Scan(&task)
        } else if all != "" && all != "0" {
            // fmt.Println("e")
            model.UserDB.Raw("select * from db_task_order where customer_id = ? and status = ? and order_type = ?", id, status, all).Scan(&task)
        } else if category != "" && category != "0" {
            model.UserDB.Raw("select * from db_task_order where customer_id = ? and status = ? and category_id = ?", id, status, category).Scan(&task)
        }

        model.UserDB.Raw("select count(*) from db_task_order where customer_id = ? and status = 0", id).Scan(&status0)
        model.UserDB.Raw("select count(*) from db_task_order where customer_id = ? and status = 1", id).Scan(&status1)
        model.UserDB.Raw("select count(*) from db_task_order where customer_id = ? and status = 2", id).Scan(&status2)
        model.UserDB.Raw("select sum(av_amount) from db_task_order_log where member_id = ?", id).Scan(&total_sum)
        currentTime := time.Now()
        zeroTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
        old1Time := zeroTime.AddDate(0, 0, -1).Unix()
        model.UserDB.Raw("select sum(av_amount) from db_task_order_log where member_id = ? and create_time >= ? ", id, old1Time).Scan(&total_today)
        info["total_status0"] = status0.Int64
        info["total_status1"] = status1.Int64
        info["total_status2"] = status2.Int64
        info["total_sum"] = total_sum.Float64
        info["total_today"] = total_today.Float64
        for _, v := range task {
            // var strategy = map[string]interface{}{}
            // model.UserDB.Raw("select * from db_task_strategy where id = ?", v["task_strategy_id"]).Scan(&strategy)
            if strings.Contains(v["task_coin_name"].(string), util.UpString(search)) {
                var coin = map[string]interface{}{}
                model.UserDB.Raw("select coin_type from db_task_coin where id = ?", v["task_coin_id"]).Scan(&coin)
                if coin["coin_type"].(int8) == int8(0) {
                    v["label"] = "现货"
                }
                if coin["coin_type"].(int8) == int8(1) {
                    v["label"] = "U本位永续合约"
                }
                if coin["coin_type"].(int8) == int8(2) {
                    v["label"] = "B本位永续合约"
                }
                if coin["coin_type"].(int8) == int8(3) {
                    v["label"] = "U本位交割合约"
                }
                if coin["coin_type"].(int8) == int8(4) {
                    v["label"] = "B本位交割合约"
                }
                res = append(res, v)
            }
        }
        result["data"] = res
        response["data"] = result

        if task_id != "" {
            response["data"] = res[0]
        } else {
            result["info"] = info
        }
    } else {
        response["msg"] = "没有必须参数"
    }
    b, _ := json.Marshal(&response)
    fmt.Fprintln(w, string(b))
}

// 修改手动策略
func UpdateTask(w http.ResponseWriter, r *http.Request) {
    w = Handler(w)
    if r.Method == "POST" {
        r.ParseForm()
        // fmt.Println(r.Form)
        var form = map[string]interface{}{}
        str, _ := ioutil.ReadAll(r.Body)
        json.Unmarshal(str, &form)
        // fmt.Println("更新数据;", form)
        var (
            response = map[string]interface{}{}
            status   interface{}
            taskStra []map[string]interface{}
            s        []map[string]interface{}
            tasks    *gorm.DB
        )
        // r.ParseForm()
        // fmt.Println("获取数据", r.PostForm, r.Form, form, len(form))
        if len(form) == 0 {
            for k, v := range r.Form {
                if k == "order_id" {
                    form["id"] = v[0]
                } else {
                    form[k] = v[0]
                }
            }
        }
        // fmt.Println(fmt.Sprintf("%+v", form))

        response["status"] = "success"
        response["msg"] = "修改用户任务列表"

        if form != nil {
            status = form["status"]
        }
        task := model.UserDB.Table("db_task_order").Where("id = ? ", form["id"]).Find(&taskStra) // 要修改的order
        strategy := model.UserDB.Table("db_task_strategy").Where("order_id = ? ", form["id"]).Find(&s)
        if strategy.RowsAffected > 0 && len(s) > 0 {
            tasks = model.UserDB.Table("db_task_order").Where("task_strategy_id = ? and status = ?", s[0]["id"], 1)
        }

        if status != "" && status == "0" {
            if taskStra[0]["status"].(int8) == int8(1) {
                task.Update("status", 0)
                if strategy.RowsAffected > 0 {
                    strategy.Update("status", 0)
                    tasks.Update("status", 0)
                }
                response["msg"] = "策略暂停"

            } else {
                response["status"] = "error"
                response["msg"] = "策略不处于执行状态"
            }
        }
        if strategy.RowsAffected > 0 && len(s) > 0 {
            tasks = model.UserDB.Table("db_task_order").Where("task_strategy_id = ? and status = ?", s[0]["id"], 0)
        }
        if status != "" && status == "1" {
            if taskStra[0]["status"].(int8) == int8(0) {
                task.Update("status", 1)
                if strategy.RowsAffected > 0 {
                    strategy.Update("status", 1)
                    tasks.Update("status", 1)
                }
                response["msg"] = "策略开启"

            } else {
                response["status"] = "error"
                response["msg"] = "策略不处于暂停状态"
            }
        }
        if status != "" && status == "3" {
            if taskStra[0]["status"].(int8) != int8(1) {
                task.Update("status", 3)
                if strategy.RowsAffected > 0 {
                    strategy.Update("status", 3)
                }
                response["msg"] = "策略删除"

            } else {
                response["status"] = "error"
                response["msg"] = "策略不处于暂停或完成状态"
            }
        }
        if strategy.RowsAffected > 0 && len(s) > 0 {
            tasks = model.UserDB.Table("db_task_order").Where("task_strategy_id = ? and status = ?", s[0]["id"], 1)
        }
        if status != "" && status == "4" {
            if taskStra[0]["stop_buy"].(int64) == int64(2) && taskStra[0]["status"].(int8) == int8(1) {
                task.Update("stop_buy", 1)
                if strategy.RowsAffected > 0 {
                    strategy.Update("stop_buy", 1)
                    tasks.Update("stop_buy", 1)
                }
                response["msg"] = "恢复买入"

            } else {
                response["status"] = "error"
                response["msg"] = "策略不处于暂停买入状态或策略不处于开启状态"
            }
        }
        if status != "" && status == "5" {
            if taskStra[0]["stop_buy"].(int64) == int64(1) && taskStra[0]["status"].(int8) == int8(1) {
                task.Update("stop_buy", 2)
                if strategy.RowsAffected > 0 {
                    strategy.Update("stop_buy", 2)
                    tasks.Update("stop_buy", 2)
                }
                response["msg"] = "关闭买入"

            } else {
                response["status"] = "error"
                response["msg"] = "策略不处于开启买入状态"
            }
        }
        if status != "" && status == "7" {
            if taskStra[0]["one_buy"].(int64) == int64(1) && taskStra[0]["status"].(int8) == int8(1) {
                task.Update("one_buy", 2)
                if strategy.RowsAffected > 0 {
                    strategy.Update("one_buy", 2)
                    tasks.Update("one_buy", 2)
                }
                response["msg"] = "补仓"

            } else if taskStra[0]["one_buy"].(int64) == int64(2) {
                response["status"] = "error"
                response["msg"] = "其他操作占用"
            } else {
                response["status"] = "error"
                response["msg"] = "重复提交"
            }
        }
        if status != "" && status == "9" {
            if taskStra[0]["one_sell"].(int64) == int64(1) && taskStra[0]["status"].(int8) == int8(1) {
                task.Update("one_sell", 2)
                if strategy.RowsAffected > 0 {
                    strategy.Update("one_sell", 2)
                    tasks.Update("one_sell", 2)
                }
                response["msg"] = "清仓"
            } else if taskStra[0]["one_sell"].(int64) == int64(2) {
                response["status"] = "error"
                response["msg"] = "其他操作占用"
            } else {
                response["status"] = "error"
                response["msg"] = "重复提交"
            }
        }
        if strategy.RowsAffected > 0 && len(s) > 0 {
            tasks = model.UserDB.Table("db_task_order").Where("task_strategy_id = ? and status = ?", s[0]["id"], 0)
        }
        if len(form) > 0 {
            for _, name := range []string{"num", "strategy_id", "price", "bc_type", "price_add", "price_rate", "price_repair", "price_growth", "price_callback",
                "price_stop", "price_reduce", "frequency", "price_growth_type", "fixed_type", "double_first", "decline", "limit_high", "high_price"} {
                // fmt.Printf("类型：%T，名称:%v", form[name], form["name"])
                if form[name] != nil && form[name] != taskStra[0][name] {
                    if taskStra[0]["status"].(int8) == int8(0) {
                        task.Update(name, form[name])
                        if strategy.RowsAffected > 0 {
                            strategy.Update(name, form[name])
                            tasks.Update(name, form[name])
                        }
                    } else {
                        response["status"] = "error"
                        response["msg"] = "不处于暂停状态"
                    }
                }
            }
        }
        b, _ := json.Marshal(&response)
        fmt.Fprintln(w, string(b))
    } else {
        fmt.Fprintln(w, "not allow method")
    }
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
    router := mux.NewRouter()
    router.HandleFunc("/", IndexHandler).Methods("GET")
    router.HandleFunc("/account", GetAccountHandler)
    router.HandleFunc("/price", GetPrice)
    router.HandleFunc("/symbol", CheckSymobl)
    router.HandleFunc("/future", GetFuture)
    router.HandleFunc("/strategy", GetStrategy)
    router.HandleFunc("/u", GetFutureU)
    router.HandleFunc("/b", GetFutureB)
    router.HandleFunc("/task", GetTask)
    router.HandleFunc("/task/update", UpdateTask)

    go http.ListenAndServe(":80", router)

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
