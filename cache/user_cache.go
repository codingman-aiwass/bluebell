package cache

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"github.com/goburrow/cache"
	"time"
)

type userCache struct {
	cache cache.LoadingCache
}

var UserCache = newUserCache()

func newUserCache() *userCache {
	return &userCache{
		cache: cache.NewLoadingCache(
			func(key cache.Key) (value cache.Value, e error) {
				value = mysql_repo.UserRepository.Get(sqls.DB(), key2Int64(key))
				if value == nil {
					e = ERROR_DATA_NOT_EXISTS
				}
				return
			},
			cache.WithMaximumSize(1000),
			cache.WithExpireAfterAccess(30*time.Minute),
		),
	}
}

func (c *userCache) Get(userId int64) *models.User {
	if userId <= 0 {
		return nil
	}
	val, err := c.cache.Get(userId)
	if err != nil {
		return nil
	}
	return val.(*models.User)
}

func (c *userCache) Invalidate(userId int64) {
	c.cache.Invalidate(userId)
}
