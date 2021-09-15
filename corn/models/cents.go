/***************************
@File        : cents.go
@Time        : 2021/08/27 16:45:49
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 盈利分红
****************************/

package model

import (
    "fmt"
    "time"
)

// CentsUser 用户分红
func CentsUser(money float64, uId float64, from interface{}) {
    t := GetAccount(uId)
    realMoney := money * 0.24 // 分红盈利
    log.Printf("用户%v;策略%v;盈利%v;分红%v;套餐余额%v", uId, from, money, realMoney, t)

    if money > 0 { // 盈利
        var (
            u = map[string]interface{}{}
        )
        tx := UserDB                                                                                                                  // 使用事务
        tx.Raw("select `id`,`profit_mine_amount`,`team_min_amount`,`level`,`inviter_id` from db_customer where id = ?", uId).Scan(&u) // 获取用户

        // 修改盈利
        ChangeAmount(money, &u, tx, true)
        // 扣除分红金额，写日志
        ownLog := &AmountLog{
            FlowType:       62,
            CustomerId:     uId,
            FromCustomerId: float64(from.(int64)),
            Direction:      2,
            CoinId:         2,
            Amount:         realMoney,
            BeforeAmount:   t,
            AfterAmount:    t - realMoney,
            Hash:           "000",
            Remark:         "盈利扣款",
            CreateTime:     time.Now().Unix(),
        }
        log.Printf("用户%v之前预充值余额:%v----之后预充值余额:%v", uId, ownLog.BeforeAmount, ownLog.AfterAmount)
        ownLog.Write(UserDB)
        tx.Table("db_customer").Where("id = ? ", uId).Update("meal_amount", ownLog.AfterAmount)

        // 大于%5 分红
        if t > money*0.05 {
            // %5
            fiveRate := money * 0.04
            realMoney -= fiveRate
            SaveMoney(1, fiveRate)

            // 股东 20%
            boss := money * 0.2 * 0.2
            realMoney -= boss
            eightyFive := boss
            SaveMoney(2, eightyFive)

            // 市场
            market := money * 0.2 * 0.8
            baseLevel := u["level"].(uint8)
            levelMoney := realMoney

            // 创始合伙人
            partner := levelMoney * 0.1
            SaveMoney(3, partner)
            // levelMoney -= partner

            // 股东
            orther := levelMoney * 0.2
            SaveMoney(4, orther)
            // levelMoney -= orther
            log.Printf("用户:%v;市场:%v;存储:%v;boos:%v;创始合伙人:%v;股东:%v", uId, market, fiveRate, boss, partner, orther)

            f := true
            sameLevel := float64(0)
            for {
                var myMoney float64
                if u["inviter_id"].(uint32) > 0 && realMoney > 0 {
                    // time.Sleep(time.Second)
                    tx.Raw("select `id`,`team_amount`,`team_min_amount`,`level`,`inviter_id`,`is_meal` from db_customer where id = ?", u["inviter_id"]).Scan(&u) // 获取用户
                    // ChangeAmount(money, &u, tx, true)
                    thisLevel := u["level"].(uint8)
                    var thisLog = &AmountLog{
                        FlowType:       float64(59),
                        CoinId:         float64(2),
                        Direction:      1,
                        Hash:           "",
                        Remark:         "级差分红",
                        FromCustomerId: uId,
                        CustomerId:     float64(u["id"].(int32)),
                        BeforeAmount:   GetAccountCach(float64(u["id"].(int32))),
                        CreateTime:     time.Now().Unix(),
                    }

                    if thisLevel >= baseLevel && realMoney > 0 {
                        // baseLevel = thisLevel // 上级的vip等级，下次分红vip必须大于此等级
                        if thisLevel == 1 && baseLevel != 1 {
                            myMoney = levelMoney * 0.2
                        }
                        if thisLevel == 2 {
                            // l.Println("我分25%")
                            if thisLevel-baseLevel == 1 {
                                myMoney = levelMoney * 0.1
                            } else if thisLevel-baseLevel == 2 {
                                myMoney = levelMoney * 0.3
                            }
                        }
                        if thisLevel == 3 {
                            // l.Println("我要30%")
                            if thisLevel-baseLevel == 1 {
                                myMoney = levelMoney * 0.1
                            } else if thisLevel-baseLevel == 2 {
                                myMoney = levelMoney * 0.2
                            } else if thisLevel-baseLevel == 3 {
                                myMoney = levelMoney * 0.4
                            }
                        }
                        if thisLevel == 4 {
                            // l.Println("我要40%")
                            if thisLevel-baseLevel == 1 {
                                myMoney = levelMoney * 0.1
                            } else if thisLevel-baseLevel == 2 {
                                myMoney = levelMoney * 0.2
                            } else if thisLevel-baseLevel == 3 {
                                myMoney = levelMoney * 0.3
                            } else if thisLevel-baseLevel == 4 {
                                myMoney = levelMoney * 0.5
                            }
                        }
                        if thisLevel == 5 {
                            // l.Println("我要50%")
                            if f {
                                if thisLevel-baseLevel == 1 {
                                    myMoney = levelMoney * 0.05
                                } else if thisLevel-baseLevel == 2 {
                                    myMoney = levelMoney * 0.15
                                } else if thisLevel-baseLevel == 3 {
                                    myMoney = levelMoney * 0.25
                                } else if thisLevel-baseLevel == 4 {
                                    myMoney = levelMoney * 0.35
                                } else {
                                    myMoney = levelMoney * 0.55
                                }
                                f = false
                                sameLevel = myMoney
                            } else {
                                myMoney = sameLevel * 0.1 //平级
                                baseLevel = 7
                            }
                        }
                        if thisLevel == 6 {
                            // log.Println(thisLevel, baseLevel)
                            // l.Println("我要60%")
                            if baseLevel == 7 {
                                baseLevel = 5
                            }
                            if thisLevel-baseLevel == 1 {
                                myMoney = levelMoney * 0.05
                            } else if thisLevel-baseLevel == 2 {
                                myMoney = levelMoney * 0.1
                            } else if thisLevel-baseLevel == 3 {
                                myMoney = levelMoney * 0.2
                            } else if thisLevel-baseLevel == 4 {
                                myMoney = levelMoney * 0.3
                            } else if thisLevel-baseLevel == 5 {
                                myMoney = levelMoney * 0.4
                            } else if thisLevel-baseLevel == 6 {
                                myMoney = levelMoney * 0.6
                            }
                        }
                        if myMoney > 0 {
                            thisLog.Amount = myMoney
                            thisLog.AfterAmount = thisLog.BeforeAmount + myMoney
                            log.Println(fmt.Sprintf("分红金额:%2f---用户:%v---我的vip:%v;之前账户余额%v;现在账户余额%v", myMoney, u["id"], thisLevel, thisLog.BeforeAmount, thisLog.AfterAmount))
                            thisLog.Write(UserDB)
                            tx.Table("db_coin_amount").Where("customer_id = ? and coin_id = 2", thisLog.CustomerId).Update("amount", thisLog.AfterAmount)
                            if baseLevel != 7 {
                                baseLevel = thisLevel
                            }
                            realMoney -= myMoney
                        }
                    }
                } else {
                    break
                }
            }
            log.Printf("剩余金额%v", realMoney-partner-orther)
        }
    }
}

// SaveMoney 需要存起来的资金
func SaveMoney(id interface{}, money float64) {
    var oldAmount float64
    UserDB.Raw("select amount from db_customer_statis where id = ?", id).Scan(&oldAmount)
    newAmount := oldAmount + money
    log.Printf("类型%v;本次获取金额%v;之前数据金额%v;之后数据金额%v", id, money, oldAmount, newAmount)
    UserDB.Exec("update db_customer_statis set amount = ? where id = ?", newAmount, id)
}
