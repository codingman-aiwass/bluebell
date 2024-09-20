package controllers

import (
	"bluebell/dao/mysql_repo"
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"bluebell/pkg/sqls"
	"bluebell/pkg/validation"
	"bluebell/settings"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
	"strings"
)

// CreatePost 创建一个新的帖子
// @Summary 创建新帖子
// @Description 可按用户输入内容在特定社区创建给定主题和内容的帖子
// @Tags 帖子相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamPostCreate true "需要创建帖子的详细信息"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostCreate
// @Router /api/v1/post [post]
func CreatePost(c *gin.Context) {
	// 1. 获取绑定的参数
	PostEntry := new(models.Post)
	PostParam := new(models.ParamPostCreate)
	err := c.ShouldBindJSON(PostParam)
	if err != nil {
		zap.L().Error("bind post failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	PostEntry.Title = PostParam.Title
	PostEntry.Content = PostParam.Content
	PostEntry.CommunityID = PostParam.CommunityId
	// 获取author_id
	author_id := c.GetInt64(ContextUserIdKey)
	PostEntry.AuthorID = author_id
	// 设置post id
	PostEntry.PostId = snowflake.GenID()
	// 判断用户此时是否处于已经验证通过且未被禁言状态.也需要检查用户是否超过了一定时间内的发帖上限
	u := mysql_repo.UserRepository.Get(sqls.DB(), author_id)
	if !(u.Status == NORMAL_STATUS || u.Verified == EMAIL_VERFIED) {
		zap.L().Warn("This user is not allowed publishing post due to its status or verified")
		ResponseError(c, CODE_NOT_ALLOW_PUBLISH_POST)
		return
	}
	if err = validation.CheckPost(u, nil); err != nil {
		zap.L().Error("This user hit some strategy, fail to publish post", zap.Error(err))
		ResponseError(c, CODE_NOT_ALLOW_PUBLISH_POST)
		return
	}

	// 2.写入数据库
	err = logic.CreatePost(PostEntry)
	if err != nil {
		zap.L().Error("create post failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, CODE_SUCCESS)

}

// GetPostById 根据id获取指定帖子
// @Summary 根据id获取指定帖子
// @Description 可按post id 返回指定帖子
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param id path string true "post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/post/{id} [get]
func GetPostById(c *gin.Context) {
	postDetail := new(models.PostDetail)
	// 获取post id
	postId := c.Param("id")

	// 获取记录
	id, err := strconv.ParseInt(postId, 10, 64)
	if err != nil {
		zap.L().Error("parse postId failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}
	// 获取post详细信息
	post, err := logic.GetPostById(id)
	if err != nil {
		zap.L().Error("get post by id failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	// 获取username
	username, err := logic.GetUsernameById(post.AuthorID)
	if err != nil {
		zap.L().Error("get username by id failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	// 获取社区详细信息
	community, err := logic.GetCommunityById(post.CommunityID)
	if err != nil {
		zap.L().Error("get community by id failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}

	postDetail.Title = post.Title
	postDetail.AuthorName = username
	postDetail.Content = post.Content

	postDetail.YesVotes, postDetail.CommentNum, postDetail.ClickNums = logic.GetPostDetailedInfo1(post.PostId)

	postDetail.UpdateAt = post.UpdateAt
	postDetail.CommunityName = community.CommunityName

	ResponseSuccess(c, postDetail)

}

// GetPostList 分页获取所有post
// @Summary 分页获取所有post
// @Description 可按用户指定分页要求（若有）返回帖子列表
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamPostList false "page, size"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/posts [get]
//func GetPostList(c *gin.Context) {
//	// 1.处理参数
//	p := &models.ParamPostList{
//		Page: 1,
//		Size: 10,
//	}
//	if err := c.ShouldBindQuery(p); err != nil {
//		zap.L().Error("get post list failed with invalid params", zap.Error(err))
//		ResponseError(c, CODE_PARAM_ERROR)
//	}
//	//pageStr := c.DefaultQuery("page", "1")
//	//pageSizeStr := c.DefaultQuery("size", "10")
//	//page, err := strconv.ParseInt(pageStr, 10, 64)
//	//if err != nil {
//	//	zap.L().Error("parse page failed", zap.Error(err))
//	//	ResponseError(c, CODE_PARAM_ERROR)
//	//}
//	//pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
//	//if err != nil {
//	//	zap.L().Error("parse page size failed", zap.Error(err))
//	//	ResponseError(c, CODE_PARAM_ERROR)
//	//}
//
//	// 2.业务逻辑处理
//	// 获取post详细信息
//	posts, err := logic.GetPosts(p.Page-1, p.Size)
//	if err != nil {
//		return
//	}
//	postDetailList := make([]models.PostDetail, 0, len(posts))
//	for _, post := range posts {
//		// 获取username
//		username, err := logic.GetUsernameById(post.AuthorID)
//		if err != nil {
//			zap.L().Error("get username by id failed", zap.Error(err))
//			ResponseError(c, CODE_INTERNAL_ERROR)
//			return
//		}
//		// 获取社区详细信息
//		community, err := logic.GetCommunityById(post.CommunityID)
//		if err != nil {
//			zap.L().Error("get community by id failed", zap.Error(err))
//			ResponseError(c, CODE_INTERNAL_ERROR)
//			return
//		}
//		postDetail := models.PostDetail{
//			AuthorName: username,
//			Post:       &post,
//			Community:  community,
//		}
//
//		postDetailList = append(postDetailList, postDetail)
//	}
//	ResponseSuccess(c, &postDetailList)
//
//}

// VoteForPost 为帖子点赞/取消点赞/点踩
// @Summary 为帖子点赞/取消点赞/点踩
// @Description 为帖子点赞/取消点赞/点踩，并计算用户操作以后的帖子分数
// @Tags 帖子相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param object body models.ParamVotePost true "post id,vote"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseVotePost
// @Router /api/v1/post/vote [post]
func VoteForPost(c *gin.Context) {
	// 1. 解析参数，参数就设置为json格式，选取post_id 和 direction(-1 0 1) 分别代表 反对/取消/赞成
	votePost := new(models.ParamVotePost)
	err := c.ShouldBindJSON(votePost)
	if err != nil {
		zap.L().Error("bind post failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}
	// 获取userId
	userId, exists := c.Get(ContextUserIdKey)
	if !exists {
		zap.L().Error("get userId failed", zap.Error(err))
		ResponseError(c, CODE_NOT_LOGIN)
	}
	// 2. 逻辑层处理
	err = logic.VotePost(userId.(int64), votePost)
	if err != nil {
		zap.L().Error("vote post failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}

// GetPostList1 分页获取post简略信息
// @Summary 分页获取post简略信息
// @Description 可按用户指定分页要求（若有）返回特定community（若有）的post简略信息列表
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamPostList false "page, size"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/posts1 [get]
func GetPostList1(c *gin.Context) {
	// 1.处理参数
	param_list_query := new(models.ParamPostList)
	err := c.ShouldBindQuery(param_list_query)
	if err != nil {
		zap.L().Error("bind post list query failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}

	// 2.业务逻辑处理
	// 从redis中获取id列表，再根据这个id列表去redis中获取点赞/反对的数量
	//ids, err := logic.GetPostsIds(param_list_query)
	//if err != nil {
	//	zap.L().Error("get posts ids from redis_repo failed", zap.Error(err))
	//	ResponseError(c, CODE_INTERNAL_ERROR)
	//	return
	//}
	//yes_vote, err := logic.GetPostVotes(ids, "1")
	//if err != nil {
	//	zap.L().Error("get post yes votes failed", zap.Error(err))
	//	ResponseError(c, CODE_INTERNAL_ERROR)
	//}
	//no_vote, err := logic.GetPostVotes(ids, "-1")
	//if err != nil {
	//	zap.L().Error("get post no votes failed", zap.Error(err))
	//	ResponseError(c, CODE_INTERNAL_ERROR)
	//}

	// 获取post详细信息
	posts, err := logic.GetPostsWithOrder(param_list_query)
	if err != nil {
		zap.L().Error("get posts failed from mysql_repo error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	postDetailList := make([]models.PostDetail, 0, len(posts))
	for _, post := range posts {
		// 获取username
		username, err := logic.GetUsernameById(post.AuthorID)
		if err != nil {
			zap.L().Error("get username by id failed", zap.Error(err))
			ResponseError(c, CODE_INTERNAL_ERROR)
			return
		}
		// 获取社区详细信息
		//community, err := logic.GetCommunityById(post.CommunityID)
		//if err != nil {
		//	zap.L().Error("get community by id failed", zap.Error(err))
		//	ResponseError(c, CODE_INTERNAL_ERROR)
		//	return
		//}

		// 获取点赞/评论/浏览数
		voteNum, commentNum, _ := logic.GetPostDetailedInfo1(post.PostId)
		var content string
		if len(post.Content) > 50 {
			content = post.Content[:50]
		} else {
			content = post.Content
		}
		postDetail := models.PostDetail{
			Title:      post.Title,
			AuthorName: username,
			Content:    content,
			YesVotes:   voteNum,
			CommentNum: commentNum,
		}

		postDetailList = append(postDetailList, postDetail)
	}
	ResponseSuccess(c, postDetailList)
}

// GetPostList2 分页获取post更简略信息（类似CSDN评论区下的帖子推荐）
// @Summary 分页获取post更简略信息
// @Description 分页获取post更简略信息（类似CSDN评论区下的帖子推荐）
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamPostList2 false "page, size"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/posts2 [get]
func GetPostList2(c *gin.Context) {
	// 1.处理参数
	param_list_query := new(models.ParamPostList2)
	err := c.ShouldBindQuery(param_list_query)
	if err != nil {
		zap.L().Error("bind post list query failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	ids := param_list_query.PostIds[0]
	param_list_query.PostIds = strings.Split(ids, ",")

	// 2.根据ids获取需要的展示post信息
	var postDetails []models.PostDetail
	for _, id := range param_list_query.PostIds {
		id_, _ := strconv.ParseInt(id, 10, 64)
		post, err := logic.GetPostById(id_)
		username, _ := logic.GetUsernameById(post.AuthorID)
		// 获取点赞/评论/浏览数
		_, _, click := logic.GetPostDetailedInfo1(post.PostId)
		if err == nil {
			var content string
			if len(post.Content) < 20 {
				content = post.Content
			} else {
				content = post.Content[:20]
			}
			postDetail := models.PostDetail{
				Title:      post.Title,
				AuthorName: username,
				Content:    content,
				ClickNums:  click,
				UpdateAt:   post.UpdateAt,
			}
			postDetails = append(postDetails, postDetail)
		}
	}
	ResponseSuccess(c, postDetails)

}

// CollectPost 收藏帖子
// @Summary 收藏帖子
// @Description 收藏帖子
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param post-id query string true "post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseCollectPost
// @Router /api/v1/post/collect [post]
func CollectPost(c *gin.Context) {
	postId, err := strconv.ParseInt(c.Query("post-id"), 10, 64)
	if err != nil {
		zap.L().Error("Parse post id error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	userId := c.GetInt64(ContextUserIdKey)

	if err = logic.AddCollectPost(postId, userId); err != nil {
		zap.L().Error("add collect post error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	ResponseSuccess(c, nil)
}

// DeletePost 删除帖子
// @Summary 删除帖子
// @Description 删除帖子
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string true "Bearer 用户令牌"
// @Param post-id query string true "post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseDeletePost
// @Router /api/v1/post [delete]
func DeletePost(c *gin.Context) {
	postId, err := strconv.ParseInt(c.Query("post-id"), 10, 64)
	if err != nil {
		zap.L().Error("Parse post id error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	userId := c.GetInt64(ContextUserIdKey)
	if err = logic.DeletePost(postId, userId); err != nil {
		zap.L().Error("Delete post error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}

	ResponseSuccess(c, nil)
}

// GetPostLink 获取帖子分享链接
// @Summary 获取帖子分享链接
// @Description 获取帖子分享链接
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param post-id query string true "post id"
// @Security ApiKeyAuth
// @Router /api/v1/post/link [get]
func GetPostLink(c *gin.Context) {
	postId, err := strconv.ParseInt(c.Query("post-id"), 10, 64)
	if err != nil {
		zap.L().Error("Parse post id error", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	url := fmt.Sprintf("%s:%d/api/v1/post/%d",
		settings.GlobalSettings.AppCfg.Host, settings.GlobalSettings.AppCfg.Port, postId)
	ResponseSuccess(c, url)
}
