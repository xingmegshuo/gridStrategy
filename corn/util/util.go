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

// NewApi 现货api  目前火币
func NewApi(c *Config) goex.API {
    api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).
        Endpoint(c.Host).ClientID(c.ClientID).HttpTimeout(time.Second * 3)
    // api := ProxySock().APIKey(c.APIKey).APISecretkey(c.Secreet).
    //     Endpoint(c.Host).ClientID(c.ClientID)
    // fmt.Println(fmt.Sprintf("%+v", api), api.GetHttpClient())
    var cli goex.API
    switch c.Name {
    case "币安":
        cli = api.Build(goex.BINANCE) //创建现货api实例
    default:
        cli = api.Build(goex.HUOBI_PRO) //创建现货api实例
    }
    return cli
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
    httpClient := &http.Client{Transport: httpTransport, Timeout: time.Second * 3}
    // set our socks5 as the dialer
    cli := builder.NewCustomAPIBuilder(httpClient)
    return cli
}
