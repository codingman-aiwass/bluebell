package logic

import (
	"bluebell/cache"
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"errors"
	"go.uber.org/zap"
	"strconv"
)

const N_SUB_COMMENTS_TO_SHOW = 2
const CommentType = 2

var (
	ERROR_ILLEGAL_COMMENT_DELETE = errors.New("can not delete other's comment")
	ERROR_WRONG_COMMENT          = errors.New("no this comment")
)

func GetCommentById(commentId int64) (comment *models.Comment, err error) {
	c := cache.CommentCache.Get(commentId)
	if c == nil {
		return nil, ERROR_WRONG_COMMENT
	}
	return c, nil
}

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
	comment, err := GetCommentById(commentId)
	if err != nil {
		zap.L().Error("find comment id error in logic.DeleteComment()", zap.Error(err))
		return
	}
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
	// 手动清除被删除的评论缓存
	for _, id := range commentIdList {
		id_, _ := strconv.ParseInt(id, 10, 64)
		cache.CommentCache.Invalidate(id_)
	}

	// 删除redis中和评论相关的所有记录
	err = redis_repo.DeleteCommentInfo(commentId, rootCommentId, comment.PostId, commentIdList)
	if err != nil {
		zap.L().Error("fail to delete post related info in redis", zap.Error(err))
		return err
	}

	return nil
}

func GetCommentListByPostId(query *models.ParamGetCommentByPostId) (res [][]*models.Comment, err error) {
	// 从bluebell:post:[post-id]中找到所有的根评论
	// 在根据这些根评论，去bluebell:comment:child_comment_record:[comment-id]中找出所有的子评论
	// 采用BFS的策略
	RootComments, err := redis_repo.GetAllRootComment(query.PostId)
	// 存放不同根评论下的所有子评论，每个数组代表一个跟评论下所有的评论
	commentsIds := make([][]string, len(RootComments))
	for idx, root := range RootComments {
		ids, err := redis_repo.GetSubCommentIdsByRootComment(root)
		if err != nil {
			zap.L().Error("logic.GetCommentListByPostId error in get sub comment ids", zap.Error(err))
			return nil, err
		}
		commentsIds[idx] = ids
	}
	// 虽然获取了所有根评论下的所有追评，但是在展示的时候没有必要一次性全部查询出来
	// 通过page 和 size确定要查哪些记录，size确定每页显示几条根评论，page确定从第几页开始选
	start := (query.Page - 1) * query.Size
	end := start + query.Size
	// 查询[start,end]范围内的评论

	for i := start; i < end; i++ {
		if i >= len(commentsIds) {
			break
		}
		tmp := []*models.Comment{}
		// 查询根评论
		rootCommentId, _ := strconv.ParseInt(RootComments[i], 10, 64)
		rootComment, err1 := GetCommentById(rootCommentId)
		if err1 != nil {
			zap.L().Error("find root comment id error in logic.GetCommentListByPostId()", zap.Error(err1))
			return
		}
		//rootComment := mysql_repo.CommentRepository.Get(sqls.DB(), rootCommentId)
		tmp = append(tmp, rootComment)
		// 只展示前几条子评论
		for j := 0; j < min(N_SUB_COMMENTS_TO_SHOW, len(commentsIds[i])); j++ {
			commentId, _ := strconv.ParseInt(commentsIds[i][j], 10, 64)
			comment, err1 := GetCommentById(commentId)
			if err1 != nil {
				zap.L().Error("find comment id error in logic.GetCommentListByPostId()", zap.Error(err1))
				return
			}
			tmp = append(tmp, comment)
		}
		res = append(res, tmp)
	}
	return res, nil
}

func GetCommentDetail(commentId int64) (res models.ResponseComment, err error) {
	// 首先根据comment id获取comment 信息，判断这个是不是根评论
	comment := mysql_repo.CommentRepository.Get(sqls.DB(), commentId)
	var parentId int64
	if comment.ParentCommentId == 0 {
		// 说明是根评论，可直接加载所有的子评论
		parentId = comment.CommentId
	} else {
		// 说明是子评论，需要找到根评论，然后加载所有的子评论
		parentId, err = mysql_repo.CommentRepository.GetRootCommentId(sqls.DB(), comment.ParentCommentId)
		if err != nil {
			zap.L().Error("find root comment id error in logic.GetCommentDetail()", zap.Error(err))
			return
		}
	}
	// 通过根评论，获取子评论的id
	subCommentIds, err := redis_repo.GetSubCommentIdsByRootComment(strconv.FormatInt(parentId, 10))
	if err != nil {
		zap.L().Error("get sub comment ids error in logic.GetCommentDetail()", zap.Error(err))
		return
	}

	// 通过这些评论ID，获取所有的评论信息
	rootComment := mysql_repo.CommentRepository.Get(sqls.DB(), parentId)
	username, err := GetUsernameById(rootComment.UserId)
	if err != nil {
		zap.L().Error("get username error in logic.GetCommentDetail()", zap.Error(err))
		return
	}

	voteNum, err := redis_repo.GetCommentVoteNumById(strconv.FormatInt(rootComment.CommentId, 10))
	if err != nil {
		zap.L().Error("get vote num by comment id error", zap.Error(err))
		return
	}

	res.Username = username
	res.Content = rootComment.Content
	res.VoteNum = voteNum
	res.UpdateAt = rootComment.UpdateAt
	res.SubComment = make([]models.ResponseComment, len(subCommentIds))
	for i := 1; i < len(subCommentIds); i++ {
		id := subCommentIds[i]
		id_, _ := strconv.ParseInt(id, 10, 64)
		comment, err = GetCommentById(id_)
		if err != nil {
			zap.L().Error("get comment error in logic.GetCommentDetail()", zap.Error(err))
			return models.ResponseComment{}, err
		}
		//comment = mysql_repo.CommentRepository.Get(sqls.DB(), id_)
		username, err = GetUsernameById(comment.UserId)
		if err != nil {
			zap.L().Error("get username error in logic.GetCommentDetail()", zap.Error(err))
			return
		}
		voteNum, err = redis_repo.GetCommentVoteNumById(id)
		if err != nil {
			zap.L().Error("get vote num by comment id error", zap.Error(err))
			return
		}
		res.SubComment[i-1].Username = username
		res.SubComment[i-1].Content = comment.Content
		res.SubComment[i-1].VoteNum = voteNum
		res.SubComment[i-1].UpdateAt = comment.UpdateAt
		if comment.ParentCommentId != rootComment.CommentId && comment.ParentCommentId != 0 {
			// 说明是对另一个子评论的追评
			replyComment, err1 := GetCommentById(comment.ParentCommentId)
			if err1 != nil {
				zap.L().Error("get comment error in logic.GetCommentDetail()", zap.Error(err1))
				return models.ResponseComment{}, err1
			}
			replyUserName, err1 := GetUsernameById(replyComment.UserId)
			if err1 != nil {
				zap.L().Error("get username error in logic.GetCommentDetail()", zap.Error(err1))
				return models.ResponseComment{}, err1
			}
			res.SubComment[i-1].ReplyTo = replyUserName
		}
	}
	return res, nil
}
