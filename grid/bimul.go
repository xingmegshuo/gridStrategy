/***************************
@File        : bimul.go
@Time        : 2021/07/24 16:50:23
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 智乘方策略  市价/现价 智乘方为整体出仓方式
****************************/

package grid

import (
	"context"
	"runtime"
	"time"
	model "zmyjobs/models"

	"github.com/shopspring/decimal"
)

// setupBi bi乘方策略
func (t *Trader) setupBi(ctx context.Context) {
    // 计数
    count := 0
    t.GetLastPrice()

    log.Println("上次交易:", t.last, "基础价格:", t.basePrice, "投入金额:", t.pay, "当前持仓:", t.amount, "---------策略开始", "用户:", t.u.ObjectId)
    var (
        low  = t.last
        high = t.last
    )
    for {
        count++
        time.Sleep(time.Millisecond * 500)                 // 间隔0.5秒查询
        t.GetMoeny()                                       // 获取当前money和持仓
        price, err := t.ex.huobi.GetPrice(t.symbol.Symbol) //获取当前价格
        if err != nil {
            t.ErrString = err.Error()
            return
        }
        high, low = ChangeHighLow(price)
        // 计算盈利
        win := float64(0)
        if t.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
            win, _ = (price.Mul(t.amount).Sub(t.pay)).Div(t.pay).Float64() // 计算盈利 当前价值-投入价值
        }
        reduce, _ := high.Sub(price).Div(t.last).Float64() // 当前回降
        top, _ := price.Sub(low).Div(t.last).Float64()     // 当前回调
        die, _ := t.last.Sub(price).Div(t.last).Float64()  // 当前跌幅
        // 输出日志
        if count%1200 == 0 {
            log.Println("当前盈利", win*100, "单数:", t.base, "下跌:", die*100, "上次交易:", t.last, "当前价格：",
                price, "持仓:", t.amount, "最高价:", high, "最低价:", low, "回降比例:", reduce*100, "回调比例:", top*100)
        }
        select {
        case <-ctx.Done():
            log.Println("close get price ", t.u.ObjectId)
            runtime.Goexit()
        case <-model.SellCh:
            t.AllSellMy()
            err := t.WaitSell(price, win*100)
            if err != nil {
                time.Sleep(time.Second * 5)
                continue
            } else {
                t.over = true
                break
            }
        default:
        }

        //  第一单 进场时机无所谓
        if t.base == 0 && !t.arg.StopBuy {
            if count >= 10 {
                t.OrderOver = false
                log.Println("开始进场首次买入:---价格", price, "---数量:", t.grids[t.base].AmountBuy, "---money", t.grids[t.base].TotalBuy)
                err := t.WaitBuy(price)
                if err != nil {
                    log.Printf("买入错误: %d, err: %s", t.base, err)
                    time.Sleep(time.Second * 5)
                    continue
                } else {
                    high = price
                    low = price
                }
            }
        }
        // 后续买入按照跌幅+回调来下单
        if 0 < t.base && t.base < len(t.grids) && !t.arg.StopBuy {
            if die*100 >= t.grids[t.base].Decline && top*100 >= t.arg.Reduce {
                t.OrderOver = false
                log.Println(t.base, "买入:", price, t.grids[t.base].AmountBuy, "下降幅度:", die, "价格:", t.grids[t.base].Price, "----------", price.Cmp(t.grids[t.base].Price))
                err := t.WaitBuy(price)
                if err != nil {
                    log.Printf("买入错误: %d, err: %s", t.base, err)
                    time.Sleep(time.Second * 5)
                    continue
                } else {
                    high = price
                    low = price
                }
            }
        }

        //  止盈 t.arg.Stop
        if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback {
            log.Println("盈利卖出", t.u.ObjectId)
            err := t.WaitSell(price, win*100)
            if err != nil {
                log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
                time.Sleep(time.Second * 5)
                continue
            }
        }

        //  如果不相等更新
        if t.base != t.u.Base {
            t.Tupdate()
        }
        if t.over {
            log.Println("任务结束", t.u.ObjectId)
            break
        }
    }
}
