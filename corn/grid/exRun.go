package grid

import (
	"context"
	"encoding/json"
	"runtime"
	"time"
	model "zmyjobs/corn/models"
	"zmyjobs/goex"

	"github.com/shopspring/decimal"
)

// RunEx 策略执行入口
func RunEx(ctx context.Context, u model.User) {
	//var ctx *cli.Context
	status := 0
	for {
		time.Sleep(time.Millisecond * 500)
		select {
		case <-ctx.Done():
			log.Printf("%v停止交易", u.ObjectId)
			return
		default:
		}
		if status == 0 {
			status = 1
			g := NewExStrategy(u)
			if g == nil || len(g.grids) != int(u.Number) {
				u.IsRun = -10
				u.Error = "无法使用api解析"
				u.Update()
				GridDone <- u.ObjectId
			} else {
				g.u = u
				// fmt.Println("ggggg")
				go g.Trade(ctx)
			}
		} else {
			runtime.Gosched()
		}
	}
}

// DelEx 删除清仓操作
func DelEx(u model.User) {
	g := NewExStrategy(u)
	if u.Base > 0 && g.CountHold().Cmp(decimal.Decimal{}) == 1 {
		price, err := g.goex.GetPrice()
		if err == nil {
			win := float64(0)
			if g.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
				if g.arg.Crile == 4 {
					win, _ = (g.pay.Sub(price.Mul(g.amount))).Div(g.pay).Float64() // 计算盈利 当前价值-投入价值
				} else {
					win, _ = (price.Mul(g.amount).Sub(g.pay)).Div(g.pay).Float64() // 计算盈利 当前价值-投入价值
				}
			}
			err = g.WaitSell(price, g.SellCount(g.CountHold()), win*100, 0)
			if err == nil {
				res := g.CalCulateProfit()
				p, _ := res.Float64()
				g.u.IsRun = 2
				g.u.BasePrice = p
				g.u.RealGrids = "***"
				g.u.Update()
				model.DB.Exec("update users set base = 0 where object_id = ?", g.u.ObjectId)
				log.Println("实际的买入信息清空,用户单数清空", g.u.ObjectId)
				model.LogStrategy(g.arg.CoinId, g.goex.symbol.Category, g.u.Name, g.u.ObjectId,
					g.u.Custom, g.CountBuy(), g.cost, g.arg.IsHand, res, 0)
				log.Println("任务结束,删除平仓或者暂停平仓", g.u.ObjectId)
			}
		}
	}
}

//* NewGrid 实例化对象，并验证api key的正确性
func NewExStrategy(u model.User) (ex *ExTrader) {
	arg := model.StringArg(u.Arg)
	grid, _ := model.SourceStrategy(u, false)
	var realGrid []model.Grid
	_ = json.Unmarshal([]byte(u.RealGrids), &realGrid)
	symbol := model.StringSymobol(u.Symbol)
	if arg.Level == nil {
		symbol.Lever = 10
	} else {
		if arg.Level.(float64) > 0 {
			symbol.Lever = arg.Level.(float64)
		} else {
			symbol.Lever = 10
		}
	}
	ex = &ExTrader{
		grids:     *grid,
		arg:       &arg,
		RealGrids: realGrid,
		goex:      NewEx(&symbol),
	}
	if ex.goex.Future != nil {
		if u.Future == 2 || u.Future == 4 {
			if !ex.goex.Future.ChangeLever(ex.goex.Currency, goex.SWAP_CONTRACT) {
				log.Println("修改杠杆倍数出错", symbol.Lever)
				return nil
			}
		}
		if u.Future == 1 || u.Future == 3 {
			if !ex.goex.Future.ChangeLever(ex.goex.Currency, goex.SWAP_USDT_CONTRACT) {
				log.Println("修改杠杆倍数出错", symbol.Lever)
				return nil
			}
		}
	} else if u.Future > 0 {
		return nil
	}

	log.Printf("用户:%v;交易对:%v;期货标识:%v;策略类型:%v;实际交易信息:%v", u.ObjectId, ex.goex.Currency, u.Future, ex.arg.Crile, ex.RealGrids)
	return
}

