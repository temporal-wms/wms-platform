module github.com/wms-platform/services/sortation-service

go 1.24.0

require (
	github.com/gin-gonic/gin v1.10.1
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.11.1
	github.com/wms-platform/shared v0.0.0
	go.mongodb.org/mongo-driver v1.13.1
)

replace github.com/wms-platform/shared => ../../shared
