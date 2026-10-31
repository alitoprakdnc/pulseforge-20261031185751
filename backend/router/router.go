package router

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/pulseforge/backend/controller"
	"github.com/pulseforge/backend/middleware"
)

// SetupRouter creates and configures the Gin engine with all routes,
// middleware and controller handlers. It expects concrete controller
// instances that already have their dependent services injected.
func SetupRouter(
	logger *zap.Logger,
	webhookCtrl *controller.WebhookController,
	pipelineCtrl *controller.PipelineController,
	alertCtrl *controller.AlertController,
) *gin.Engine {
	// Use release mode for production; can be overridden by env in main.go if needed.
	gin.SetMode(gin.ReleaseMode)

	// Create a new Gin engine without any default middleware.
	r := gin.New()

	// Global middleware.
	r.Use(gin.Recovery())
	r.Use(requestLogger(logger))

	// -------------------------------------------------------------------------
	// Public / health‑check endpoint (no auth, no HMAC).
	// -------------------------------------------------------------------------
	r.GET("/api/health", middleware.HealthCheckHandler())

	// -------------------------------------------------------------------------
	// Webhook endpoints – protected only by constant‑time HMAC validation.
	// -------------------------------------------------------------------------
	webhookGroup := r.Group("/webhook")
	webhookGroup.Use(middleware.HMACMiddleware())
	{
		// POST /webhook/:provider  →  GitHub, GitLab, CircleCI etc.
		webhookGroup.POST("/:provider", webhookCtrl.HandleWebhook)
	}

	// -------------------------------------------------------------------------
	// Authenticated API – JWT authentication required.
	// -------------------------------------------------------------------------
	apiGroup := r.Group("/api")
	apiGroup.Use(middleware.AuthMiddleware())
	{
		// Pipeline CRUD & query endpoints.
		pipelines := apiGroup.Group("/pipelines")
		{
			pipelines.GET("", pipelineCtrl.ListPipelines)          // GET /api/pipelines
			pipelines.POST("", pipelineCtrl.CreatePipeline)        // POST /api/pipelines
			pipelines.GET("/:id", pipelineCtrl.GetPipeline)      // GET /api/pipelines/:id
			pipelines.PUT("/:id", pipelineCtrl.UpdatePipeline)    // PUT /api/pipelines/:id
			pipelines.DELETE("/:id", pipelineCtrl.DeletePipeline) // DELETE /api/pipelines/:id

			// Nested routes for jobs & AI suggestions.
			pipelines.GET("/:id/jobs", pipelineCtrl.ListJobs)                     // GET /api/pipelines/:id/jobs
			pipelines.GET("/:id/jobs/:jobId/suggestion", pipelineCtrl.GetSuggestion) // GET /api/pipelines/:id/jobs/:jobId/suggestion
		}

		// Alert management endpoints.
		alerts := apiGroup.Group("/alerts")
		{
			alerts.GET("", alertCtrl.ListAlerts)          // GET /api/alerts
			alerts.POST("", alertCtrl.CreateAlert)       // POST /api/alerts
			alerts.GET("/:id", alertCtrl.GetAlert)       // GET /api/alerts/:id
			alerts.PUT("/:id", alertCtrl.UpdateAlert)    // PUT /api/alerts/:id
			alerts.DELETE("/:id", alertCtrl.DeleteAlert) // DELETE /api/alerts/:id
		}
	}

	return r
}

// requestLogger returns a Gin middleware that logs each request using the
// provided zap.Logger. It records method, path, status, latency and client IP.
func requestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		logger.Info("http request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}