// Trade 创建websocket 并执行策略中的交易任务
func (t *ExTrader) Trade(ctx context.Context) {
	//_ = t.Print()
	c := 0
	for {
		time.Sleep(time.Millisecond * 500)
		select {
		case <-ctx.Done():
			log.Printf("%v结束交易", t.u.ObjectId)
			return
		default:
			for i := 0; i < 1; i++ {
				if c == 0 {
					c = 1
					log.Printf("尝试获取%v用户账户数据，校验余额，api 等信息正确性", t.u.ObjectId)
					if err := t.ReBalance(ctx); err != nil {
						log.Printf("校验%v账户余额不足够，策略不开始----", t.u.ObjectId)
						t.u.IsRun = -10
						t.u.Error = err.Error()
						log.Println(err, t.u.ObjectId)
						t.u.Update()
						// 执行报错就关闭
						GridDone <- t.u.ObjectId
					} else {
						t.setupGridOrders(ctx)
						if t.ErrString != "" {
							log.Println("网络链接问题：", t.u.ObjectId)
							t.u.IsRun = -10
							t.u.Error = t.ErrString
							t.u.Update()
							model.StrategyError(t.u.ObjectId, t.ErrString)
							// 执行报错就关闭
							GridDone <- t.u.ObjectId
						} else if t.over && t.ErrString == "" {
							// 策略执行完毕 to do 计算盈利
							log.Println("策略一次执行完毕:", t.u.ObjectId, "盈利:", t.CalCulateProfit())
							res := t.CalCulateProfit()
							p, _ := res.Float64()
							// 盈利ctx
							if t.arg.Crile >= 2 && !t.automatic {
								t.u.IsRun = 100
							} else {
								t.u.IsRun = 2
							}
							t.u.BasePrice = p
							t.u.RealGrids = "***"
							t.u.IsRun = 1000
							t.u.Update()
							model.DB.Exec("update users set base = 0 where object_id = ?", t.u.ObjectId)
							log.Println("实际的买入信息清空,用户单数清空", t.u.ObjectId)
							var status interface{}
							if t.arg.StopFlow {
								status = 2
							} else if !t.automatic {
								if t.arg.Crile >= 2 {
									status = 1
								} else {
									status = 2
								}
							} else if t.automatic {
								status = 0
							}
							model.LogStrategy(t.arg.CoinId, t.goex.symbol.Category, t.u.Name, t.u.ObjectId,
								t.u.Custom, t.CountBuy(), t.cost, t.arg.IsHand, res, status)
							log.Printf("%v任务结束;是否用户主动结束:%v;策略类型:%v", t.u.ObjectId, t.automatic, t.arg.IsHand)
						}
					}
				} else {
					runtime.Gosched()
				}
			}
		}
	}
}

