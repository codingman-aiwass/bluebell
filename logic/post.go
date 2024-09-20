package logic

import (
	"bluebell/cache"
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"errors"
	"go.uber.org/zap"
	"strconv"
)

var (
	ERROR_POST_NOT_EXISTS     = errors.New("post not exists")
	ERROR_ILLEGAL_POST_DELETE = errors.New("can not delete other's post")
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

// 返回帖子的其他详细信息，例如点赞数，评论数，浏览量
func GetPostDetailedInfo1(postId int64) (yes_vote, comment_num, click_num int64) {
	yes, err := redis_repo.GetPostVoteNumById(postId)
	if err != nil {
		yes = 0
	}
	comment, err := redis_repo.GetPostCommentNumById(postId)
	if err != nil {
		comment = 0
	}

	click, err := redis_repo.GetPostClickNumById(postId)
	if err != nil {
		click = 0
	}
	yes_vote = int64(yes)
	comment_num = int64(comment)
	click_num = int64(click)
	return
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
	// 首先需要检查离帖子发布时间是否超过一周（取消该限制）
	//expired, err := redis_repo.CheckPostExpired(post.PostId)
	//if err != nil {
	//	zap.L().Error("redis_repo check post expired failed", zap.Error(err))
	//	return
	//}
	//if expired {
	//	return redis_repo.ERROR_EXPIRED_POST
	//}

	// 需要从Redis中获取当前用户对该帖子的评分情况
	oValue, err := redis_repo.GetUser2PostVoted(strconv.Itoa(int(userId)), post.PostId)
	if err != nil {
		zap.L().Error("get post score error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
		return
	}
	// 如果原值和新值相同，不做处理
	if oValue == float64(post.Direction) {
		return nil
	}

	// 修改点赞数量

	//if post.Direction == 1 {
	//	// 需要将bluebell:post:voted:postId 下该用户的记录设置为1
	//	err = redis_repo.SetPostVote(post.PostId, 1)
	//	if oValue == -1 {
	//		// 取消点踩
	//		err = redis_repo.SetPostDevote(post.PostId, -1)
	//	}
	//} else if post.Direction == 0 {
	//	// 需要删除bluebell:post:voted:postId 下该用户的记录
	//	if oValue == 1 {
	//		// 取消点赞
	//		err = redis_repo.SetPostVote(post.PostId, -1)
	//	} else if oValue == -1 {
	//		// 取消点踩
	//		err = redis_repo.SetPostDevote(post.PostId, -1)
	//	}
	//} else if post.Direction == -1 {
	//	// 需要将bluebell:post:voted:postId 下该用户的记录设置为-1
	//	err = redis_repo.SetPostDevote(post.PostId, 1)
	//	if oValue == 1 {
	//		err = redis_repo.SetPostVote(post.PostId, -1)
	//	}
	//}

	err = redis_repo.SetUser2PostVotedAndPostVoteNum(float64(post.Direction), oValue, post.PostId, strconv.FormatInt(userId, 10))

	if err != nil {
		zap.L().Error("error occur during modify redis post vote or devote...", zap.Error(err))
		return err
	}

	// 修改帖子点赞/点踩数量，和修改user目前点赞情况，都用SetUser2PostVotedAndPostVoteNum 在一个事务中处理了

	// 修改评分（之前的根据点赞/点踩情况计算帖子评分，暂时废弃）
	//diff := math.Abs(float64(post.Direction) - oValue)

	//if post.Direction != 0 {
	//	err = redis_repo.SetPostScore(post.PostId, redis_repo.PER_VOTE_VALUE*diff*float64(post.Direction))
	//} else {
	//	err = redis_repo.SetPostScore(post.PostId, -redis_repo.PER_VOTE_VALUE*diff*oValue)
	//}
	//if err != nil {
	//	zap.L().Error("set post score error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
	//	return
	//}

	// 修改user目前点赞/点踩情况
	//err = redis_repo.SetUser2PostVoted(strconv.Itoa(int(userId)), post.PostId, float64(post.Direction))
	//if err != nil {
	//	zap.L().Error("update user score to redis_repo error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
	//	return err
	//}

	return nil
}

func GetPostsIds(param *models.ParamPostList) (ids []string, err error) {
	ids, err = redis_repo.GetPostIds(param)
	return ids, err
}

// bluebell:post:voted:post_id 中存放了所有用户对这个帖子的点赞情况
// bluebell:post:vote 记录了每个帖子的点赞数
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

func AddCollectPost(postId, userId int64) (err error) {
	// 查看该用户是否已经收藏过该post，已经收藏过则取消收藏，没有收藏过则加入收藏
	// 然后修改redis中的收藏数
	like := mysql_repo.LikeRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("user_id = ?", userId).Where("post_id = ?", postId))
	if like == nil {
		// 说明没有收藏
		newLike := &models.Like{PostId: postId, UserId: userId, LikeId: snowflake.GenID()}
		err = mysql_repo.LikeRepository.Create(sqls.DB(), newLike)
		if err != nil {
			zap.L().Error("save new collect error", zap.Error(err))
			return err
		}
		err = redis_repo.AddPostCollectionNumber(postId, 1)
		if err != nil {
			zap.L().Error("add collection number in redis error", zap.Error(err))
			return err
		}
	} else {
		// 说明已经被收藏，删除收藏
		mysql_repo.LikeRepository.Delete(sqls.DB(), like.LikeId)
		err = redis_repo.AddPostCollectionNumber(postId, -1)

		if err != nil {
			zap.L().Error("add collection number in redis error", zap.Error(err))
			return err
		}
	}
	return err

}

func DeletePost(postId, userId int64) (err error) {
	// 先确认这个userID是否为该post的作者
	post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
	if post.AuthorID != userId {
		zap.L().Warn("only author can delete his own post")
		return ERROR_ILLEGAL_POST_DELETE
	}

	// 删除对该帖子所有的点赞/点踩/收藏/评论/分数
	// 点赞/点踩/分数/评论数在redis中
	// 收藏/评论在MySQL中
	err = redis_repo.DeletePostInfo(postId, post.CommunityID)
	if err != nil {
		zap.L().Error("fail to delete post related info in redis", zap.Error(err))
		return err
	}

	if err = mysql_repo.PostRepository.DeletePostInfo(sqls.DB(), postId); err != nil {
		zap.L().Error("fail to delete post related info in mysql", zap.Error(err))
		return err
	}
	return nil
}
