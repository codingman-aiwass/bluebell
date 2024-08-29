package controllers

import (
	"bluebell/logic"
	"github.com/gin-gonic/gin"
	"strconv"
)

// GetAllCommunities 获取所有社区信息
// @Summary 获取所有社区信息
// @Description 获取所有社区信息
// @Tags 社区相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Success 200 {object} _ResponseCommunities
// @Router /api/v1/community [get]
func GetAllCommunities(c *gin.Context) {
	// 1. 处理参数，但是此处没有参数需要处理
	// 2. 调用逻辑层，返回值应该是一个list和error
	communities, err := logic.GetAllCommunities()
	if err != nil {
		ResponseError(c, CODE_NO_ROW_IN_DB)
		return
	}
	ResponseSuccess(c, communities)

}

// GetCommunityById 获取指定社区信息
// @Summary 获取指定社区信息
// @Description 根据ID获取指定社区信息
// @Tags 社区相关接口
// @Produce application/json
// @Param Authorization header string false "Bearer 用户令牌"
// @Param object query string true "community id"
// @Success 200 {object} _ResponseCommunities
// @Router /api/v1/community/:id [get]
func GetCommunityById(c *gin.Context) {
	// 1. 处理参数
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		// 说明传进来的id有误，无法解析成int
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	// 2. 调用逻辑层
	community, err := logic.GetCommunityById(id)
	if err != nil {
		ResponseError(c, CODE_NO_ROW_IN_DB)
		return
	}
	ResponseSuccess(c, community)

}
