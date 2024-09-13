package cache

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"github.com/goburrow/cache"
	"time"
)

type communityCache struct {
	cache cache.LoadingCache
}

var CommunityCache = newCommunityCache()

func newCommunityCache() *communityCache {
	return &communityCache{
		cache: cache.NewLoadingCache(
			func(key cache.Key) (value cache.Value, err error) {
				value = mysql_repo.CommunityRepository.Get(sqls.DB(), key2Int64(key))
				if value == nil {
					err = ERROR_DATA_NOT_EXISTS
				}
				return
			},
			cache.WithMaximumSize(1000),
			cache.WithExpireAfterAccess(30*time.Minute),
		),
	}
}

func (c *communityCache) Get(communityId int64) *models.Community {
	if communityId <= 0 {
		return nil
	}
	val, err := c.cache.Get(communityId)
	if err != nil {
		return nil
	}
	return val.(*models.Community)
}

func (c *communityCache) Invalidate(communityId int64) {
	c.cache.Invalidate(communityId)
}
