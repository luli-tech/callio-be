package module

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/features/user"
	"github.com/luli-tech/twilio-Boss/internal/features/user/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	controller *user.Controller
}

func NewRepository(db *mongo.Database) user.Repository {
	return mongodb.NewRepository(db)
}

func New(repository user.Repository) *Module {
	service := user.NewService(repository)
	return &Module{
		controller: user.NewController(service),
	}
}

func (m *Module) RegisterProtectedRoutes(group *gin.RouterGroup) {
	user.RegisterProtectedRoutes(group, m.controller)
}
