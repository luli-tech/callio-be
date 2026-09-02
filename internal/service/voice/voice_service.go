package voice

import (
	"context"
	"fmt"
	"strings"

	"github.com/luli-tech/twilio-Boss/internal/domain"
	"github.com/luli-tech/twilio-Boss/pkg/eslclient"
	"github.com/luli-tech/twilio-Boss/pkg/logger"
)

// Engine implements domain.VoiceEngine communicating with FreeSWITCH.
type Engine struct {
	eslClient  *eslclient.Client
	callRepo   domain.CallRepository
	billingSvc domain.BillingService
	logger     *logger.Logger
}

// NewEngine creates a new FreeSWITCH Voice Engine service.
func NewEngine(
	eslClient *eslclient.Client,
	callRepo domain.CallRepository,
	billingSvc domain.BillingService,
	log *logger.Logger,
) *Engine {
	if log == nil {
		log = logger.Default()
	}
	return &Engine{
		eslClient:  eslClient,
		callRepo:   callRepo,
		billingSvc: billingSvc,
		logger:     log,
	}
}

// OriginateCall sends FreeSWITCH originate command via ESL.
func (e *Engine) OriginateCall(ctx context.Context, call *domain.CallSession) (string, error) {
	// Format FreeSWITCH originate string: originate sofia/gateway/carrier/<to> &playback(...)
	dialStr := fmt.Sprintf("sofia/gateway/outbound/%s &park()", call.To)
	args := fmt.Sprintf("{origination_caller_id_number=%s,call_sid=%s}%s", call.From, call.SID, dialStr)

	res, err := e.eslClient.SendAPI(ctx, "originate", args)
	if err != nil {
		e.logger.Error("failed to originate call via ESL", "call_sid", call.SID, "error", err)
		return "", domain.ErrESLConnectionFailed
	}

	if !strings.HasPrefix(strings.TrimSpace(res), "+OK") {
		return "", fmt.Errorf("%w: %s", domain.ErrCarrierRejected, res)
	}

	return strings.TrimSpace(res), nil
}

// HangupCall issues a channel hangup command to FreeSWITCH.
func (e *Engine) HangupCall(ctx context.Context, callSID string, reason string) error {
	if reason == "" {
		reason = "NORMAL_CLEARING"
	}
	args := fmt.Sprintf("call_sid %s %s", callSID, reason)
	_, err := e.eslClient.SendAPI(ctx, "uuid_kill", args)
	return err
}

// PlayAudio streams an audio file to an active call channel.
func (e *Engine) PlayAudio(ctx context.Context, callSID string, audioURL string) error {
	args := fmt.Sprintf("%s %s", callSID, audioURL)
	_, err := e.eslClient.SendAPI(ctx, "uuid_broadcast", args)
	return err
}

