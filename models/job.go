/***************************
@File        : job.go.go
@Time        : 2021/7/1 18:12
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package model

import (
	"errors"

	"gorm.io/gorm"
)

// Job 任务
type Job struct {
	gorm.Model
	Type   string // 任务类型
	Status string // 任务状态
	Name   string // 任务名称
	Count  int    // 任务执行次数
	Spec   string // 任务配置
}

// NewJob 新建任务
func NewJob(types string, name string, spec string) *Job {
	job := Job{Type: types, Name: name, Spec: spec}
	j := GetJob(&job)
	return j
}

// GetJob 检测数据库是否存在,不存在写入，存在返回
func GetJob(j *Job) *Job {
	var res Job
	result := DB.Where(&Job{Name: j.Name, Spec: j.Spec}).First(&res)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		log.Println("添加任务", j.Name)
		DB.Create(&j)
		return j
	}
	return &res
}

// UpdateJob 更新数据库状态
func (j *Job) UpdateJob() {
	DB.First(&Job{}, j.ID).Updates(j)
}
