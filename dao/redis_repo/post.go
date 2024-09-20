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

// GetUser2PostVoted 获取用户对该帖子的点赞/点踩情况
func GetUser2PostVoted(userId, postId string) (score float64, err error) {
	score, err = rdb.ZScore(ctx, getKey(KeyPostVotedZset+":"+postId), userId).Result()
	if errors.Is(err, redis.Nil) {
		score = 0
		err = nil
	}
	return
}

// SetUser2PostVoted 修改用户对该帖子的1点赞/点踩情况
func SetUser2PostVoted(userId, postId string, score float64) (err error) {
	err = rdb.ZAdd(ctx, getKey(KeyPostVotedZset+":"+postId), redis.Z{Score: score, Member: userId}).Err()
	return
}

// SetUser2PostVotedAndPostVoteNum 在事务中修改用户对帖子的点赞/点踩情况以及帖子的点赞/点踩统计数据
func SetUser2PostVotedAndPostVoteNum(curDirection, oDirection float64, postId, userId string) (err error) {
	pipe := rdb.TxPipeline()
	if curDirection == 1 {
		// 需要将bluebell:post:voted:postId 下该用户的记录设置为1
		pipe.ZRem(ctx, getKey(KeyPostVotedZset+":"+postId), userId)
		pipe.ZAdd(ctx, getKey(KeyPostVotedZset+":"+postId), redis.Z{Score: curDirection, Member: userId})
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteZset), 1, postId)
		if oDirection == -1 {
			// 取消点踩
			pipe.ZIncrBy(ctx, getKey(KeyPostDevoteZset), -1, postId)
		}
	} else if curDirection == 0 {
		// 需要删除bluebell:post:voted:postId 下该用户的记录
		pipe.ZRem(ctx, getKey(KeyPostVotedZset+":"+postId), userId)
		if oDirection == 1 {
			// 取消点赞
			pipe.ZIncrBy(ctx, getKey(KeyPostVoteZset), -1, postId)
		} else if oDirection == -1 {
			// 取消点踩
			pipe.ZIncrBy(ctx, getKey(KeyPostDevoteZset), -1, postId)
		}
	} else if curDirection == -1 {
		// 需要将bluebell:post:voted:postId 下该用户的记录设置为-1
		pipe.ZRem(ctx, getKey(KeyPostVotedZset+":"+postId), userId)
		pipe.ZAdd(ctx, getKey(KeyPostVotedZset+":"+postId), redis.Z{Score: curDirection, Member: userId})
		pipe.ZIncrBy(ctx, getKey(KeyPostDevoteZset), 1, postId)
		if oDirection == 1 {
			pipe.ZIncrBy(ctx, getKey(KeyPostVoteZset), -1, postId)
		}
	}
	_, err = pipe.Exec(ctx)
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
	// 如果没有指定community id，则从所有的帖子数据中按照指定顺序排序
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

// GetPostVoteNumById 获取特定帖子的点赞数
func GetPostVoteNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostVoteZset), strconv.FormatInt(postId, 10)).Result()
	return
}

// GetPostDeVoteNumById 获取特定帖子的点踩数
func GetPostDeVoteNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostDevoteZset), strconv.FormatInt(postId, 10)).Result()
	return
}

// GetPostCommentNumById 获取特定帖子的评论数
func GetPostCommentNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostCommentZset), strconv.FormatInt(postId, 10)).Result()
	return
}

// GetPostClickNumById 获取特定帖子的浏览数
func GetPostClickNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostClickZset), strconv.FormatInt(postId, 10)).Result()
	return
}

// AddPostClickNum 设置帖子浏览量+1
func AddPostClickNum(postId int64) (err error) {
	err = rdb.ZIncrBy(ctx, getKey(KeyPostClickZset), 1, strconv.FormatInt(postId, 10)).Err()
	return
}
