package module

import (
	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/features/account"
	"github.com/luli-tech/twilio-Boss/internal/features/account/mongodb"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	repository account.Repository
	controller *account.AccountController
}

func NewRepository(db *mongo.Database) account.Repository {
	return mongodb.NewRepository(db)
}

func New(repository account.Repository, balanceReader account.BalanceReader) *Module {
	service := account.NewService(repository, balanceReader)
	return &Module{
		repository: repository,
		controller: account.NewAccountController(service),
	}
}

func (m *Module) Repository() account.Repository {
	return m.repository
}

func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	account.RegisterPublicRoutes(group, m.controller)
}

func (m *Module) RegisterProtectedRoutes(group *gin.RouterGroup) {
	account.RegisterProtectedRoutes(group, m.controller)
}
