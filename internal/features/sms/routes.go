package sms

import "github.com/gin-gonic/gin"

func RegisterPublicRoutes(group *gin.RouterGroup, ctrl *SMSController) {
	group.POST("/sms/dlr", ctrl.DeliveryReport)
}

func RegisterProtectedRoutes(group *gin.RouterGroup, ctrl *SMSController) {
	group.POST("/sms/send", ctrl.SendSMS)
	group.GET("/sms/:sid", ctrl.GetSMS)
}
