package cache

import (
	"bluebell/settings"
	"context"
	"github.com/coocood/freecache"
	"strconv"
	"sync"
)

var freeCache *Cache

type Cache struct {
	cache *freecache.Cache
	mutex sync.RWMutex
}

// NewCache 创建一个新的缓存实例
func NewCache(config *settings.FreeCacheConfig) {
	freeCache = &Cache{
		cache: freecache.NewCache(config.CacheSize),
	}
}

// Increment 方法实现计数器的增量
func (c *Cache) Increment(key []byte, increment int) (int, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// 尝试获取当前值
	value, err := c.cache.Get(key)
	if err != nil {
		// 如果获取失败（键不存在），初始值为 0
		value = []byte("0")
	}

	// 将当前值转换为整数并递增
	currentValue, _ := strconv.Atoi(string(value))
	currentValue += increment

	// 更新缓存中的值
	err = c.cache.Set(key, []byte(strconv.Itoa(currentValue)), 0)
	if err != nil {
		return 0, err
	}

	return currentValue, nil
}

func (c *Cache) StoreFollowRequest(userId, targetUserId int64, action int8) (err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	fansKey := strconv.FormatInt(targetUserId, 10)
	followKey := strconv.FormatInt(userId, 10)

	if action == -1 {
		// 说明是取关操作
		// 需要给目标用户的粉丝人数和用户的关注人数-1
		_, err = freeCache.Increment([]byte(fansKey), -1)
		_, err = freeCache.Increment([]byte(followKey), -1)

	} else {
		// 说明是关注操作
		// 需要给目标用户的粉丝人数和用户的关注人数+1
		_, err = freeCache.Increment([]byte(fansKey), 1)
		_, err = freeCache.Increment([]byte(followKey), 1)
	}
	return err
}

func (c *Cache) BatchProcessFollow(ctx context.Context) {

	c.mutex.Lock()
	defer c.mutex.Unlock()

}
