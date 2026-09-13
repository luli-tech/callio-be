package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/luli-tech/twilio-Boss/internal/features/sms"
	"github.com/luli-tech/twilio-Boss/internal/shared/apperrors"
)

// MockGateway simulates an upstream SMPP/HTTP carrier (e.g., Jasmin SMS Gateway / Telnyx).
type MockGateway struct {
	SimulatedLatency time.Duration
	FailureRate      float64 // 0.0 to 1.0
}

// NewMockGateway creates a new MockGateway instance.
func NewMockGateway(latency time.Duration) *MockGateway {
	if latency <= 0 {
		latency = 50 * time.Millisecond
	}
	return &MockGateway{
		SimulatedLatency: latency,
	}
}

// Dispatch simulates network transmission of an SMS to upstream telecom carrier.
func (g *MockGateway) Dispatch(ctx context.Context, msg *sms.Message) (*sms.DispatchResult, error) {
	select {
	case <-time.After(g.SimulatedLatency):
	case <-ctx.Done():
		return nil, apperrors.ErrCarrierTimeout
	}

	// Test case: Reject invalid special test destination "+10000000000"
	if msg.To == "+10000000000" {
		return &sms.DispatchResult{
			Status:       sms.StatusFailed,
			ErrorCode:    "30008",
			ErrorMessage: "Unknown carrier error",
		}, nil
	}

	carrierID := fmt.Sprintf("CARRIER-%s", uuid.New().String()[:8])

	return &sms.DispatchResult{
		CarrierMessageID: carrierID,
		Status:           sms.StatusSent,
	}, nil
}
