/***************************
@File        : bimul.go
@Time        : 2021/07/24 16:50:23
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 智乘方策略  市价/现价 智乘方为整体出仓方式
****************************/

package grid

import (
	"time"

	"github.com/shopspring/decimal"
)

// setupBi bi乘方策略 卖出
func (t *Trader) setupBi(win float64, reduce float64, price decimal.Decimal) error {
	if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback {
		log.Println("盈利卖出", t.u.ObjectId, "当前价格:", price)
		err := t.WaitSell(price, win*100)
		if err != nil {
			log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
			time.Sleep(time.Second * 5)
			return err
		}
	}
	return nil
}

// setupBi bi乘方策略 卖出
func (t *ExTrader) setupBi(win float64, reduce float64, price decimal.Decimal) error {
	if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback {
		log.Println("智乘方盈利卖出", t.u.ObjectId, "当前价格:", price, "回降:", reduce, t.arg.Callback)
		t.canBuy = false
		err := t.WaitSell(price, t.SellCount(t.CountHold()), win*100, 0)
		if err != nil {
			log.Printf("error when setupGridOrders, grid number: %d, err: %s", t.base, err)
			time.Sleep(time.Second * 5)
			return err
		} else {
			t.over = true
			t.centMoney = true
		}
	}
	return nil
}
