package redis_repo

import (
	"bluebell/models"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strconv"
	"time"
)

func CheckPostExpired(postId string) (bool, error) {
	t, err := rdb.ZScore(ctx, getKey(KeyPostTimeZset), postId).Result()
	if err != nil {
		zap.L().Error("redis_repo get post time error", zap.Error(err))
		return true, err
	}
	if float64(time.Now().Unix())-t > float64(POST_VALID_TIME) {
		return true, nil
	}
	return false, nil
}

// 获取用户对帖子的评分

func GetUser2PostScore(userId, postId string) (score float64, err error) {
	score, err = rdb.ZScore(ctx, getKey(KeyPostVotedZset+":"+postId), userId).Result()
	if errors.Is(err, redis.Nil) {
		score = 0
		err = nil
	}
	return
}

func SetUser2PostScore(userId, postId string, score float64) (err error) {
	err = rdb.ZAdd(ctx, getKey(KeyPostVotedZset+":"+postId), redis.Z{Score: score, Member: userId}).Err()
	return
}

func getPostScore(userId, postId string) (score float64, err error) {
	score, err = rdb.ZScore(ctx, getKey(KeyPostScoreZset+":"+postId), userId).Result()
	return
}

func SetPostScore(postId string, score float64) (err error) {
	err = rdb.ZIncrBy(ctx, getKey(KeyPostScoreZset), score, postId).Err()
	return
}
func SetPostVote(postId string, vote float64) (err error) {
	err = rdb.ZIncrBy(ctx, getKey(KeyPostVoteZset), vote, postId).Err()
	return
}

func SetPostDevote(postId string, vote float64) (err error) {
	err = rdb.ZIncrBy(ctx, getKey(KeyPostDevoteZset), vote, postId).Err()
	return
}

func CreatePost(post *models.Post) (err error) {
	pipe := rdb.TxPipeline()
	// 创建帖子的time和score记录
	pipe.ZAdd(ctx, getKey(KeyPostTimeZset), redis.Z{Score: float64(time.Now().Unix()), Member: post.PostId})
	pipe.ZAdd(ctx, getKey(KeyPostScoreZset), redis.Z{Score: 0, Member: post.PostId})
	pipe.SAdd(ctx, getKey(KeyCommunityPrefix+strconv.FormatInt(post.CommunityID, 10)), post.PostId)
	_, err = pipe.Exec(ctx)
	return
}

func GetPostIds(param *models.ParamPostList) (postIds []string, err error) {
	//zinterstore out 2 bluebell:community:1 bluebell:post:time aggregate max
	//zrange out 0 -1 withscores

	var key string
	if param.Order == models.OrderByTime {
		key = getKey(KeyPostTimeZset)
	} else if param.Order == models.OrderByScore {
		key = getKey(KeyPostScoreZset)
	}
	// 首先需要判断是否需要做交集操作，生成一个新的zset，然后从这个zset中获取id
	var target_key = key + ":" + param.CommunityId
	flag := false
	if len(param.CommunityId) > 0 {
		// 先查看redis中是否有过这个键了，有过就不用再算了
		flag = true
		val := rdb.Exists(ctx, target_key).Val()
		if val == int64(1) {
			// 说明存在，不用再次计算

		} else {
			// 说明需要计算
			store := redis.ZStore{
				Keys:      []string{getKey(KeyCommunityPrefix + param.CommunityId), key},
				Aggregate: "max",
			}
			rdb.ZInterStore(ctx, target_key, &store)
		}
	}
	start := (param.Page - 1) * param.Size
	end := start + param.Size
	if flag {
		postIds, err = rdb.ZRevRange(ctx, target_key, int64(start), int64(end)).Result()

	} else {
		postIds, err = rdb.ZRevRange(ctx, key, int64(start), int64(end)).Result()
	}
	return
}

func GetPostVote(ids []string, vote string) (result []int64, err error) {
	pipe := rdb.TxPipeline()
	for _, id := range ids {
		key := getKey(KeyPostVotedZset + ":" + id)
		pipe.ZCount(ctx, key, vote, vote)
	}
	cmders, err := pipe.Exec(ctx)
	if err != nil {
		zap.L().Error("query post votes from redis_repo error", zap.Error(err))
		return nil, err
	}
	result = make([]int64, 0, len(ids))

	for _, cmder := range cmders {
		val := cmder.(*redis.IntCmd).Val()
		result = append(result, val)
	}
	return result, nil
}

func DeletePostInfo(postId, communityId int64) (err error) {
	pipe := rdb.TxPipeline()
	pipe.ZRem(ctx, getKey(KeyPostVoteZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostDevoteZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostScoreZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostCommentZset), postId)
	pipe.ZRem(ctx, fmt.Sprintf("%s:%d", getKey(KeyPostScoreZset), communityId), postId)
	_, err = pipe.Exec(ctx)
	return

}
