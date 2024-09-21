package controllers

import (
	"bluebell/dao/mysql_repo"
	"bluebell/dao/redis_repo"
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"bluebell/pkg/validation"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

// CreateComment 创建一个新的评论
// @Summary 创建新评论
// @Description 可按用户输入内容在指定帖子/评论下创建给定评论
// @Tags 评论相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamCreateComment true "评论信息"
// @Security ApiKeyAuth
// @Success 200 {object} _GeneralResponse
// @Router /api/v1/comment [post]
func CreateComment(c *gin.Context) {
	CommentEntry := new(models.Comment)
	CommentParam := new(models.ParamCreateComment)
	err := c.ShouldBindJSON(CommentParam)
	if err != nil {
		zap.L().Error("bind comment failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}

	CommentEntry.PostId = CommentParam.PostId
	CommentEntry.Content = CommentParam.Content
	CommentEntry.CommentId = snowflake.GenID()
	CommentEntry.UserId = c.GetInt64(ContextUserIdKey)
	CommentEntry.ParentCommentId = CommentParam.ParentCommentId

	// 判断用户是否在该帖子作者的黑名单上
	flag, err := logic.CheckInBlacklist(CommentEntry.UserId, CommentParam.PostId)
	if err != nil {
		zap.L().Error("call logic.CheckInBlacklist error...", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	if flag {
		ResponseError(c, CODE_NOT_ALLOW_PUBLISH_COMMENT)
		return
	}
	// 判断是否触发规则
	u := mysql_repo.UserRepository.Get(sqls.DB(), CommentEntry.UserId)
	if err = validation.CheckComment(u, CommentEntry); err != nil {
		zap.L().Error("This user hit some strategy, fail to publish post", zap.Error(err))
		ResponseError(c, CODE_NOT_ALLOW_PUBLISH_COMMENT)
		return
	}

	// 写入数据库
	err = logic.CreateComment(CommentEntry)
	if err != nil {
		zap.L().Error("fail to save comment to the database...", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}

// VoteForComment 为评论点赞/取消点赞/点踩
// @Summary 为评论点赞/取消点赞/点踩
// @Description 为评论点赞/取消点赞/点踩，并计算用户操作以后的帖子分数
// @Tags 评论相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamVoteComment true "post id,vote"
// @Security ApiKeyAuth
// @Success 200 {object} _GeneralResponse
// @Router /api/v1/comment/vote [post]
func VoteForComment(c *gin.Context) {
	// 1. 解析参数，参数就设置为json格式，选取comment_id 和 direction(-1 0 1) 分别代表 反对/取消/赞成
	voteComment := new(models.ParamVoteComment)
	err := c.ShouldBindJSON(voteComment)
	if err != nil {
		zap.L().Error("bind post failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}
	// 获取userId
	userId := c.GetInt64(ContextUserIdKey)

	// 2. 逻辑层处理
	err = logic.VoteComment(userId, voteComment)
	if err != nil {
		zap.L().Error("vote post failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}

// DeleteComment 删除评论
// @Summary 删除评论
// @Description 删除评论
// @Tags 评论相关接口
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param comment-id query string true "comment id"
// @Security ApiKeyAuth
// @Success 200 {object} _GeneralResponse
// @Router /api/v1/comment [delete]
func DeleteComment(c *gin.Context) {
	commentId, err := strconv.ParseInt(c.Query("comment-id"), 10, 64)
	if err != nil {
		zap.L().Error("Parse comment id error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	userId := c.GetInt64(ContextUserIdKey)
	if err = logic.DeleteComment(commentId, userId); err != nil {
		zap.L().Error("Delete comment error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}

	ResponseSuccess(c, nil)
}

// GetCommentByPostId 根据post id分页获取其下的所有评论
// @Summary 根据post id分页获取其下的所有评论
// @Description 根据post id分页获取其下的所有评论
// @Tags 评论相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamGetCommentByPostId true "page size post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseComments
// @Router /api/v1/comment/by-post-id [get]
func GetCommentByPostId(c *gin.Context) {
	query := new(models.ParamGetCommentByPostId)
	err := c.ShouldBindQuery(query)
	if err != nil {
		zap.L().Error("bind get comment by post id  query failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	// 从bluebell:post:[post-id]中找到所有的根评论
	// 在根据这些根评论，去bluebell:comment:child_comment_record:[comment-id]中找出所有的子评论
	// 采用BFS的策略
	commentss, err := logic.GetCommentListByPostId(query)
	if err != nil {
		zap.L().Error("get comments and sub comments error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	// comments是一个二维数组，每个数组的第一个元素是根评论，后面的是子评论
	// 根评论需要显示用户名/评论内容/时间/点赞数
	// 子评论显示用户名/评论内容
	ResponseArr := make([]models.ResponseComment, len(commentss))
	for i, comments := range commentss {
		username, err := logic.GetUsernameById(comments[0].UserId)
		if err != nil {
			zap.L().Error("get username by user id error", zap.Error(err))
			ResponseError(c, CODE_INTERNAL_ERROR)
			return
		}
		voteNum, err := redis_repo.GetCommentVoteNumById(strconv.FormatInt(comments[0].CommentId, 10))
		if err != nil {
			zap.L().Error("get vote num by comment id error", zap.Error(err))
			ResponseError(c, CODE_INTERNAL_ERROR)
			return
		}
		ResponseArr[i].Username = username
		ResponseArr[i].Content = comments[0].Content
		ResponseArr[i].UpdateAt = comments[0].UpdateAt
		ResponseArr[i].VoteNum = voteNum
		ResponseArr[i].SubComment = make([]models.ResponseComment, len(comments)-1)
		for j := 1; j < len(comments); j++ {
			username, err = logic.GetUsernameById(comments[j].UserId)
			if err != nil {
				zap.L().Error("get username by user id error", zap.Error(err))
				ResponseError(c, CODE_INTERNAL_ERROR)
				return
			}
			ResponseArr[i].SubComment[j-1].Username = username
			ResponseArr[i].SubComment[j-1].Content = comments[j].Content
		}
	}
	ResponseSuccess(c, ResponseArr)
}

// GetTotalCommentsCount 根据post id，获取该post下所有的评论总数
// @Summary 根据post id，获取该post下所有的评论总数
// @Description 根据post id，获取该post下所有的评论总数
// @Tags 评论相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param post-id query string true "post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseCount
// @Router /api/v1/comment/total-count [get]
func GetTotalCommentsCount(c *gin.Context) {
	postId := c.Query("post-id")
	cnt, err := redis_repo.GetTotalCommentCountOfAPost(postId)
	if err != nil {
		zap.L().Error("get total comment count of a post error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, cnt)
}

// GetSubCommentsCount 根据comment id，获取该comment下所有的评论总数
// @Summary 根据comment id，获取该comment下所有的评论总数
// @Description 根据comment，获取该comment下所有的评论总数
// @Tags 评论相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param comment-id query string true "comment id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseCount
// @Router /api/v1/comment/sub-comments-count [get]
func GetSubCommentsCount(c *gin.Context) {
	commentId := c.Query("comment-id")
	cnt, err := redis_repo.GetSubCommentsCountOfAComment(commentId)
	if err != nil {
		zap.L().Error("get total comment count of a post error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, cnt)
}

// GetCommentsDetail 根据comment id，获取该comment的详细信息
// @Summary 根据comment id，获取该comment的详细信息
// @Description 根据comment，获取该comment的详细信息
// @Tags 评论相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param comment-id query string true "comment id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseComments
// @Router /api/v1/comment/comment-detail [get]
func GetCommentsDetail(c *gin.Context) {
	commentId := c.Query("comment-id")
	id, err := strconv.ParseInt(commentId, 10, 64)
	if err != nil {
		zap.L().Error("parse integer error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	res, err := logic.GetCommentDetail(id)
	if err != nil {
		zap.L().Error("get comment detail error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, res)

}
