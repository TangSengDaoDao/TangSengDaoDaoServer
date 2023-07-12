package cache

import "time"

// Cache 缓存接口
type Cache interface {
	// Set 设置key value
	Set(key string, value string) error
	// 删除key
	Delete(key string) error
	// SetAndExpire  设置key value 并支持过期时间
	SetAndExpire(key string, value string, expire time.Duration) error
	// 获取key对应的值
	Get(key string) (string, error)
}
