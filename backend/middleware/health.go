package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/streadway/amqp"
	"gorm.io/gorm"

	"backend/logger"
)

// ComponentStatus represents the health of an individual component.
type ComponentStatus struct {
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// HealthResponse aggregates the health of all critical components.
type HealthResponse struct {
	Database  ComponentStatus `json:"database"`
	Redis     ComponentStatus `json:"redis"`
	RabbitMQ  ComponentStatus `json:"rabbitmq"`
	Timestamp string           `json:"timestamp"`
}

// NewHealthHandler creates a Gin handler that reports the health of the
// PostgreSQL database, Redis cache, and RabbitMQ connection.
//
// The handler returns HTTP 200 when all components are healthy, otherwise
// HTTP 503. The response body contains a JSON payload with per‑component
// status and an optional error message.
func NewHealthHandler(db *gorm.DB, rdb *redis.Client, rabbitConn *amqp.Connection) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var resp HealthResponse
		resp.Timestamp = time.Now().UTC().Format(time.RFC3339)

		// ---- PostgreSQL ----
		if sqlDB, err := db.DB(); err != nil {
			resp.Database = ComponentStatus{
				Healthy: false,
				Error:   "unable to retrieve sql.DB: " + err.Error(),
			}
			logger.Error("healthcheck: failed to get sql.DB", logger.Err(err))
		} else if err = sqlDB.PingContext(ctx); err != nil {
			resp.Database = ComponentStatus{
				Healthy: false,
				Error:   "ping failed: " + err.Error(),
			}
			logger.Error("healthcheck: postgres ping failed", logger.Err(err))
		} else {
			resp.Database = ComponentStatus{Healthy: true}
		}

		// ---- Redis ----
		if pong, err := rdb.Ping(ctx).Result(); err != nil {
			resp.Redis = ComponentStatus{
				Healthy: false,
				Error:   "ping failed: " + err.Error(),
			}
			logger.Error("healthcheck: redis ping failed", logger.Err(err))
		} else if pong != "PONG" {
			resp.Redis = ComponentStatus{
				Healthy: false,
				Error:   "unexpected ping response: " + pong,
			}
			logger.Error("healthcheck: unexpected redis ping response", logger.String("pong", pong))
		} else {
			resp.Redis = ComponentStatus{Healthy: true}
		}

		// ---- RabbitMQ ----
		if rabbitConn == nil {
			resp.RabbitMQ = ComponentStatus{
				Healthy: false,
				Error:   "connection is nil",
			}
			logger.Error("healthcheck: rabbitmq connection is nil")
		} else if rabbitConn.IsClosed() {
			resp.RabbitMQ = ComponentStatus{
				Healthy: false,
				Error:   "connection is closed",
			}
			logger.Error("healthcheck: rabbitmq connection is closed")
		} else {
			// Perform a lightweight channel open/close to verify connectivity.
			ch, err := rabbitConn.Channel()
			if err != nil {
				resp.RabbitMQ = ComponentStatus{
					Healthy: false,
					Error:   "channel creation failed: " + err.Error(),
				}
				logger.Error("healthcheck: rabbitmq channel creation failed", logger.Err(err))
			} else {
				_ = ch.Close()
				resp.RabbitMQ = ComponentStatus{Healthy: true}
			}
		}

		// Determine overall status.
		overallHealthy := resp.Database.Healthy && resp.Redis.Healthy && resp.RabbitMQ.Healthy
		if overallHealthy {
			c.JSON(http.StatusOK, resp)
		} else {
			c.JSON(http.StatusServiceUnavailable, resp)
		}
	}
}