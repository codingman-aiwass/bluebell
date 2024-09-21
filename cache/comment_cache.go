package cache

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"github.com/goburrow/cache"
	"time"
)

type commentCache struct {
	cache cache.LoadingCache
}

var CommentCache = newCommentCache()

func newCommentCache() *commentCache {
	return &commentCache{
		cache: cache.NewLoadingCache(
			func(key cache.Key) (value cache.Value, err error) {
				value = mysql_repo.CommentRepository.Get(sqls.DB(), key2Int64(key))
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

func (c *commentCache) Get(commentId int64) *models.Comment {
	if commentId <= 0 {
		return nil
	}
	val, err := c.cache.Get(commentId)
	if err != nil {
		return nil
	}
	return val.(*models.Comment)
}

func (c *commentCache) Invalidate(commentId int64) {
	c.cache.Invalidate(commentId)
}
