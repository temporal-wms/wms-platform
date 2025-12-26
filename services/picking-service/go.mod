module github.com/wms-platform/picking-service

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/google/uuid v1.6.0
	github.com/wms-platform/shared v0.0.0
	go.mongodb.org/mongo-driver v1.13.1
	go.temporal.io/sdk v1.26.1
)

replace github.com/wms-platform/shared => ../../shared
