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

func CreateComment(comment *models.Comment, rootCommentId int64) (err error) {
	pipe := rdb.TxPipeline()
	// 创建评论的time和score记录，以及该评论归属于哪一post
	pipe.ZAdd(ctx, getKey(KeyCommentTimeZset), redis.Z{Score: float64(time.Now().Unix()), Member: comment.CommentId})
	pipe.ZAdd(ctx, getKey(KeyCommentScoreZset), redis.Z{Score: 0, Member: comment.CommentId})
	// 添加帖子的评论数量 post:comment_numbers,zset， key 为帖子id
	pipe.ZIncrBy(ctx, getKey(KeyPostCommentZset), 1, strconv.FormatInt(comment.PostId, 10))
	if comment.ParentCommentId == 0 {
		// 只有post的根评论才加到Redis数据库 bluebell:post:[commentId] set
		pipe.SAdd(ctx, getKey(KeyPostPrefix+strconv.FormatInt(comment.PostId, 10)), comment.CommentId)

	} else {
		// 评论的评论，需要记录到Redis中，方便从根评论找到所有子评论,记录到bluebell:comment:child_comment_record:[parent_comment_id],set
		pipe.SAdd(ctx, getKey(KeyCommentSubCommentSet+":"+strconv.FormatInt(comment.ParentCommentId, 10)), comment.CommentId)
		// 记录根评论下的评论总数,bluebell:comment:comment_numbers,zset key 为rootCommentId
		pipe.ZIncrBy(ctx, getKey(KeyCommentSubCommentCntZset), 1, strconv.FormatInt(rootCommentId, 10))
	}

	_, err = pipe.Exec(ctx)
	return
}

func GetUser2CommentVoted(userId, commentId string) (score float64, err error) {
	score, err = rdb.ZScore(ctx, getKey(KeyCommentVotedZset+":"+commentId), userId).Result()
	if errors.Is(err, redis.Nil) {
		score = 0
		err = nil
	}
	return
}

// SetUser2CommentVotedAndPostVoteNum 在事务中修改用户对comment的点赞/点踩情况以及帖子的点赞/点踩统计数据
func SetUser2CommentVotedAndPostVoteNum(curDirection, oDirection float64, postId, userId string) (err error) {
	pipe := rdb.TxPipeline()
	if curDirection == 1 {
		// 需要将bluebell:comment:voted:commentId 下该用户的记录设置为1
		pipe.ZRem(ctx, getKey(KeyCommentVotedZset+":"+postId), userId)
		pipe.ZAdd(ctx, getKey(KeyCommentVotedZset+":"+postId), redis.Z{Score: curDirection, Member: userId})
		pipe.ZIncrBy(ctx, getKey(KeyCommentVoteZset), 1, postId)
		if oDirection == -1 {
			// 取消点踩
			pipe.ZIncrBy(ctx, getKey(KeyCommentDevoteZset), -1, postId)
		}
	} else if curDirection == 0 {
		// 需要删除bluebell:comment:voted:commentId 下该用户的记录
		pipe.ZRem(ctx, getKey(KeyCommentVotedZset+":"+postId), userId)
		if oDirection == 1 {
			// 取消点赞
			pipe.ZIncrBy(ctx, getKey(KeyCommentVoteZset), -1, postId)
		} else if oDirection == -1 {
			// 取消点踩
			pipe.ZIncrBy(ctx, getKey(KeyCommentDevoteZset), -1, postId)
		}
	} else if curDirection == -1 {
		// 需要将bluebell:comment:voted:commentId 下该用户的记录设置为-1
		pipe.ZRem(ctx, getKey(KeyCommentVotedZset+":"+postId), userId)
		pipe.ZAdd(ctx, getKey(KeyCommentVotedZset+":"+postId), redis.Z{Score: curDirection, Member: userId})
		pipe.ZIncrBy(ctx, getKey(KeyCommentDevoteZset), 1, postId)
		if oDirection == 1 {
			pipe.ZIncrBy(ctx, getKey(KeyCommentVoteZset), -1, postId)
		}
	}
	_, err = pipe.Exec(ctx)
	return
}

// GetToDeleteComment 从Redis中获取所有需要删除的commentId，从KeyCommentSubCommentSet:commentId逐个记录
func GetToDeleteComment(commentId, rootCommentId int64) (res []string, err error) {
	var que []string
	que = append(que, strconv.FormatInt(commentId, 10))
	for len(que) > 0 {
		cur := que[0]
		que = que[1:]
		res = append(res, cur)
		tmp, err := rdb.SMembers(ctx, getKey(KeyCommentSubCommentSet+":"+cur)).Result()
		if err != nil {
			zap.L().Error("find child comments error in redis_repo.GetToDeleteComment()", zap.Error(err))
			return nil, err
		}
		que = append(que, tmp...)
	}
	return res, nil
}

