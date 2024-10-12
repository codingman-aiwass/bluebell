package logic

import (
	"bluebell/cache"
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/message_queue"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"errors"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"strconv"
)

const PostType = 1

var Directions = [3]string{"none", "like", "dislike"}
var (
	ERROR_POST_NOT_EXISTS     = errors.New("post not exists")
	ERROR_ILLEGAL_POST_DELETE = errors.New("can not delete other's post")
)

func CreatePost(post *models.Post) (err error) {
	err = mysql_repo.PostRepository.Create(sqls.DB(), post)
	if err != nil {
		zap.L().Error("mysql_repo.CreatePost(post) failed", zap.Error(err))
		return err
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
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ERROR_POST_NOT_EXISTS
	}
	return p, nil
}

// GetPostDetailedInfo1 返回帖子的其他详细信息，例如点赞数，评论数，浏览量
func GetPostDetailedInfo1(postId int64) (yes_vote, comment_num, click_num int64) {
	yes_vote = GetPostVoteNumById(postId)
	comment_num = GetPostCommentNumById(postId)
	click_num = GetPostClickNumById(postId)
	return
}

func GetPostVoteNumById(postId int64) int64 {
	result, err := redis_repo.GetPostVoteNumById(postId)
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		cnt := mysql_repo.VoteRepository.Count(sqls.DB(), sqls.NewCnd().Where("type = ?", 1).Where("target_id = ?", postId).Where("val = ?", 1))
		err = nil

		redis_repo.AddToZset(ctx, redis_repo.KeyPostVoteUpZset, redis.Z{Score: float64(cnt), Member: postId})
	}
	return int64(result)
}

func GetPostCommentNumById(postId int64) int64 {
	result, err := redis_repo.GetPostCommentNumById(postId)
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		cnt := mysql_repo.CommentRepository.Count(sqls.DB(), sqls.NewCnd().Where("post_id = ?", postId))
		err = nil

		redis_repo.AddToZset(ctx, redis_repo.KeyPostCommentZset, redis.Z{Score: float64(cnt), Member: postId})
	}
	return int64(result)
}

