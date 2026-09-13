package account

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(group *gin.RouterGroup, ctrl *AccountController) {
	group.POST("/accounts", ctrl.CreateAccount)
}

func RegisterProtectedRoutes(group *gin.RouterGroup, ctrl *AccountController) {
	group.GET("/accounts/me", ctrl.GetCurrentAccount)
}
