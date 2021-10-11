/***************************
@File        : util.go
@Time        : 2021/07/28 15:08:45
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 工具
****************************/

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"zmyjobs/goex"
	"zmyjobs/goex/builder"

	"github.com/shopspring/decimal"
	"golang.org/x/net/proxy"
)

// Config 创建client需要的配置Struct
type Config struct {
    Name     string  //平台名称
    Host     string  // url
    APIKey   string  // 秘钥
    Secreet  string  // 秘钥
    ClientID string  // 连接id
    Lever    float64 // 期货杠杆倍数
    Passhare string  // ok 需要
}

/**
 *@title        : NewApi
 *@desc         : 新建现货货cli
 *@auth         : small_ant / time(2021/08/11 10:05:44)
 *@param        : c   / *Config            / `配置文件`
 *@return       : cli / goex.FutureRestAPI / `goex httpApiclient`
 */
func NewApi(c *Config) (cli goex.API) {
    api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID).HttpTimeout(time.Second * 60)
    // 本地使用代理
    // api := ProxySock().APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID)
    // fmt.Println(fmt.Sprintf("%+v", api), api.GetHttpClient())
    switch c.Name {
    case "币安":
        // api.BuildFuture(goex.BINANCE) 期货api
        cli = api.Build(goex.BINANCE) //创建现货api实例
    case "ok":
        api = api.ApiPassphrase(c.Passhare)
        cli = api.Build(goex.OKEX)
    default:
        cli = api.Build(goex.HUOBI_PRO) //创建现货api实例
    }
    return
}

/**
 *@title        : NewFutrueApi
 *@desc         : 新建期货cli
 *@auth         : small_ant / time(2021/08/11 10:05:44)
 *@param        : c   / *Config            / `配置文件`
 *@return       : cli / goex.FutureRestAPI / `goex httpApiclient`
 */
func NewFutrueApi(c *Config) (cli goex.FutureRestAPI) {
    api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).
        ClientID(c.ClientID).HttpTimeout(time.Second * 60).FuturesLever(c.Lever)
    // api := ProxySock().APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID).FuturesLever(c.Lever)
    switch c.Name {
    case "币安":
        // api.BuildFuture(goex.BINANCE) 期货api
        cli = api.BuildFuture(goex.BINANCE_SWAP)
    case "OKex":
        api = builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID).FuturesLever(c.Lever)
        cli = api.ApiPassphrase(c.Passhare).BuildFuture(goex.OKEX_SWAP)
    default:
        // 火币没有期货api
        cli = api.BuildFuture(goex.HUOBI)
    }
    return
}

// ProxySock socks5代理
func ProxySock() *builder.APIBuilder {
    cli := builder.NewCustomAPIBuilder(ProxyHttp("1124"))
    return cli
}

/**
 *@title        : UpString
 *@desc         : 字符串大写
 *@auth         : small_ant  / time(2021/08/11 10:03:50)
 *@param        : b / string / `传入字符串`
 *@return       : b / string / `大写字符串`
 */
func UpString(s string) string {
    return strings.ToUpper(s)
}

/*
     @title   : GetPrice
     @desc    : 获取行情价格 弃用
     @auth    : small_ant / time(2021/10/08 09:28:38)
     @param   :  / / ``
     @return  :  / / ``
**/
func (c *Config) GetPrice(symbol string, f bool) (price decimal.Decimal, err error) {
    var b *goex.Ticker
    if f {
        cli := NewFutrueApi(c)
        c := SwitchCoinType(symbol)
        if c == 1 || c == 3 {
            b, err = cli.GetFutureTicker(goex.NewCurrencyPair2(symbol), goex.SWAP_USDT_CONTRACT)
        } else {
            b, err = cli.GetFutureTicker(goex.NewCurrencyPair2(symbol), goex.SWAP_CONTRACT)
        }
    } else {
        cli := NewApi(c)
        b, err = cli.GetTicker(goex.NewCurrencyPair2(symbol))
    }
    if err == nil {
        price = decimal.NewFromFloat(b.Last)
    }
    return
}

/*
     @title   : SwitchCoinType
     @desc    : 从symbol 转换至交易对类型
     @auth    : small_ant / time(2021/10/08 09:29:49)
     @param   :  / / ``
     @return  :  / / ``
**/
func SwitchCoinType(name string) int {
    if name[len(name)-4:] == "usdt" || name[len(name)-4:] == "USDT" {
        return 1
    }
    if strings.Contains(name, "USDT") || strings.Contains(name, "usdt") {
        return 3
    }
    if name[len(name)-3:] == "usd" || name[len(name)-3:] == "USD" {
        return 2
    }
    if strings.Contains(name, "USD") || strings.Contains(name, "usd") {
        return 4
    }
    return 0
}

func ProxyHttp(port string) *http.Client {
    dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:"+port, nil, proxy.Direct)
    if err != nil {
        fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
    }
    // setup a http client
    httpTransport := &http.Transport{}
    httpTransport.Dial = dialer.Dial
    return &http.Client{Transport: httpTransport, Timeout: 20 * time.Second}
}

// ToMySymbol 把数据库交易对转换成api交易对
func ToMySymbol(name string) string {
    var (
        d int
        p int
    )
    if len(name) > 4 {
        d = len(name) - 4
    }

    if len(name) > 5 {
        p = len(name) - 5
    }
    // fmt.Println(name[p:])
    if name[p:] == "-USDT" {
        return strings.ToUpper(name[:p]) + "/" + "USDT"
    }
    if name[len(name)-3:] == "USD" {
        s := strings.Replace(name[:len(name)-3], "-", "", 1)
        return strings.ToUpper(s) + "/" + "USD"
    }
    if name[p:] == "_PERP" || name[p:] == "-SWAP" {
        return ToMySymbol(name[:p])
    }
    if strings.ToLower(name[d:]) == "usdt" {
        return strings.ToUpper(name[:d]) + "/" + strings.ToUpper(name[d:])
    }
    return name
}

/*
     @title   : HttpGet
     @desc    : 获取http get 请求内容
     @auth    : small_ant / time(2021/10/08 09:30:55)
     @param   :  / / ``
     @return  :  / / ``
**/
func HttpGet(url string, client *http.Client) (d interface{}) {
    // client := ProxyHttp("1123")
    // client := ProxyHttp()
    resp, err := client.Get(url)
    if err == nil {
        defer resp.Body.Close()
        content, _ := ioutil.ReadAll(resp.Body)
        _ = json.Unmarshal(content, &d)
    }
    return d
}
