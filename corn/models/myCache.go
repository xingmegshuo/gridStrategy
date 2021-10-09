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

	"github.com/go-redis/redis/v8"
)

// SetCache 设置缓存
func SetCache(name string, data interface{}, t time.Duration) {
	if !CheckCache(name) {
		CacheDB.Set(ctx, name, data, t)
	}
}

func AddCache(name string, data ...*redis.Z) {
	CacheDB.ZAdd(ctx, name, data...)
	// fmt.Println(res.Result())
	// fmt.Println(name)
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

// Del 删除
func Del(name string) {
	CacheDB.Del(ctx, name)
}
// ListCacheGet 从缓存中获取
func ListCacheGet(name string, op *redis.ZRangeBy) *redis.StringSliceCmd {
	return CacheDB.ZRangeByScore(ctx, name, op)
}
// ListCacheRm 从缓存中删除
func ListCacheRm(name string, min string, max string) {
	CacheDB.ZRemRangeByScore(ctx, name, min, max)
}
// ListCacheAddOne 向缓存中添加一个
func ListCacheAddOne(name string, data *redis.Z) {
	CacheDB.ZAdd(ctx, name, data)
}
