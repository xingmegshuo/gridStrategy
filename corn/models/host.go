/***************************
@File        : host.go
@Time        : 2021/7/3 10:57
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        : 数据连接
****************************/

package model

import "gorm.io/gorm"

// Host 数据来源
type Host struct {
	gorm.Model
	Name   string // 平台名称
	Url    string // 平台链接
	Status string // 平台状态
}

func (h *Host) Get(name string) {
	DB.Where(&Host{Name: name}).First(&h)
}

func (h *Host) Update() {
	DB.Updates(&h)
}
