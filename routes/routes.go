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
	// rate_limit1 := ratelimit.New(1, ratelimit.Per(time.Minute))
	// v1.GET("/test",middleware.BlockingRateLimitMiddleware(rate_limit1),testHandler)
	// 注册业务路由
	{
		v1.POST("/signup", controllers.SignUp)
		v1.POST("/login", controllers.SignIn)
		v1.POST("/login-via-email", controllers.SignInViaEmail)
		v1.GET("/refresh-access-token", controllers.RefreshAccessToken)
		v1.GET("/community", controllers.GetAllCommunities)
		v1.GET("/community/:id", controllers.GetCommunityById)
		v1.GET("/verify-email", controllers.VerifyEmail)
		v1.GET("/get-email-verification-code", middleware.NonBlockingRateLimitMiddleware(60), controllers.GetVerificationCode)
		captchas.GET("/request", controllers.GetCaptchaInfo)
		captchas.GET("/show", controllers.GetShow)
		captchas.GET("/verify", middleware.NonBlockingRateLimitMiddleware(60), controllers.GetVerify)

		v1.GET("/post/link", controllers.GetPostLink)
		v1.GET("/posts1", controllers.GetPostList1)
		v1.GET("/posts2", controllers.GetPostList2)

		v1.GET("/comment/by-post-id", controllers.GetCommentByPostId)
		v1.GET("/comment/total-count", controllers.GetTotalCommentsCount)
		v1.GET("/comment/sub-comments-count", controllers.GetSubCommentsCount)
		v1.GET("/comment/comment-detail", controllers.GetCommentsDetail)

	}
	v1.Use(middleware.JWTAuthMiddleware())
	{

		v1.POST("/edit-info", controllers.EditUserInfo)
		v1.POST("/post", controllers.CreatePost)
		v1.GET("/post/:id", controllers.GetPostById)
		v1.POST("/post/vote", controllers.VoteForPost)
		v1.POST("/send-email", controllers.SendEmail)
		v1.POST("/post/collect", controllers.CollectPost)
		v1.DELETE("/post", controllers.DeletePost)
		v1.POST("/comment", controllers.CreateComment)
		v1.POST("/comment/vote", controllers.VoteForComment)
		v1.DELETE("/comment", controllers.DeleteComment)

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
