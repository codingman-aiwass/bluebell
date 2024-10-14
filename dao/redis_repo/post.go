package redis_repo

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
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
func GetUser2PostVoted(userId, postId string) (score string, err error) {
	score, err = rdb.HGet(ctx, getKey(KeyPostActionPrefix+postId), userId).Result()
	return
}

// SetUser2PostVotedAndPostVoteNum 在事务中修改用户对帖子的点赞/点踩情况以及帖子的点赞/点踩统计数据
func SetUser2PostVotedAndPostVoteNum(curDirection, oDirection string, postId, userId int64) (err error) {
	if curDirection == "like" {
		err = likePost(postId, userId, oDirection)
		if err != nil {
			zap.L().Error("error in likePost() in redis_repo.SetUser2PostVotedAndPostVoteNum()", zap.Error(err))
			return err
		}
	} else if curDirection == "none" {
		err = cancelLikeOrDislike(postId, userId, oDirection)
		if err != nil {
			zap.L().Error("error in cancelLikeOrDislike() in redis_repo.SetUser2PostVotedAndPostVoteNum()", zap.Error(err))
			return err
		}
	} else if curDirection == "dislike" {
		err = dislikePost(postId, userId, oDirection)
		if err != nil {
			zap.L().Error("error in dislikePost() in redis_repo.SetUser2PostVotedAndPostVoteNum()", zap.Error(err))
			return err
		}
	}
	return
}

func likePost(postId, userId int64, oValue string) (err error) {
	pipe := rdb.TxPipeline()
	// 取消点踩
	if oValue == "dislike" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteDownZset), -1, strconv.FormatInt(postId, 10))
	}

	// 设置为点赞
	if oValue != "like" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteUpZset), 1, strconv.FormatInt(postId, 10))
		pipe.HSet(ctx, getKey(KeyPostActionPrefix+strconv.FormatInt(postId, 10)), userId, "like")
		pipe.Do(ctx, "BF.ADD", UserLikeOrDislike2PostBloomFilter, userId)

	}
	_, err = pipe.Exec(ctx)
	return
}

func dislikePost(postId, userId int64, oValue string) (err error) {
	pipe := rdb.TxPipeline()
	// 取消点赞
	if oValue == "like" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteUpZset), -1, strconv.FormatInt(postId, 10))
	}

	// 设置为点踩
	if oValue != "dislike" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteDownZset), 1, strconv.FormatInt(postId, 10))
		pipe.HSet(ctx, getKey(KeyPostActionPrefix+strconv.FormatInt(postId, 10)), userId, "dislike")
		pipe.Do(ctx, "BF.ADD", UserLikeOrDislike2PostBloomFilter, userId)
	}
	_, err = pipe.Exec(ctx)
	return
}

func cancelLikeOrDislike(postId, userId int64, oValue string) (err error) {
	pipe := rdb.TxPipeline()
	pipe.HSet(ctx, getKey(KeyPostActionPrefix+strconv.FormatInt(postId, 10)), userId, "none")
	if oValue == "like" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteUpZset), -1, strconv.FormatInt(postId, 10))
	} else if oValue == "dislike" {
		pipe.ZIncrBy(ctx, getKey(KeyPostVoteDownZset), -1, strconv.FormatInt(postId, 10))
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
	err = rdb.ZIncrBy(ctx, getKey(KeyPostVoteUpZset), vote, postId).Err()
	return
}

