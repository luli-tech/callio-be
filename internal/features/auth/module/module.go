package module

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/features/auth"
)

type Module struct {
	service    auth.ServiceContract
	controller *auth.AuthController
}

func New(accountStore auth.AccountStore, userStore auth.UserStore, jwtConfig config.JWTConfig) *Module {
	service := auth.NewService(accountStore, userStore, jwtConfig)
	return &Module{
		service:    service,
		controller: auth.NewAuthController(service),
	}
}

func (m *Module) Service() auth.ServiceContract {
	return m.service
}

func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	auth.RegisterPublicRoutes(group, m.controller)
}

func (m *Module) RegisterProtectedRoutes(group *gin.RouterGroup) {
	auth.RegisterProtectedRoutes(group, m.controller)
}
