package router

import (
	"bootstrap/telemetry"
	"github.com/gin-gonic/gin"
	"net/http"
	"quicknode/handler"
	"quicknode/service"
)

type Router struct {
	engine  *gin.Engine
	handler *handler.Handler
	apiKey  string
}

var l = telemetry.NewLogger()

func APIKeyMiddleware(validAPIKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != validAPIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"status":  "error",
				"message": "Invalid API key",
			})
			return
		}
		c.Next()
	}
}

func NewRouter(transactionService *service.TransactionService, apiKey string) *Router {
	l.Info("Initializing router...")
	r := &Router{
		engine:  gin.Default(),
		handler: handler.NewHandler(transactionService),
		apiKey:  apiKey,
	}

	r.registerRoutes()
	l.Info("API key: " + r.apiKey)

	return r
}

func (r *Router) registerRoutes() {
	quicknode := r.engine.Group("/quicknode")
	quicknode.Use(APIKeyMiddleware(r.apiKey))
	l.Info("API key: " + r.apiKey)

	{
		quicknode.POST("/sync", r.handler.SyncAddresses)
		quicknode.POST("/fetch", r.handler.FetchTransactions)
		quicknode.POST("/sync-address", r.handler.SyncAddressToQuicknode)
	}
}

func (r *Router) Engine() *gin.Engine {
	return r.engine
}
