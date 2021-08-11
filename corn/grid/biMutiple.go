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
		if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback && t.RealGrids[i].AmountSell.Cmp(decimal.Decimal{}) != 1 {
			err := t.WaitSellLimit(price, rate*100, g.AmountBuy, g.Id)
			if err != nil {
				log.Printf("一单一单卖出, grid number: %d, err: %s", g.Id, err)
				time.Sleep(time.Second * 5)
				return err
			}
		}
	}
	return nil
}

func (t *ExTrader) SetupBeMutiple(price decimal.Decimal, reduce float64, rate float64) error {
	for _, g := range t.RealGrids {
		win, _ := g.AmountBuy.Mul(price).Sub(g.TotalBuy).Div(g.TotalBuy).Float64()
		if win*100 > t.arg.Stop && reduce*100 > t.arg.Callback && g.AmountSell.Cmp(decimal.Decimal{}) != 1 {
			err := t.WaitSell(price, t.SellCount(g.AmountBuy), rate*100, g.Id-1)
			if err != nil {
				log.Printf("一单一单卖出, grid number: %d, err: %s", g.Id, err)
				time.Sleep(time.Second * 5)
				return err
			} else {
				t.base = t.base - 1
				t.Tupdate()
				continue
			}
		}
	}

	return nil
}

func (t *ExTrader) HaveOver() bool {
	for _, v := range t.RealGrids {
		if v.AmountSell.Cmp(decimal.Decimal{}) == 1 {
			return false
		}
	}
	return true
}
