/***************************
@File        : biMutiple.go
@Time        : 2021/07/28 11:31:40
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 智多元分开卖出
****************************/

package grid

import (
	"time"

	"github.com/shopspring/decimal"
)

func (t *Trader) SetupBeMutiple(price decimal.Decimal, reduce float64, rate float64) error {
	for i, g := range t.RealGrids {
		win, _ := t.RealGrids[i].AmountBuy.Mul(price).Sub(t.RealGrids[i].TotalBuy).Div(t.RealGrids[i].TotalBuy).Float64()
		if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback && t.RealGrids[i].AmountSell.Cmp(decimal.Decimal{}) == 1 {
			err := t.WaitSellLimit(price, rate*100, g.AmountBuy)
			if err != nil {
				log.Printf("一单一单卖出, grid number: %d, err: %s", g.Id, err)
				time.Sleep(time.Second * 5)
				return err
			}
		}
	}
	return nil
}
