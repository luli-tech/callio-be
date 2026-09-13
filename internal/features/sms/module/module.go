package module

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/features/billing"
	"github.com/luli-tech/twilio-Boss/internal/features/sms"
	"github.com/luli-tech/twilio-Boss/internal/features/sms/jasmin"
	"github.com/luli-tech/twilio-Boss/internal/features/sms/mock"
	"github.com/luli-tech/twilio-Boss/internal/features/sms/mongodb"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Module struct {
	controller *sms.SMSController
	closeFn    func()
}

func NewRepository(db *mongo.Database) sms.Repository {
	return mongodb.NewRepository(db)
}

func New(repository sms.Repository, billingService billing.Service, cfg *config.Config, log *logger.Logger) *Module {
	gateway := newGateway(cfg)
	service := sms.NewService(repository, billingService, gateway, &cfg.Billing, log)
	return &Module{
		controller: sms.NewSMSController(service),
		closeFn:    service.Close,
	}
}

func (m *Module) Close() {
	if m.closeFn != nil {
		m.closeFn()
	}
}

func (m *Module) RegisterPublicRoutes(group *gin.RouterGroup) {
	sms.RegisterPublicRoutes(group, m.controller)
}

func (m *Module) RegisterProtectedRoutes(group *gin.RouterGroup) {
	sms.RegisterProtectedRoutes(group, m.controller)
}

func newGateway(cfg *config.Config) sms.Gateway {
	if strings.EqualFold(cfg.SMSGateway.Provider, "jasmin") {
		return jasmin.NewJasminGateway(cfg.SMSGateway)
	}
	return mock.NewMockGateway(50 * time.Millisecond)
}