func SetPostDevote(postId string, vote float64) (err error) {
	err = rdb.ZIncrBy(ctx, getKey(KeyPostVoteDownZset), vote, postId).Err()
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
	// 需要检查post:time或者post_score是否存在，不存在的话需要从MySQL中读取
	if param.Order == models.OrderByTime {
		key = getKey(KeyPostTimeZset)
		exists, err := Exists(ctx, getKey(KeyPostTimeZset))
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error checking key: %s", getKey(KeyPostTimeZset)))
			return nil, err
		}
		if !exists {
			// 从数据库中提取数据构造缓存
			posts := mysql_repo.PostRepository.Find(sqls.DB(), sqls.NewCnd())
			pipe := rdb.TxPipeline()
			for _, post := range posts {
				pipe.ZAdd(ctx, getKey(KeyPostTimeZset), redis.Z{Score: float64(post.CreateAt.Unix()), Member: post.PostId})
			}
			_, err = pipe.Exec(ctx)
			if err != nil {
				zap.L().Error("Error adding post id to post:create_time set")
				return nil, err
			}
		}
	} else if param.Order == models.OrderByScore {
		key = getKey(KeyPostScoreZset)
		exists, err := Exists(ctx, getKey(KeyPostScoreZset))
		if err != nil {
			zap.L().Error(fmt.Sprintf("Error checking key: %s", getKey(KeyPostScoreZset)))
			return nil, err
		}
		if !exists {
			// 从数据库中提取数据构造缓存
			posts := mysql_repo.PostRepository.Find(sqls.DB(), sqls.NewCnd())
			pipe := rdb.TxPipeline()
			for _, post := range posts {
				pipe.ZAdd(ctx, getKey(KeyPostScoreZset), redis.Z{Score: float64(post.Score), Member: post.PostId})
			}
			_, err = pipe.Exec(ctx)
			if err != nil {
				zap.L().Error("Error adding post id to post:score set")
				return nil, err
			}
		}
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

			// 检查bluebell:community:[community-id]是否存在，防止Redis因为意外失去这部分数据
			exists, err := Exists(ctx, getKey(KeyCommunityPrefix+param.CommunityId))
			if err != nil {
				zap.L().Error(fmt.Sprintf("Error checking key: %s", getKey(KeyCommunityPrefix+param.CommunityId)))
				return nil, err
			}
			if !exists {
				// 从数据库中提取数据构造缓存
				posts := mysql_repo.PostRepository.Find(sqls.DB(), sqls.NewCnd().Where("community_id = ?", param.CommunityId))
				pipe := rdb.TxPipeline()
				for _, post := range posts {
					pipe.SAdd(ctx, getKey(KeyCommunityPrefix+param.CommunityId), post.PostId)
				}
				_, err = pipe.Exec(ctx)
				if err != nil {
					zap.L().Error("Error adding post id to community:[community] set")
					return nil, err
				}
			}
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

func DeletePostInfo(postId, communityId int64) (err error) {
	pipe := rdb.TxPipeline()
	pipe.ZRem(ctx, getKey(KeyPostVoteUpZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostVoteDownZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostScoreZset), postId)
	pipe.ZRem(ctx, getKey(KeyPostCommentZset), postId)
	pipe.ZRem(ctx, fmt.Sprintf("%s:%d", getKey(KeyPostScoreZset), communityId), postId)
	_, err = pipe.Exec(ctx)
	return

}

// GetPostVoteNumById 获取特定帖子的点赞数
func GetPostVoteNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostVoteUpZset), strconv.FormatInt(postId, 10)).Result()
	return
}

// GetPostDeVoteNumById 获取特定帖子的点踩数
func GetPostDeVoteNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostVoteDownZset), strconv.FormatInt(postId, 10)).Result()
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
		result = float64(post.VoteDownNums)
		err = nil
		rdb.ZAdd(ctx, getKey(KeyPostVoteDownZset), redis.Z{Score: float64(post.VoteDownNums), Member: post.PostId})
	}
	return
}

// GetPostCommentNumById 获取特定帖子的评论数
func GetPostCommentNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostCommentZset), strconv.FormatInt(postId, 10)).Result()
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
		result = float64(post.CommentNums)
		err = nil
		rdb.ZAdd(ctx, getKey(KeyPostCommentZset), redis.Z{Score: float64(post.CommentNums), Member: post.PostId})
	}
	return
}

// GetPostClickNumById 获取特定帖子的浏览数
func GetPostClickNumById(postId int64) (result float64, err error) {
	result, err = rdb.ZScore(ctx, getKey(KeyPostClickZset), strconv.FormatInt(postId, 10)).Result()
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
		result = float64(post.ClickNums)
		err = nil
		rdb.ZAdd(ctx, getKey(KeyPostClickZset), redis.Z{Score: float64(post.ClickNums), Member: post.PostId})
	}
	return
}

// AddPostClickNum 设置帖子浏览量+1
func AddPostClickNum(postId int64) (err error) {
	_, err = rdb.ZScore(ctx, getKey(KeyPostClickZset), strconv.FormatInt(postId, 10)).Result()
	if errors.Is(err, redis.Nil) {
		// 去MySQL查询浏览量
		post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
		err = rdb.ZIncrBy(ctx, getKey(KeyPostClickZset), float64(post.ClickNums+1), strconv.FormatInt(postId, 10)).Err()
		return nil
	} else if err != nil {
		zap.L().Error("fail to check post click num in redis", zap.Error(err))
		return err
	}
	err = rdb.ZIncrBy(ctx, getKey(KeyPostClickZset), 1, strconv.FormatInt(postId, 10)).Err()
	return
}

func AddCollection(postId int64, userId int64) (exist bool, err error) {
	key := getKey(KeyPostUserCollection + strconv.FormatInt(userId, 10))
	flag, err := rdb.SIsMember(ctx, key, postId).Result()
	if err != nil {
		zap.L().Error("check if user collect post error", zap.Error(err))
		return false, err
	}

	if flag {
		// 说明用户已经收藏了该帖子，不必重复收藏
		return true, nil
	}
	pipe := rdb.TxPipeline()
	pipe.SAdd(ctx, key, postId)
	pipe.ZIncrBy(ctx, getKey(KeyPostCollectionZset), 1, strconv.FormatInt(postId, 10))
	_, err = pipe.Exec(ctx)
	return false, err

}
