package redis_repo

import (
	"strconv"
)

// AddPostCollectionNumber 修改post收藏数
func AddPostCollectionNumber(postId, val int64) (err error) {
	cmd := rdb.ZIncrBy(ctx, getKey(KeyPostCollectionZset), float64(val), strconv.FormatInt(postId, 10))
	//zap.L().Info(fmt.Sprintf("AddPostCollectionNumber execute command: %s", cmd.String()))
	return cmd.Err()
}
