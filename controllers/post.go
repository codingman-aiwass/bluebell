package controllers

import (
	"bluebell/logic"
	"bluebell/models"
	"bluebell/pkg/snowflake"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"strconv"
)

// CreatePost 创建一个新的帖子
// @Summary 创建新帖子
// @Description 可按用户输入内容在特定社区创建给定主题和内容的帖子
// @Tags 帖子相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object body models.Post true "需要创建帖子的详细信息"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostCreate
// @Router /api/v1/post [post]
func CreatePost(c *gin.Context) {
	// 1. 获取绑定的参数
	PostEntry := new(models.Post)
	err := c.ShouldBindJSON(PostEntry)
	if err != nil {
		zap.L().Error("bind post failed", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	// 获取author_id
	author_id, _ := c.Get(ContextUserIdKey)
	PostEntry.AuthorID, _ = author_id.(int64)
	// 设置post id
	PostEntry.ID = snowflake.GenID()

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
// @Param object query string true "post id"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/post/:id [get]
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

	postDetail.AuthorName = username
	postDetail.Post = post
	postDetail.Community = community

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
func GetPostList(c *gin.Context) {
	// 1.处理参数
	p := &models.ParamPostList{
		Page: 1,
		Size: 10,
	}
	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("get post list failed with invalid params", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}
	//pageStr := c.DefaultQuery("page", "1")
	//pageSizeStr := c.DefaultQuery("size", "10")
	//page, err := strconv.ParseInt(pageStr, 10, 64)
	//if err != nil {
	//	zap.L().Error("parse page failed", zap.Error(err))
	//	ResponseError(c, CODE_PARAM_ERROR)
	//}
	//pageSize, err := strconv.ParseInt(pageSizeStr, 10, 64)
	//if err != nil {
	//	zap.L().Error("parse page size failed", zap.Error(err))
	//	ResponseError(c, CODE_PARAM_ERROR)
	//}

	// 2.业务逻辑处理
	// 获取post详细信息
	posts, err := logic.GetPosts(p.Page-1, p.Size)
	if err != nil {
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
		community, err := logic.GetCommunityById(post.CommunityID)
		if err != nil {
			zap.L().Error("get community by id failed", zap.Error(err))
			ResponseError(c, CODE_INTERNAL_ERROR)
			return
		}
		postDetail := models.PostDetail{
			AuthorName: username,
			Post:       post,
			Community:  community,
		}

		postDetailList = append(postDetailList, postDetail)
	}
	ResponseSuccess(c, &postDetailList)

}

// VoteForPost 为帖子点赞/取消点赞/点踩
// @Summary 为帖子点赞/取消点赞/点踩
// @Description 为帖子点赞/取消点赞/点踩，并计算用户操作以后的帖子分数
// @Tags 帖子相关接口
// @Accept application/json
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object body models.ParamVotePost true "post id,vote"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponseVotePost
// @Router /api/v1/vote-post [post]
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

// GetPostList2 分页获取所有post升级版，可以根据community查询
// @Summary 分页获取所有post升级版
// @Description 可按用户指定分页要求（若有）返回特定community的帖子列表
// @Tags 帖子相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query models.ParamPostList false "page, size"
// @Security ApiKeyAuth
// @Success 200 {object} _ResponsePostDetail
// @Router /api/v1/posts2 [get]
func GetPostList2(c *gin.Context) {
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
	ids, err := logic.GetPostsIds(param_list_query)
	if err != nil {
		zap.L().Error("get posts ids from redis failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	yes_vote, err := logic.GetPostVotes(ids, "1")
	if err != nil {
		zap.L().Error("get post yes votes failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
	}
	no_vote, err := logic.GetPostVotes(ids, "-1")
	if err != nil {
		zap.L().Error("get post no votes failed", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
	}

	// 获取post详细信息
	posts, err := logic.GetPostsWithOrder(param_list_query)
	if err != nil {
		zap.L().Error("get posts failed from mysql error", zap.Error(err))
		ResponseError(c, CODE_INTERNAL_ERROR)
		return
	}
	postDetailList := make([]models.PostDetail, 0, len(posts))
	for idx, post := range posts {
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
		postDetail := models.PostDetail{
			AuthorName: username,
			YesVotes:   yes_vote[idx],
			NoVotes:    no_vote[idx],
			Post:       post,
			Community:  community,
		}

		postDetailList = append(postDetailList, postDetail)
	}
	ResponseSuccess(c, &postDetailList)
}
