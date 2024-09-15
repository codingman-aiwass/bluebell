package controllers

import (
	"bluebell/models"
	"bluebell/pkg/strs"
	"bluebell/settings"
	"fmt"
	"github.com/dchest/captcha"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// GetCaptchaInfo 获取Captcha相关信息
// @Summary 获取Captcha相关信息
// @Description 获取Captcha相关信息：captcha-id, captcha-url
// @Tags 验证码相关接口
// @Produce application/json
// @Success 200 {object} _ResponseCaptchaInfo
// @Router /api/v1/captcha/request [get]
func GetCaptchaInfo(c *gin.Context) {
	captchaId := captcha.NewLen(4)
	host := settings.GlobalSettings.AppCfg.Host
	port := settings.GlobalSettings.AppCfg.Port
	captchaUrl := fmt.Sprintf("%s:%d/api/v1/captcha/show?captcha-id=%s&r=%s", host, port, captchaId, strs.UUID())
	ResponseSuccess(c, gin.H{
		"captchaId":  captchaId,
		"captchaUrl": captchaUrl,
	})
}

// GetShow 返回验证码图片
// @Summary 返回验证码图片
// @Description 返回captcha验证码图片
// @Tags 验证码相关接口
// @Produce image/png
// @Param captcha-id query string true "captcha id"
// @Router /api/v1/captcha/show [get]
func GetShow(c *gin.Context) {
	captchaId := c.DefaultQuery("captcha-id", "")
	if len(captchaId) == 0 {
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}

	if !captcha.Reload(captchaId) {
		ResponseError(c, CODE_PARAM_ERROR)
		return
	}
	ResponseCaptcha(c, captchaId)
}

// GetVerify 检查验证码是否正确
// @Summary 检查验证码是否正确
// @Description 检查验证码是否正确
// @Tags 验证码相关接口
// @Param object query models.ParamCaptchaInfo false "captchaId, captchaCode"
// @Success 200 {object} _ResponseCaptchaVerification
// @Router /api/v1/captcha/verify [get]
func GetVerify(c *gin.Context) {
	// 1.处理参数
	p := &models.ParamCaptchaInfo{
		Id:   "",
		Code: "",
	}
	if err := c.ShouldBindQuery(p); err != nil {
		zap.L().Error("get captcha info failed with invalid params", zap.Error(err))
		ResponseError(c, CODE_PARAM_ERROR)
	}

	success := captcha.VerifyString(p.Id, p.Code)
	ResponseSuccess(c, success)
}
