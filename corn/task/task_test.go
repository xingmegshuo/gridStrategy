/***************************
@File        : task_test.go
@Time        : 2021/08/10 16:54:47
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 抓取货币价格测试
****************************/

package job

import (
	"fmt"
	"testing"
)

func TestCraw(t *testing.T) {
    fmt.Println("testing start ...")
    xhttpCraw("https://api.huobi.de.com/market/tickers")
}
