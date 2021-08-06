/***************************
@File        : util.go
@Time        : 2021/07/28 15:08:45
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 工具
****************************/

package util

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"zmyjobs/goex"
	"zmyjobs/goex/builder"

	"golang.org/x/net/proxy"
)

type Config struct {
    Name     string //平台名称
    Host     string // url
    APIKey   string // 秘钥
    Secreet  string // 秘钥
    ClientID string // 连接id
}

/**
 * @desc NewApi 现货api  目前火币
 * @param *config 配置信息
 */
func NewApi(c *Config) (cli goex.API) {
    api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID).HttpTimeout(time.Second * 60)
    // api := ProxySock().APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID)
    // fmt.Println(fmt.Sprintf("%+v", api), api.GetHttpClient())
    switch c.Name {
    case "币安":
        // api.BuildFuture(goex.BINANCE) 期货api
        cli = api.Build(goex.BINANCE) //创建现货api实例
    default:
        cli = api.Build(goex.HUOBI_PRO) //创建现货api实例
    }
    return
}

func NewFutrueApi(c *Config) (cli goex.FutureRestAPI) {
    // api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).
    // ClientID(c.ClientID).HttpTimeout(time.Second * 60)
    api := ProxySock().APIKey(c.APIKey).APISecretkey(c.Secreet).ClientID(c.ClientID)
    switch c.Name {
    case "币安":
        // api.BuildFuture(goex.BINANCE) 期货api
        cli = api.BuildFuture(goex.BINANCE)
    default:
        // 火币没有期货api
        cli = api.BuildFuture(goex.HUOBI)
    }
    return
}

// ProxySock socks5代理
func ProxySock() *builder.APIBuilder {
    dialer, err := proxy.SOCKS5("tcp", "127.0.0.1:1123", nil, proxy.Direct)
    if err != nil {
        fmt.Fprintln(os.Stderr, "can't connect to the proxy:", err)
        os.Exit(1)
    }
    // setup a http client
    httpTransport := &http.Transport{}
    httpTransport.Dial = dialer.Dial

    // set our socks5 as the dialer
    cli := builder.NewCustomAPIBuilder(&http.Client{Transport: httpTransport, Timeout: time.Second * 60})
    return cli
}

/*
@title        : UpString
@desc         : 字符串大写
@auth         : small_ant                   time(2021/08/04 11:52:33)
@param        : b string                           ``
@return       : b string                            ``
*/
func UpString(s string) string {
    return strings.ToUpper(s)
}
