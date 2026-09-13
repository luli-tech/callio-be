package jasmin

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/luli-tech/twilio-Boss/internal/config"
	"github.com/luli-tech/twilio-Boss/internal/features/sms"
)

func TestJasminGatewayDispatchUsesHTTPAPI(t *testing.T) {
	var gotPath string
	var gotQuery = make(map[string]string)

	gateway := NewJasminGateway(config.SMSGatewayConfig{
		BaseURL:  "http://jasmin.local:1401",
		Username: "callio_user",
		Password: "callio_secret",
		DLRURL:   "http://localhost:8080/v1/sms/dlr",
		Timeout:  2 * time.Second,
	})
	gateway.httpClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			gotPath = r.URL.Path
			for key, values := range r.URL.Query() {
				if len(values) > 0 {
					gotQuery[key] = values[0]
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`Success "carrier-123"`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	result, err := gateway.Dispatch(context.Background(), &sms.Message{
		From: "+12025550100",
		To:   "+12025550101",
		Body: "hello",
	})
	if err != nil {
		t.Fatalf("Dispatch returned error: %v", err)
	}
	if gotPath != "/send" {
		t.Fatalf("expected /send endpoint, got %q", gotPath)
	}
	if result.CarrierMessageID != "carrier-123" {
		t.Fatalf("expected parsed carrier id, got %q", result.CarrierMessageID)
	}
	if result.Status != sms.StatusSent {
		t.Fatalf("expected sent status, got %q", result.Status)
	}

	expected := map[string]string{
		"username":   "callio_user",
		"password":   "callio_secret",
		"from":       "+12025550100",
		"to":         "+12025550101",
		"content":    "hello",
		"dlr-url":    "http://localhost:8080/v1/sms/dlr",
		"dlr-level":  "3",
		"dlr-method": http.MethodPost,
	}
	for key, value := range expected {
		if gotQuery[key] != value {
			t.Fatalf("expected query %s=%q, got %q", key, value, gotQuery[key])
		}
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}
