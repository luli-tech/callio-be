package system

import "github.com/gin-gonic/gin"

func RegisterRoutes(engine *gin.Engine, ctrl *HealthController) {
	engine.GET("/health", ctrl.Check)
}
