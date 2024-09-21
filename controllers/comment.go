package controllers

import (
	"bluebell/dao/mysql_repo"
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