func GetPostClickNumById(postId int64) int64 {
	result, err := redis_repo.GetPostClickNumById(postId)
	if errors.Is(err, redis.Nil) {
		// 说明需要去mysql中查询，缓存中无此项数据
		post := mysql_repo.PostRepository.Get(sqls.DB(), postId)
		result = float64(post.ClickNums)
		err = nil

		redis_repo.AddToZset(ctx, redis_repo.KeyPostClickZset, redis.Z{Score: float64(post.ClickNums), Member: post.PostId})
	}
	return int64(result)
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

	// 需要从Redis中获取当前用户对该帖子的点赞情况
	oValue, err := redis_repo.GetUser2PostVoted(strconv.FormatInt(userId, 10), post.PostId)
	if errors.Is(err, redis.Nil) {
		// 说明Redis中不存在该用户对此帖子的点赞情况，可能是Redis中数据丢失了（需要去MySQL中查找），也可能是用户确实没有点赞/点踩过该帖子（mysql数据库中也没有记录）
		// 检查是否存在布隆过滤器
		exists, err := redis_repo.Exists(ctx, redis_repo.UserLikeOrDislike2PostBloomFilter)
		if err != nil {
			zap.L().Error("checking UserLikeOrDislike2PostBloomFilter error", zap.Error(err))
			return err
		}
		if exists {
			// 当查询Redis时发现查不到该用户的操作记录以后，就去布隆过滤器查。布隆过滤器也没有此数据的话，说明MySQL中不可能会有，不需要去查，布隆过滤器中有，才去MySQL查
			// 去布隆过滤器中查询该用户是否点赞/点踩过此帖子
			exist, err := redis_repo.CheckInBloomFilter(redis_repo.UserLikeOrDislike2PostBloomFilter, strconv.FormatInt(userId, 10))
			if err != nil {
				zap.L().Error("Error occurred in CheckInBloomFilter", zap.Error(err))
				return err
			}
			if exist {
				// 说明布隆过滤器中存在该用户记录，数据库中有记录，去数据库查询
				vote := mysql_repo.VoteRepository.FindOne(sqls.DB(), sqls.NewCnd().Where("user_id = ?", userId).Where("type = ?", PostType).Where("target_id = ?", post.PostId))
				if vote != nil {
					oValue = Directions[vote.Val]
				} else {
					// 数据库中不存在用户点赞记录
					oValue = Directions[0]
				}

			} else {
				// 布隆过滤器中不存在该用户记录，数据库中也没有记录，直接设置为none
				oValue = Directions[0]
			}
		} else {
			// 布隆过滤器不存在，需要创建，同时也说明当前没有用户对帖子点赞/点踩过（项目刚上线）,此时数据库中也不会有记录，直接设置为none
			err = redis_repo.CreateBloomFilter(ctx, redis_repo.UserLikeOrDislike2PostBloomFilter, 0.01, 10000)
			if err != nil {
				zap.L().Error("Error occurred in CreateBloomFilter", zap.Error(err))
				return err
			}
			oValue = Directions[0]
		}
	} else if err != nil {
		zap.L().Error("get post voted error", zap.Int64("userid", userId), zap.String("postId", post.PostId), zap.Error(err))
		return
	}
	// 如果原值和新值相同，不做处理
	if oValue == Directions[*post.Direction] {
		return nil
	}
	// 去redis中写入数据，并在写入Redis之前发送修改消息到消息队列，消费者需要处理消息（修改MySQL，以及向用户发送私信）
	postId, _ := strconv.ParseInt(post.PostId, 10, 64)
	err = redis_repo.SetUser2PostVotedAndPostVoteNum(Directions[*post.Direction], oValue, postId, userId)

	if err != nil {
		zap.L().Error("error occur during modify redis post vote or devote...", zap.Error(err))
		return err
	}

	return nil
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

	// 先删除MySQL数据，然后删除缓存
	if err = mysql_repo.PostRepository.DeletePostInfo(sqls.DB(), postId); err != nil {
		zap.L().Error("fail to delete post related info in mysql", zap.Error(err))
		return err
	}
	// 清除post缓存
	cache.PostCache.Invalidate(postId)

	// 删除对该帖子所有的点赞/点踩/收藏/评论/分数
	// 点赞/点踩/分数/评论数在redis中
	// 收藏/评论在MySQL中
	err = redis_repo.DeletePostInfo(postId, post.CommunityID)
	if err != nil {
		zap.L().Error("fail to delete post related info in redis", zap.Error(err))
		return err
	}
	return nil
}

// GetBlackList 获取帖子作者的黑名单
func GetBlackList(postId int64) (blackList []string, err error) {
	// 首先获取帖子的作者
	post, err := GetPostById(postId)
	if err != nil {
		zap.L().Error("fail to find post via postId", zap.Error(err))
		return nil, err
	}

	// 根据帖子作者ID获取它的黑名单
	blackList, err = redis_repo.GetBlackListById(ctx, post.AuthorID)
	if err != nil {
		zap.L().Error("fail to get blacklist via author id", zap.Error(err))
		return nil, err
	}

	return blackList, nil
}

// CheckInBlacklist 判断userId1是否在post作者的黑名单上
func CheckInBlacklist(userId, postId int64) (res bool, err error) {
	// 首先获取帖子的作者
	post, err := GetPostById(postId)
	if err != nil {
		zap.L().Error("fail to find post via postId", zap.Error(err))
		return
	}
	res, err = redis_repo.CheckInBlackList(ctx, strconv.FormatInt(userId, 10), strconv.FormatInt(post.AuthorID, 10))
	if err != nil {
		zap.L().Error("fail to run redis_repo.CheckInBlackList", zap.Error(err))
		return
	}
	return res, nil
}

func AddPostClickNum(userId, postId int64) (err error) {
	// 先写入redis，然后再发送到消息队列
	// 如果redis中没有值，读取数据库中的值并加1写入redis
	err = redis_repo.AddPostClickNum(postId)
	if err != nil {
		zap.L().Error("error in logic.AddPostClickNum()", zap.Error(err))
		return err
	}
	// 往消息队列中发送一个请求
	event := message_queue.PostClickEvent{UserId: userId, PostId: postId}
	err = message_queue.SendPostClickEvent(ctx, event)
	if err != nil {
		return err
	}
	return nil
}
