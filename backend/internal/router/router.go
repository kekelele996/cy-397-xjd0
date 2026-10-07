package router

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/contractapi/contractapi/internal/handler"
	"github.com/contractapi/contractapi/internal/middleware"
	"github.com/contractapi/contractapi/pkg/jwtutil"
)

// New 集中注册中间件与路由。
func New(
	logger *slog.Logger,
	jwtManager *jwtutil.Manager,
	authHandler *handler.AuthHandler,
	templateHandler *handler.TemplateHandler,
	contractHandler *handler.ContractHandler,
	ticketHandler *handler.TicketHandler,
	knowledgeHandler *handler.KnowledgeHandler,
) *gin.Engine {
	engine := gin.New()
	engine.Use(
		middleware.Recovery(logger),
		middleware.RequestID(),
		middleware.Logger(logger),
		middleware.CORS(),
		middleware.ErrorHandler(logger),
	)

	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := engine.Group("/api/v1")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)

		api.GET("/templates", templateHandler.List)
		api.GET("/templates/:id", templateHandler.Get)
		api.GET("/faqs", knowledgeHandler.Search)
		api.GET("/faqs/:id", knowledgeHandler.Get)

		authed := api.Group("")
		authed.Use(middleware.JWTAuth(jwtManager))
		{
			authed.GET("/templates/favorites", templateHandler.Favorites)
			authed.POST("/templates/:id/favorite", templateHandler.Favorite)
			authed.DELETE("/templates/:id/favorite", templateHandler.Unfavorite)

			authed.POST("/admin/templates", templateHandler.Create)
			authed.PUT("/admin/templates/:id", templateHandler.Update)
			authed.DELETE("/admin/templates/:id", templateHandler.Delete)

			authed.POST("/contracts", contractHandler.Create)
			authed.GET("/contracts", contractHandler.List)
			authed.GET("/contracts/:id", contractHandler.Get)
			authed.POST("/contracts/:id/submit", contractHandler.Submit)
			authed.POST("/contracts/:id/sign", contractHandler.Sign)
			authed.POST("/contracts/:id/expire", contractHandler.Expire)
			authed.GET("/contracts/:id/signers", contractHandler.Signers)
			authed.GET("/contracts/:id/export", contractHandler.Export)

			authed.POST("/tickets", ticketHandler.Create)
			authed.GET("/tickets", ticketHandler.List)
			authed.GET("/tickets/:id", ticketHandler.Get)
			authed.POST("/tickets/:id/replies", ticketHandler.AddReply)
			authed.GET("/tickets/:id/replies", ticketHandler.Replies)
			authed.PATCH("/tickets/:id/status", ticketHandler.UpdateStatus)

			authed.POST("/admin/faqs", knowledgeHandler.Create)
			authed.PUT("/admin/faqs/:id", knowledgeHandler.Update)
			authed.DELETE("/admin/faqs/:id", knowledgeHandler.Delete)
		}
	}

	return engine
}
