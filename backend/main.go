package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulseforge/backend/config"
	"pulseforge/backend/logger"
	"pulseforge/backend/router"
	"pulseforge/backend/service"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	// -------------------------------------------------------------------------
	// Load configuration
	// -------------------------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// -------------------------------------------------------------------------
	// Initialise structured logger
	// -------------------------------------------------------------------------
	log := logger.NewLogger(cfg.LogLevel)

	// -------------------------------------------------------------------------
	// Initialise PostgreSQL connection (GORM)
	// -------------------------------------------------------------------------
	db, err := initPostgres(cfg, log)
	if err != nil {
		log.Fatal("database initialization failed", zap.Error(err))
	}
	// Ensure DB is reachable
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("unable to obtain generic DB from GORM", zap.Error(err))
	}
	if err := sqlDB.Ping(); err != nil {
		log.Fatal("cannot ping database", zap.Error(err))
	}
	log.Info("PostgreSQL connection established")

	// -------------------------------------------------------------------------
	// Initialise Redis client
	// -------------------------------------------------------------------------
	redisClient := initRedis(cfg, log)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatal("cannot ping Redis", zap.Error(err))
	}
	log.Info("Redis connection established")

	// -------------------------------------------------------------------------
	// Initialise RabbitMQ connection
	// -------------------------------------------------------------------------
	rabbitConn, err := initRabbitMQ(cfg, log)
	if err != nil {
		log.Fatal("RabbitMQ connection failed", zap.Error(err))
	}
	log.Info("RabbitMQ connection established")

	// -------------------------------------------------------------------------
	// Start background workers
	// -------------------------------------------------------------------------
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()

	// Queue monitor worker
	go func() {
		if err := service.StartQueueMonitor(workerCtx, db, redisClient, cfg, log); err != nil {
			log.Error("queue monitor exited with error", zap.Error(err))
			workerCancel()
		}
	}()

	// AI suggester worker
	go func() {
		if err := service.StartAISuggester(workerCtx, db, redisClient, rabbitConn, cfg, log); err != nil {
			log.Error("AI suggester exited with error", zap.Error(err))
			workerCancel()
		}
	}()

	// -------------------------------------------------------------------------
	// Setup HTTP router
	// -------------------------------------------------------------------------
	httpRouter := router.SetupRouter(db, redisClient, rabbitConn, log)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.ServerPort),
		Handler: httpRouter,
	}

	// -------------------------------------------------------------------------
	// Run server in a goroutine
	// -------------------------------------------------------------------------
	go func() {
		log.Info("starting HTTP server", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server crashed", zap.Error(err))
		}
	}()

	// -------------------------------------------------------------------------
	// Graceful shutdown handling
	// -------------------------------------------------------------------------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info("shutdown signal received")

	// Context for shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Stop accepting new requests
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	} else {
		log.Info("HTTP server gracefully stopped")
	}

	// Cancel background workers
	workerCancel()
	// Give workers a moment to finish
	time.Sleep(2 * time.Second)

	// Close external resources
	if err := rabbitConn.Close(); err != nil {
		log.Error("error closing RabbitMQ connection", zap.Error(err))
	}
	if err := redisClient.Close(); err != nil {
		log.Error("error closing Redis client", zap.Error(err))
	}
	if err := sqlDB.Close(); err != nil {
		log.Error("error closing DB connection", zap.Error(err))
	}

	log.Info("application shutdown complete")
}

// initPostgres creates a GORM DB instance using configuration values.
func initPostgres(cfg *config.Config, log *zap.Logger) (*gorm.DB, error) {
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}
	return db, nil
}

// initRedis creates a Redis client.
func initRedis(cfg *config.Config, log *zap.Logger) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword, // empty string means no password
		DB:           cfg.RedisDB,
		MinRetryBackoff: 250 * time.Millisecond,
		MaxRetryBackoff: 5 * time.Second,
	})
}

// initRabbitMQ establishes a connection to RabbitMQ.
func initRabbitMQ(cfg *config.Config, log *zap.Logger) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}
	return conn, nil
}