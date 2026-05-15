package channel

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// MetaChannel holds the shared HTTP transport, base URL, access token, app
// secret and verify token used by both WhatsApp and Instagram channels.
type MetaChannel struct {
	httpClient    *http.Client
	baseURL       string
	accessToken   string
	appSecret     string
	verifyToken   string
	channelType   string
	apiVersion    string
}

// NewMetaChannel creates the shared base for Meta-derived channels.
func NewMetaChannel(baseURL, accessToken, appSecret, verifyToken, channelType string) MetaChannel {
	return MetaChannel{
		httpClient: &http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		baseURL:     baseURL,
		accessToken: accessToken,
		appSecret:   appSecret,
		verifyToken: verifyToken,
		channelType: channelType,
		apiVersion:  "v19.0",
	}
}

// ChannelType returns the platform identifier.
func (m *MetaChannel) ChannelType() string { return m.channelType }

// VerifyWebhook checks the subscription challenge from Meta.
func (m *MetaChannel) VerifyWebhook(mode, verifyToken, challenge string) (string, error) {
	if mode != "subscribe" {
		return "", errors.New("invalid mode")
	}
	if verifyToken != m.verifyToken {
		return "", errors.New("invalid verify token")
	}
	return challenge, nil
}

// VerifySignature validates the HMAC-SHA256 signature on a webhook payload.
func (m *MetaChannel) VerifySignature(payload []byte, signature string) error {
	if m.appSecret == "" {
		return errors.New("app secret not configured")
	}

	const prefix = "sha256="
	if !strings.HasPrefix(signature, prefix) {
		return errors.New("invalid signature format")
	}

	expectedMAC, err := hex.DecodeString(strings.TrimPrefix(signature, prefix))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	mac := hmac.New(sha256.New, []byte(m.appSecret))
	mac.Write(payload)
	computedMAC := mac.Sum(nil)

	if !hmac.Equal(expectedMAC, computedMAC) {
		return errors.New("signature mismatch")
	}
	return nil
}

// apiURL builds a Meta Graph API URL for the given path segments.
func (m *MetaChannel) apiURL(parts ...string) string {
	return fmt.Sprintf("%s/%s/%s", m.baseURL, m.apiVersion, strings.Join(parts, "/"))
}
