/***************************
@File        : util.go
@Time        : 2021/07/28 15:08:45
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 工具
****************************/

package util

import (
	"github.com/nntaoli-project/goex"
	"github.com/nntaoli-project/goex/builder"
)

type Config struct {
    Name    string //平台名称
    Host    string // url
    APIKey  string // 秘钥
    Secreet string // 秘钥
}

// NewApi 现货api  目前火币
func NewApi(c *Config) goex.API {
    api := builder.DefaultAPIBuilder.APIKey(c.APIKey).APISecretkey(c.Secreet).
        Endpoint(c.Host).Build(goex.HUOBI_PRO) //创建现货api实例
    return api
}
