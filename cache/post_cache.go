package cache

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"github.com/goburrow/cache"
	"time"
)

type postCache struct {
	cache cache.LoadingCache
}

var PostCache = newPostCache()

func newPostCache() *postCache {
	return &postCache{
		cache: cache.NewLoadingCache(
			func(key cache.Key) (value cache.Value, err error) {
				value = mysql_repo.PostRepository.Get(sqls.DB(), key2Int64(key))
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

func (c *postCache) Get(postId int64) *models.Post {
	if postId <= 0 {
		return nil
	}
	val, err := c.cache.Get(postId)
	if err != nil {
		return nil
	}
	return val.(*models.Post)
}

func (c *postCache) Invalidate(postId int64) {
	c.cache.Invalidate(postId)
}
