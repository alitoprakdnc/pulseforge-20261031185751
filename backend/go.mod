module github.com/pulseforge/backend

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/go-redis/redis/v9 v9.5.0
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/google/uuid v1.5.0
	github.com/joho/godotenv v1.5.1
	github.com/openai/openai-go v0.2.5
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.8.4
	github.com/swaggo/files v0.0.0-20231201123456
	github.com/swaggo/gin-swagger v1.6.0
	cloud.google.com/go/secretmanager/apiv1 v1.13.0
	gorm.io/driver/postgres v1.5.5
	gorm.io/gorm v1.25.4
	go.uber.org/zap v1.26.0
)