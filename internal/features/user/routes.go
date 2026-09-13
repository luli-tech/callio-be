package user

import "github.com/gin-gonic/gin"

func RegisterProtectedRoutes(group *gin.RouterGroup, ctrl *Controller) {
	group.GET("/users", ctrl.List)
	group.POST("/users", ctrl.Create)
	group.GET("/users/me", ctrl.Me)
	group.PATCH("/users/me", ctrl.UpdateMe)
	group.PATCH("/users/:id/role", ctrl.UpdateRole)
	group.PATCH("/users/:id/status", ctrl.UpdateStatus)
	group.DELETE("/users/:id", ctrl.Delete)
}
