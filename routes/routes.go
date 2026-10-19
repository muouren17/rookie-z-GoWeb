package routes

import (
	"net/http"
	"time"
	"webapp/controllers"
	"webapp/logger"
	"webapp/middlewares"

	_ "webapp/docs" // 千万不要忘了导入把你上一步生成的docs

	files "github.com/swaggo/files"
	gs "github.com/swaggo/gin-swagger"

	"github.com/gin-gonic/gin"
)

func Setup(mode string) *gin.Engine {
	if mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(logger.GinLogger(), logger.GinRecovery(true), middlewares.RateLimitMiddleware(10*time.Millisecond, 50))
	v1 := r.Group("/api/v1")
	v1.POST("/signup", controllers.SignUpHandler)
	v1.POST("login", controllers.LoginHandler)

	v1.Use(middlewares.JWTAuthMiddleware())
	{
		v1.GET("/community", controllers.CommunityHandler)
		v1.GET("/community/:id", controllers.CommunityDetailHandler)
		v1.POST("/post", controllers.CommunityCreatePostHandler)
		v1.GET("/post/:id", controllers.GetPostDetailHandler)
		v1.GET("/posts", controllers.GetPostListHandler)
		v1.GET("/posts2", controllers.GetPostListHandler2)

		v1.POST("/vote", controllers.PostVoteHandler)
	}

	r.POST("/refresh", controllers.RefreshHandler)
	r.GET("/swagger/*any", gs.WrapHandler(files.Handler))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"response": "404",
		})
	})
	return r
}