func DeleteCommentInfo(commentId, rootCommentId, postId int64, relatedCommentIds []string) (err error) {
	pipe := rdb.TxPipeline()

	for _, id := range relatedCommentIds {
		_, err = rdb.ZScore(ctx, getKey(KeyCommentVoteZset), id).Result()
		if errors.Is(err, redis.Nil) {
			zap.L().Info(fmt.Sprintf("%s:%d not exists in redis.", getKey(KeyCommentVoteZset), id))
		} else if err == nil {
			pipe.ZRem(ctx, getKey(KeyCommentVoteZset), id)
		} else {
			zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyCommentVoteZset),id).Result() error!", zap.Error(err))
			return err
		}

		_, err = rdb.ZScore(ctx, getKey(KeyCommentDevoteZset), id).Result()
		if errors.Is(err, redis.Nil) {
			zap.L().Info(fmt.Sprintf("%s:%d not exists in redis.", getKey(KeyCommentDevoteZset), id))
		} else if err == nil {
			pipe.ZRem(ctx, getKey(KeyCommentDevoteZset), id)
		} else {
			zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyCommentDevoteZset),id).Result() error!", zap.Error(err))
			return err
		}

		flag, err := rdb.Exists(ctx, getKey(KeyCommentVotedZset+":"+id)).Result()
		if err != nil {
			zap.L().Error("execute rdb.Exists(ctx,getKey(KeyCommentSubCommentCntZset),commentId).Result() error!", zap.Error(err))
			return err
		}
		if flag > 0 {
			pipe.Del(ctx, getKey(KeyCommentVotedZset+":"+id))
		}

		key := getKey(KeyCommentSubCommentSet + ":" + id)
		flag, err = rdb.Exists(ctx, key).Result()
		if err != nil {
			zap.L().Error("execute rdb.Exists(ctx,getKey(KeyCommentSubCommentCntZset),commentId).Result() error!", zap.Error(err))
			return err
		}
		if flag > 0 {
			pipe.Del(ctx, key)
		}

		// score需要在for中删除
		_, err = rdb.ZScore(ctx, getKey(KeyCommentScoreZset), id).Result()
		if errors.Is(err, redis.Nil) {
			zap.L().Info(fmt.Sprintf("%s:%s not exists in redis.", getKey(KeyCommentScoreZset), id))

		} else if err == nil {
			pipe.ZRem(ctx, getKey(KeyCommentScoreZset), id)
		} else {
			zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyCommentScoreZset),id).Result() error!", zap.Error(err))
			return err
		}
		// time需要在for中删除
		_, err = rdb.ZScore(ctx, getKey(KeyCommentTimeZset), id).Result()
		if errors.Is(err, redis.Nil) {
			zap.L().Info(fmt.Sprintf("%s:%s not exists in redis.", getKey(KeyCommentTimeZset), id))
		} else if err == nil {
			pipe.ZRem(ctx, getKey(KeyCommentTimeZset), id)
		} else {
			zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyPostCommentZset),commentId).Result() error!", zap.Error(err))
			return err
		}

		// 对帖子下的评论总数执行-1操作
		_, err = rdb.ZScore(ctx, getKey(KeyPostCommentZset), strconv.FormatInt(postId, 10)).Result()
		if errors.Is(err, redis.Nil) {
			zap.L().Info(fmt.Sprintf("%s:%d not exists in redis.", getKey(KeyPostCommentZset), postId))

		} else if err == nil {
			pipe.ZIncrBy(ctx, getKey(KeyPostCommentZset), -1, strconv.FormatInt(postId, 10))
		} else {
			zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyPostCommentZset),commentId).Result() error!", zap.Error(err))
			return err
		}

		// 对根评论下的追评总数执行-1操作
		if id != strconv.FormatInt(rootCommentId, 10) {
			_, err = rdb.ZScore(ctx, getKey(KeyCommentSubCommentCntZset), strconv.FormatInt(rootCommentId, 10)).Result()
			if errors.Is(err, redis.Nil) {
				zap.L().Info(fmt.Sprintf("%s:%d not exists in redis.", getKey(KeyCommentSubCommentCntZset), rootCommentId))
			} else if err == nil {
				pipe.ZIncrBy(ctx, getKey(KeyCommentSubCommentCntZset), -1, strconv.FormatInt(rootCommentId, 10))
			} else {
				zap.L().Error("execute rdb.ZScore(ctx,getKey(KeyCommentSubCommentCntZset),commentId).Result() error!", zap.Error(err))
				return err
			}
		}

	}
	// 如果删除根评论，还需要删除bluebell:post:[postid] 下的根comment id，以及bluebell:comment:comment_numbers 下的根comment id
	if commentId == rootCommentId {
		pipe.SRem(ctx, getKey(KeyPostPrefix+strconv.FormatInt(postId, 10)), commentId)
		pipe.ZRem(ctx, getKey(KeyCommentSubCommentCntZset), commentId)
	}

	//pipe.ZRem(ctx, getKey(KeyCommentVoteZset), commentId)
	//pipe.ZRem(ctx, getKey(KeyCommentDevoteZset), commentId)
	//pipe.ZRem(ctx, getKey(KeyCommentScoreZset), commentId)
	//pipe.ZRem(ctx, getKey(KeyCommentTimeZset), commentId)
	//pipe.ZRem(ctx, getKey(KeyPostCommentZset), rootCommentId)
	//pipe.Del(ctx,getKey(KeyCommentVotedZset + ":" + commentId))
	// 删除子评论计数缓存
	//pipe.ZRem(ctx, getKey(KeyCommentSubCommentCntZset), rootCommentId)
	// 删除所有相关的bluebell:child_comment_record:[commentId]
	//for _, id := range relatedCommentIds {
	//	key := getKey(KeyCommentSubCommentSet + ":" + id)
	//	pipe.SRem(ctx, key)
	//}
	_, err = pipe.Exec(ctx)

	return
}
