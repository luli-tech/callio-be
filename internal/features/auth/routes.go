package auth

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(group *gin.RouterGroup, ctrl *AuthController) {
	group.POST("/auth/register", ctrl.Register)
	group.POST("/auth/login", ctrl.Login)
	group.POST("/auth/refresh", ctrl.Refresh)
}

func RegisterProtectedRoutes(group *gin.RouterGroup, ctrl *AuthController) {
	group.POST("/auth/account/credentials/rotate", ctrl.RotateCredentials)
}
