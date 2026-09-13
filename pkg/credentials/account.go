package credentials

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GenerateAccountCredentials returns a Twilio-style account SID and auth token.
func GenerateAccountCredentials() (string, string, error) {
	accountSID := fmt.Sprintf("AC%s", strings.ReplaceAll(uuid.New().String(), "-", "")[:32])

	tokenBytes := make([]byte, 16)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", err
	}

	return accountSID, hex.EncodeToString(tokenBytes), nil
}