// setupGridOrders 测试
func (t *ExTrader) setupGridOrders(ctx context.Context) {
	errorCount := 0
	count := 0
	t.GetLastPrice()
	log.Println("上次交易:", t.last, "基础价格:", t.basePrice, "投入金额:", t.pay, "当前持仓:", t.amount, "策略开始", "用户:", t.u.ObjectId, "限价启动:", t.arg.LimitHigh)
	var (
		low     = t.last
		high    = t.last
		willbuy = false
	)
	for {
		count++
		time.Sleep(time.Second * 3) // 间隔0.5秒查询
		price := model.GetPrice(model.ParseFloatString(t.arg.CoinId.(float64)))
		var u model.User
		model.DB.Raw("select * from users where object_id = ?", t.u.ObjectId).Scan(&u)
		t.arg = model.ParseStrategy(u)

		// price, err := t.goex.GetPrice()
		// if err != nil {
		// errorCount++
		// if errorCount > 2 {
		// t.ErrString = err.Error()
		// log.Println(err, t.u.ObjectId)
		// return
		// } else {
		// time.Sleep(time.Second * 3)
		// continue
		// }
		// }
		low, high = ChangeHighLow(price, high, low)
		// 计算盈利
		win := float64(0)
		if t.pay.Cmp(decimal.NewFromFloat(0)) == 1 {
			if t.arg.Crile == 4 {
				win, _ = (t.pay.Sub(price.Mul(t.amount))).Div(t.pay).Float64() // 计算盈利 当前价值-投入价值
			} else {
				win, _ = (price.Mul(t.amount).Sub(t.pay)).Div(t.pay).Float64() // 计算盈利 当前价值-投入价值
			}
		}
		reduce, _ := high.Sub(price).Div(t.last).Float64() // 当前回降
		top, _ := price.Sub(low).Div(t.last).Float64()     // 当前回调
		die, _ := t.last.Sub(price).Div(t.last).Float64()  // 当前跌幅
		if t.arg.Crile == 4 {
			die, _ = price.Sub(t.last).Div(t.last).Float64() // 当前跌幅
		}
		if count == 50 {
			log.Printf("当前盈利:%v;当前回调:%v;当前回降:%v;当前跌幅:%v;当前价格:%v", win, top, reduce, die, price)
		}

		if win < -t.arg.StopEnd {
			t.arg.AllSell = true
		}

		select {
		case <-ctx.Done():
			log.Println("close get price ", t.u.ObjectId)
			runtime.Goexit()
		// case op := <-model.OperateCh:
		// 	log.Printf("管道数据:%+v,是否相等%v,协程中的用户%v", op, op.Id == float64(t.u.ObjectId), t.u.ObjectId)
		// 	if op.Id == float64(t.u.ObjectId) {
		// 		if op.Op == 1 {
		// 			t.arg.AllSell = true
		// 			log.Printf("用户%d接收到清仓操作----", t.u.ObjectId)
		// 		}
		// 		if op.Op == 2 {
		// 			t.arg.OneBuy = true
		// 			log.Printf("用户%d接收到一键补仓----", t.u.ObjectId)
		// 		}
		// 		if op.Op == 3 {
		// 			t.arg.StopBuy = true
		// 			log.Printf("用户%d接收到停止买入----", t.u.ObjectId)
		// 		}
		// 		if op.Op == 4 {
		// 			if t.arg.StopBuy {
		// 				t.arg.StopBuy = false
		// 				log.Printf("用户%d接收到恢复买入----", t.u.ObjectId)
		// 			}
		// 		}
		// 	}
		default:
			//  第一单 进场时机无所谓
			if t.base == 0 && !t.arg.StopBuy {
				if t.arg.IsLimit && price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh).Add(decimal.NewFromFloat(1))) >= 0 &&
					price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh).Sub(decimal.NewFromFloat(1))) < 0 {
					log.Println(price.Cmp(decimal.NewFromFloat(t.arg.LimitHigh)), price, t.arg.LimitHigh, "限价启动")
					willbuy = true
				} else if !t.arg.IsLimit && count > 2 {
					time.Sleep(time.Second * 2)
					willbuy = true
				}
				if willbuy {
					log.Printf("首次买入信息:{价格:%v,数量:%v,用户:%v,钱:%v}", price, t.grids[t.base].AmountBuy, t.u.ObjectId, t.grids[t.base].TotalBuy)
					err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), 0)
					if err != nil {
						errorCount++
						if errorCount > 2 {
							log.Printf("买入错误: %d, err: %s", t.base, err)
							t.ErrString = err.Error()
							time.Sleep(time.Second * 5)
							t.over = true
						} else {
							time.Sleep(time.Second * 10)
							continue
						}
					} else {
						high = price
						low = price
						t.last = t.RealGrids[0].Price
						t.base = t.base + 1
						log.Printf("用户%v首次买入成功;交易价格%v", t.u.ObjectId, t.last)
						t.Tupdate()
					}
				}
			}
			// 后续买入按照跌幅+回调来下单
			if 0 < t.base && t.base < len(t.grids) && !t.arg.StopBuy {
				if die*100 >= t.grids[t.base].Decline && top*100 >= t.arg.Reduce {
					log.Printf("第%d买入信息:{价格:%v,数量:%v,用户:%v,钱:%v,跌幅:%v}", t.base+1, price, t.grids[t.base].AmountBuy, t.u.ObjectId, t.grids[t.base].TotalBuy, die)
					err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), die*100)
					if err != nil {
						errorCount++
						if errorCount > 2 {
							log.Printf("买入错误: %d, err: %s", t.base, err)
							t.ErrString = err.Error()
							time.Sleep(time.Second * 5)
							t.over = true
						} else {
							time.Sleep(time.Second * 10)
							continue
						}
					} else {
						high = price
						low = price
						t.last = t.RealGrids[t.base].Price
						t.base = t.base + 1
						log.Printf("用户%v第%v次买入成功;交易价格%v", t.u.ObjectId, t.base, t.last)
						t.Tupdate()
					}
				}
			}

			// 智乘方
			if t.arg.StrategyType == 1 || t.arg.StrategyType == 3 {
				if err := t.setupBi(win, reduce, price); err != nil {
					errorCount++
					if errorCount > 2 {
						log.Printf("卖出错误: %d, err: %s", t.base, err)
						t.ErrString = err.Error()
						time.Sleep(time.Second * 5)
						t.over = true
					} else {
						time.Sleep(time.Second * 10)
						continue
					}
				}
				if t.arg.AllSell {
					log.Printf("%v用户智乘方清仓-----实际操作", t.u.ObjectId)
					t.AllSellMy()
					err := t.WaitSell(price, t.SellCount(t.CountHold()), win*100, 0)
					if err != nil {
						errorCount++
						if errorCount > 2 {
							log.Printf("清仓错误: %d, err: %s", t.base, err)
							t.ErrString = err.Error()
							time.Sleep(time.Second * 5)
							t.over = true
						} else {
							time.Sleep(time.Second * 10)
							continue
						}
					} else {
						t.Tupdate()
					}
					time.Sleep(time.Second * 3)
					t.automatic = true
					t.over = true
				}
			}
			// 智多元
			if t.arg.StrategyType == 2 || t.arg.StrategyType == 4 {
				if err := t.SetupBeMutiple(price, reduce, win); err != nil {
					errorCount++
					if errorCount > 2 {
						log.Printf("买入错误: %d, err: %s", t.base, err)
						t.ErrString = err.Error()
						time.Sleep(time.Second * 5)
						t.over = true
					} else {
						time.Sleep(time.Second * 10)
						continue
					}
				}
				if t.HaveOver() {
					t.over = true
				}
				if t.arg.AllSell {
					log.Printf("%v用户智多元清仓=---实际操作", t.u.ObjectId)
					t.AllSellMy()
					for {
						for _, g := range t.RealGrids {
							err := t.WaitSell(price, t.SellCount(g.AmountBuy), win*100, g.Id-1)
							if err != nil {
								errorCount++
								if errorCount > 2 {
									log.Printf("清仓错误: %d, err: %s", t.base, err)
									t.ErrString = err.Error()
									time.Sleep(time.Second * 5)
									t.over = true
								} else {
									time.Sleep(time.Second * 10)
									continue
								}
							} else {
								t.Tupdate()
							}
						}
						if t.CountHold().Cmp(decimal.Decimal{}) < 1 {
							break
						} else {
							continue
						}
					}
					time.Sleep(time.Second * 3)
					t.automatic = true
					t.over = true
				}
			}
			// 立即买入
			if t.arg.OneBuy && t.base < len(t.grids) {
				log.Printf("%v用户一键补仓----实际操作", t.u.ObjectId)
				t.arg.OneBuy = false
				model.OneBuy(t.u.ObjectId)
				err := t.WaitBuy(price, t.grids[t.base].TotalBuy.Div(price).Round(t.goex.symbol.AmountPrecision), die*100)
				if err != nil {
					errorCount++
					if errorCount > 2 {
						log.Printf("买入错误: %d, err: %s", t.base, err)
						t.ErrString = err.Error()
						time.Sleep(time.Second * 5)
						t.over = true
					} else {
						time.Sleep(time.Second * 10)
						continue
					}
				} else {
					high = price
					low = price
					t.last = t.RealGrids[t.base].Price
					t.base = t.base + 1
					t.Tupdate()
				}
				time.Sleep(time.Second * 3)
			}
		}
		if t.over {
			log.Printf("%v用户任务结束", t.u.ObjectId)
			break
		}
	}
}

// GetLastPrice 获取上次交易价格
func (t *ExTrader) GetLastPrice() {
	if len(t.u.RealGrids) > 0 && t.base >= 1 {
		t.last = t.RealGrids[t.base-1].Price
	} else {
		t.last = t.grids[0].Price
	}
}

// Tupdate 更新数据
func (t *ExTrader) Tupdate() {
	t.u.Base = t.base
	t.u.Total = t.amount.String()
	s, _ := json.Marshal(t.grids)
	t.u.Grids = string(s)
	t.u.RealGrids = model.ToStringJson(t.RealGrids)
	t.u.Update()
}

// AllSellMy 平仓
func (t *ExTrader) AllSellMy() {
	model.OneSell(t.u.ObjectId)
	log.Println("一键平仓：", t.u.ObjectId)
	// t.arg.AllSell = false
	// t.u.Arg = model.ToStringJson(&t.arg)
	// t.u.Update()
}

func (t *ExTrader) ToPrecision(p decimal.Decimal) decimal.Decimal {
	return p.Truncate(t.goex.symbol.AmountPrecision).Round(t.goex.symbol.AmountPrecision)
}
