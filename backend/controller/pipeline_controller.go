package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"

	"pulseforge/backend/logger"
	"pulseforge/backend/models"
	"pulseforge/backend/service"
)

// PipelineController groups handlers that operate on pipelines and related resources.
type PipelineController struct {
	pipelineService service.PipelineService
	aiService       service.AIService
	alertService    service.AlertService
}

// NewPipelineController registers pipeline routes onto the provided router group.
func NewPipelineController(
	rg *gin.RouterGroup,
	pipelineSvc service.PipelineService,
	aiSvc service.AIService,
	alertSvc service.AlertService,
) {
	c := &PipelineController{
		pipelineService: pipelineSvc,
		aiService:       aiSvc,
		alertService:    alertSvc,
	}

	pipelines := rg.Group("/pipelines")
	{
		pipelines.GET("", c.ListPipelines)                         // GET /api/pipelines
		pipelines.GET("/:id", c.GetPipeline)                       // GET /api/pipelines/:id
		pipelines.GET("/:id/suggestions", c.GetAISuggestions)    // GET /api/pipelines/:id/suggestions
		pipelines.GET("/:id/alerts", c.ListAlerts)                // GET /api/pipelines/:id/alerts
		pipelines.POST("/:id/alerts", c.CreateAlert)               // POST /api/pipelines/:id/alerts
		pipelines.DELETE("/:id/alerts/:alertId", c.DeleteAlert) // DELETE /api/pipelines/:id/alerts/:alertId
	}
}

// getUserID extracts the authenticated user ID from the Gin context.
// The auth middleware stores the user ID under the key "userID".
func getUserID(c *gin.Context) (uint, error) {
	val, exists := c.Get("userID")
	if !exists {
		return 0, errors.New("user not authenticated")
	}
	uid, ok := val.(uint)
	if !ok {
		return 0, errors.New("invalid user ID type")
	}
	return uid, nil
}

// parseUintParam extracts a uint path parameter from the request.
func parseUintParam(c *gin.Context, name string) (uint, error) {
	param := c.Param(name)
	if param == "" {
		return 0, errors.New("missing parameter: " + name)
	}
	val, err := strconv.ParseUint(param, 10, 0)
	if err != nil {
		return 0, errors.Wrap(err, "invalid parameter: "+name)
	}
	return uint(val), nil
}

// ListPipelines handles GET /api/pipelines
func (pc *PipelineController) ListPipelines(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		logger.Error("unauthenticated request to ListPipelines", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	pipelines, err := pc.pipelineService.ListPipelines(userID)
	if err != nil {
		logger.Error("failed to list pipelines", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve pipelines"})
		return
	}
	c.JSON(http.StatusOK, pipelines)
}

// GetPipeline handles GET /api/pipelines/:id
func (pc *PipelineController) GetPipeline(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		logger.Error("unauthenticated request to GetPipeline", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	pipelineID, err := parseUintParam(c, "id")
	if err != nil {
		logger.Warn("invalid pipeline id", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return
	}

	pipeline, err := pc.pipelineService.GetPipeline(userID, pipelineID)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not found"})
			return
		}
		logger.Error("failed to fetch pipeline", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve pipeline"})
		return
	}
	c.JSON(http.StatusOK, pipeline)
}

// GetAISuggestions handles GET /api/pipelines/:id/suggestions
// Supports optional pagination via ?limit=&offset=
func (pc *PipelineController) GetAISuggestions(c *gin.Context) {
	pipelineID, err := parseUintParam(c, "id")
	if err != nil {
		logger.Warn("invalid pipeline id for suggestions", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return
	}

	// Pagination defaults
	limit := 20
	offset := 0
	if lStr := c.Query("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := c.Query("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	suggestions, err := pc.aiService.GetSuggestions(pipelineID, limit, offset)
	if err != nil {
		logger.Error("failed to fetch AI suggestions", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve suggestions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"suggestions": suggestions,
		"limit":       limit,
		"offset":      offset,
	})
}

// AlertCreateRequest defines the payload for creating a new alert.
type AlertCreateRequest struct {
	Title       string `json:"title" binding:"required,min=3,max=200"`
	Severity    string `json:"severity" binding:"required,oneof=info warning critical"`
	Expression  string `json:"expression" binding:"required"` // e.g., a PromQL‑like condition
	NotifyEmail string `json:"notify_email" binding:"omitempty,email"`
	NotifySlack string `json:"notify_slack" binding:"omitempty,url"`
}

// ListAlerts handles GET /api/pipelines/:id/alerts
func (pc *PipelineController) ListAlerts(c *gin.Context) {
	pipelineID, err := parseUintParam(c, "id")
	if err != nil {
		logger.Warn("invalid pipeline id for alerts list", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return
	}

	alerts, err := pc.alertService.ListAlerts(pipelineID)
	if err != nil {
		logger.Error("failed to list alerts", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve alerts"})
		return
	}
	c.JSON(http.StatusOK, alerts)
}

// CreateAlert handles POST /api/pipelines/:id/alerts
func (pc *PipelineController) CreateAlert(c *gin.Context) {
	pipelineID, err := parseUintParam(c, "id")
	if err != nil {
		logger.Warn("invalid pipeline id for alert creation", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return
	}

	var req AlertCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Warn("invalid alert payload", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload", "details": err.Error()})
		return
	}

	alert := &models.Alert{
		PipelineID:  pipelineID,
		Title:       req.Title,
		Severity:    req.Severity,
		Expression: req.Expression,
		NotifyEmail: req.NotifyEmail,
		NotifySlack: req.NotifySlack,
	}

	created, err := pc.alertService.CreateAlert(pipelineID, alert)
	if err != nil {
		logger.Error("failed to create alert", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create alert"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// DeleteAlert handles DELETE /api/pipelines/:id/alerts/:alertId
func (pc *PipelineController) DeleteAlert(c *gin.Context) {
	pipelineID, err := parseUintParam(c, "id")
	if err != nil {
		logger.Warn("invalid pipeline id for alert deletion", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid pipeline id"})
		return
	}
	alertID, err := parseUintParam(c, "alertId")
	if err != nil {
		logger.Warn("invalid alert id for deletion", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid alert id"})
		return
	}

	if err := pc.alertService.DeleteAlert(pipelineID, alertID); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
			return
		}
		logger.Error("failed to delete alert", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete alert"})
		return
	}
	c.Status(http.StatusNoContent)
}