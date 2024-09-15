package routes

import (
	"bluebell/controllers"
	_ "bluebell/docs"
	"bluebell/logger"
	"bluebell/middleware"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"
	"net/http"
)

func SetupRouter(mode string) *gin.Engine {
	if mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true))
	v1 := r.Group("/api/v1")
	captchas := v1.Group("/captcha")
	// 注册业务路由
	{
		v1.POST("/signup", controllers.SignUp)
		v1.POST("/login", controllers.SignIn)
		v1.POST("/login-via-email", controllers.SignInViaEmail)
		v1.GET("/refresh-access-token", controllers.RefreshAccessToken)
		v1.GET("/community", controllers.GetAllCommunities)
		v1.GET("/community/:id", controllers.GetCommunityById)
		v1.GET("/verify-email", controllers.VerifyEmail)
		v1.GET("/get-email-verification-code", controllers.GetVerificationCode)
		captchas.GET("/request", controllers.GetCaptchaInfo)
		captchas.GET("/show", controllers.GetShow)
		captchas.GET("/verify", controllers.GetVerify)
	}
	v1.Use(middleware.JWTAuthMiddleware())
	{
		v1.GET("/post/:id", controllers.GetPostById)
		v1.GET("/posts", controllers.GetPostList)
		v1.GET("/posts2", controllers.GetPostList2)
		v1.POST("/edit-info", controllers.EditUserInfo)
		v1.POST("/post", controllers.CreatePost)
		v1.POST("/vote-post", controllers.VoteForPost)
		v1.POST("/send-email", controllers.SendEmail)

		// 测试jwt-token，使得只有登录了的用户才能访问ping接口
		r.GET("/ping", middleware.JWTAuthMiddleware(), func(c *gin.Context) {
			c.JSON(200, gin.H{
				"message": "pong",
			})
		})
	}
	r.GET("/swagger/*any", gs.WrapHandler(swaggerFiles.Handler))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "404 not found",
		})
	})

	return r
}
