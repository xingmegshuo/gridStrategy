/***************************
@File        : myCache.go.go
@Time        : 2021/7/2 12:50
@AUTHOR      : small_ant
@Email       : xms.chnb@gmail.com
@Desc        :
****************************/

package model

import (
	"time"
)

// SetCache 设置缓存
func SetCache(name string, data interface{}, t time.Duration) {
	if !CheckCache(name) {
		CacheDB.Set(ctx, name, data, t)
	}
}

// CheckCache 查看缓存是否过期
func CheckCache(name string) bool {
	n, err := CacheDB.Exists(ctx, name).Result()
	if err != nil {
		return false
	}
	if n > 0 {
		ttl, _ := CacheDB.TTL(ctx, name).Result()
		if ttl != 0 {
			return true
		}
	}
	return false
}

// GetCache 获取数据
func GetCache(name string) string {
	if CheckCache(name) {
		data, _ := CacheDB.Get(ctx, name).Result()
		return data
	}
	return ""
}
