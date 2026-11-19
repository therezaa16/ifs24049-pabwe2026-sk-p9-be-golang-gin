// Package server merangkai seluruh rute Gin, Swagger UI, dan handler 404.
package server

import (
	"net/http"
	"time"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// NewRouter membuat engine Gin lengkap dengan seluruh endpoint.
func NewRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), middleware.CORS(), middleware.RateLimiter(6000, time.Minute))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		c.Data(http.StatusOK, "application/json; charset=utf-8", docs.JSON(c.Request.Host, scheme))
	})

	authSvc := auth.NewService(auth.NewRepository(db), cfg.AppKey)
	authH := auth.NewHandler(authSvc)
	postH := post.NewHandler(post.NewService(post.NewRepository(db)))
	aiH := ai.NewHandler(ai.NewService(
		ai.NewSearchClient(cfg.SearchEngineEndpoint, cfg.OpenAIKey),
		ai.NewChatClient(cfg.OpenAIEndpoint, cfg.OpenAIKey, cfg.OpenAIModel),
	))

	api := r.Group("/api")

	pub := api.Group("/public")
	pub.GET("/categories", postH.PublicCategories)
	pub.GET("/posts", postH.PublicPosts)
	pub.GET("/posts/:slug", postH.PublicPostDetail)

	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)
	api.POST("/auth/logout", authH.Logout)

	guard := middleware.Auth(cfg.AppKey)

	profile := api.Group("/auth", guard)
	profile.GET("/profile", authH.Profile)
	profile.PUT("/profile", authH.UpdateProfile)

	posts := api.Group("/posts", guard)
	posts.GET("", postH.List)
	posts.POST("", postH.Create)
	posts.GET("/stats", postH.Stats)
	posts.POST("/ai-generate", aiH.Generate)
	posts.GET("/:id", postH.Get)
	posts.PUT("/:id", postH.Update)
	posts.PATCH("/:id/toggle-publish", postH.TogglePublish)
	posts.DELETE("/:id", postH.Delete)

	admin := api.Group("/admin", guard, middleware.RequireRole("admin"))
	admin.GET("/users", postH.AdminUsers)
	admin.GET("/posts", postH.AdminPosts)
	admin.GET("/users/:id/posts", postH.AdminUserPosts)

	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})
	return r
}
