package logic

import (
	"bluebell/cache"
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"errors"
	"go.uber.org/zap"
	"math"
	"strconv"
)

var (
	ERROR_POST_NOT_EXISTS = errors.New("post not exists")
)

func CreatePost(post *models.Post) (err error) {
	err = mysql_repo.PostRepository.Create(sqls.DB(), post)
	if err != nil {
		zap.L().Error("mysql_repo.CreatePost(post) failed", zap.Error(err))
		return
	}
	err = redis_repo.CreatePost(post)
	if err != nil {
		zap.L().Error("create post in redis_repo failed", zap.Error(err))
		return err
	}
	return nil
}

func GetPostById(id int64) (post *models.Post, err error) {
	p := cache.PostCache.Get(id)
	if p == nil {
		return nil, ERROR_POST_NOT_EXISTS
	}
	return p, nil
}

func GetPosts(page int, size int) (posts []models.Post, err error) {
	posts = mysql_repo.PostRepository.Find(sqls.DB(), sqls.NewCnd().Page(page, size))
	if len(posts) == 0 {
		err = ERROR_POST_NOT_EXISTS
	}
	return posts, err
}

// 有几种情况
// 1. direction为1，原值为0，-1.最终的值会在原值的基础上加1或者2
// 2. direction为0，原值为1，-1。最终的值会在原值的基础上加1或者-1
// 3. direction为-1，原值为1，0。最终的值会在原值的基础上减1或者2

func VotePost(userId int64, post *models.ParamVotePost) (err error) {
	// 首先需要检查离帖子发布时间是否超过一周
	expired, err := redis_repo.CheckPostExpired(post.PostId)
	if err != nil {
		zap.L().Error("redis_repo check post expired failed", zap.Error(err))
		return
	}
	if expired {
		return redis_repo.ERROR_EXPIRED_POST
	}

	// 需要从Redis中获取当前用户对该帖子的评分情况
	oValue, err := redis_repo.GetUser2PostScore(strconv.Itoa(int(userId)), post.PostId)
	if err != nil {
		zap.L().Error("get post score error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
		return
	}

	diff := math.Abs(float64(post.Direction) - oValue)
	// 修改评分
	if post.Direction != 0 {
		err = redis_repo.SetPostScore(post.PostId, redis_repo.PER_VOTE_VALUE*diff*float64(post.Direction))
	} else {
		err = redis_repo.SetPostScore(post.PostId, -redis_repo.PER_VOTE_VALUE*diff*oValue)
	}
	if err != nil {
		zap.L().Error("set post score error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
		return
	}
	// 修改user目前评分
	err = redis_repo.SetUser2PostScore(strconv.Itoa(int(userId)), post.PostId, float64(post.Direction))
	if err != nil {
		zap.L().Error("update user score to redis_repo error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
		return err
	}

	return nil
}

func GetPostsIds(param *models.ParamPostList) (ids []string, err error) {
	ids, err = redis_repo.GetPostIds(param)
	return ids, err
}

func GetPostVotes(ids []string, vote string) (result []int64, err error) {
	// vote =  1  即为查询赞成票
	// vote = -1  即为查询反对票
	// zcount bluebell:post:voted:613378252513218560 1 1
	result, err = redis_repo.GetPostVote(ids, vote)
	return result, err
}

func GetPostsWithOrder(param *models.ParamPostList) (posts []models.Post, err error) {
	// 首先从redis中获取id列表
	ids, err := redis_repo.GetPostIds(param)
	if err != nil {
		zap.L().Error("redis_repo get post ids failed", zap.Error(err))
		return nil, err
	}
	// 根据id列表从mysql中获取post信息
	posts = mysql_repo.PostRepository.Find(sqls.DB(), sqls.NewCnd().In("post_id", ids).Desc("post_id"))
	if len(posts) == 0 {
		err = ERROR_POST_NOT_EXISTS
	}
	//posts, err = mysql_repo.GetPostsByIds(ids)
	return posts, err
}
