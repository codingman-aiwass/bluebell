package logic

import (
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"errors"
	"go.uber.org/zap"
	"strconv"
)

var (
	ERROR_ILLEGAL_COMMENT_DELETE = errors.New("can not delete other's comment")
)

func CreateComment(comment *models.Comment) (err error) {
	err = mysql_repo.CommentRepository.Create(sqls.DB(), comment)
	if err != nil {
		zap.L().Error("mysql_repo.CreateComment(comment) failed", zap.Error(err))
		return
	}
	var rootCommentId int64
	if comment.ParentCommentId != 0 {
		rootCommentId, err = mysql_repo.CommentRepository.GetRootCommentId(sqls.DB(), comment.ParentCommentId)
		if err != nil {
			zap.L().Error("mysql_repo.CommentRepository.GetRootCommentId failed", zap.Error(err))
		}
	}
	// 获取根评论的id

	err = redis_repo.CreateComment(comment, rootCommentId)
	if err != nil {
		zap.L().Error("create comment in redis_repo failed", zap.Error(err))
		return err
	}
	return nil
}

// 有几种情况
// 1. direction为1，原值为0，-1.最终的值会在原值的基础上加1或者2
// 2. direction为0，原值为1，-1。最终的值会在原值的基础上加1或者-1
// 3. direction为-1，原值为1，0。最终的值会在原值的基础上减1或者2

func VoteComment(userId int64, comment *models.ParamVoteComment) (err error) {
	// 从Redis中获取当前用户对评论的点赞情况
	oValue, err := redis_repo.GetUser2CommentVoted(strconv.FormatInt(userId, 10), strconv.FormatInt(comment.CommentId, 10))
	if err != nil {
		zap.L().Error("get post voted error", zap.Int64("userid", userId), zap.String("postId", strconv.FormatInt(comment.CommentId, 10)), zap.Error(err))
		return
	}
	// 如果原值和新值相同，不做处理
	if oValue == float64(comment.Direction) {
		return nil
	}
	err = redis_repo.SetUser2CommentVotedAndPostVoteNum(float64(comment.Direction), oValue, strconv.FormatInt(comment.CommentId, 10), strconv.FormatInt(userId, 10))

	if err != nil {
		zap.L().Error("error occur during modify redis post vote or devote...", zap.Error(err))
		return err
	}
	return nil
}

func DeleteComment(commentId, userId int64) (err error) {
	// 先确认这个userID是否为该post的作者
	comment := mysql_repo.CommentRepository.Get(sqls.DB(), commentId)
	if comment.UserId != userId {
		zap.L().Warn("only author can delete his own post")
		return ERROR_ILLEGAL_COMMENT_DELETE
	}

	// 删除对该comment所有的点赞/点踩/收藏/评论/分数
	// 删除对该comment的所有追评，如果该评论是根评论
	// 点赞/点踩/分数/评论数在redis中
	// 子评论在MySQL中
	var rootCommentId int64 = 0
	if comment.ParentCommentId != 0 {
		// 说明不是根节点
		rootCommentId, err = mysql_repo.CommentRepository.GetRootCommentId(sqls.DB(), comment.ParentCommentId)
		if err != nil {
			return err
		}
	} else {
		rootCommentId = commentId
	}

	// 从redis中获取需要删除的所有评论ID
	commentIdList, err := redis_repo.GetToDeleteComment(commentId, rootCommentId)
	if err != nil {
		zap.L().Error("get all to delete comment ids error in logic.DeleteComment()", zap.Error(err))
		return err
	}

	// 通过获取的ID列表删除所有评论
	if err = mysql_repo.CommentRepository.DeleteCommentInfo(sqls.DB(), commentIdList); err != nil {
		zap.L().Error("delete comments in mysql error in logic.DeleteComment()", zap.Error(err))
		return err
	}

	// 删除redis中和评论相关的所有记录
	err = redis_repo.DeleteCommentInfo(commentId, rootCommentId, comment.PostId, commentIdList)
	if err != nil {
		zap.L().Error("fail to delete post related info in redis", zap.Error(err))
		return err
	}

	return nil
}
