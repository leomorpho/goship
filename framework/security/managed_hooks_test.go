package security

import (
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedHookVerifierVerifyRequest(t *testing.T) {
	verifier := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)

	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)
	verifier.now = func() time.Time { return now }

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/managed/backup", nil)
	ts := now.Unix()
	nonce := "nonce-1"
	sig := SignManagedHookRequest("secret", req.Method, "/managed/backup", ts, nonce, body)

	req.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(HeaderManagedNonce, nonce)
	req.Header.Set(HeaderManagedSignature, sig)

	err := verifier.VerifyRequest(req, body)
	require.NoError(t, err)
}

func TestManagedHookVerifierRejectsReplay(t *testing.T) {
	verifier := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)
	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)
	verifier.now = func() time.Time { return now }

	body := []byte(`{}`)
	ts := now.Unix()
	nonce := "shared-nonce"

	req1 := httptest.NewRequest("GET", "/managed/status", nil)
	sig := SignManagedHookRequest("secret", req1.Method, "/managed/status", ts, nonce, body)
	req1.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req1.Header.Set(HeaderManagedNonce, nonce)
	req1.Header.Set(HeaderManagedSignature, sig)
	require.NoError(t, verifier.VerifyRequest(req1, body))

	req2 := httptest.NewRequest("GET", "/managed/status", nil)
	req2.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req2.Header.Set(HeaderManagedNonce, nonce)
	req2.Header.Set(HeaderManagedSignature, sig)
	err := verifier.VerifyRequest(req2, body)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManagedReplayDetected)
}

func TestManagedHookVerifierRejectsClockSkew(t *testing.T) {
	verifier := NewManagedHookVerifier("secret", 1*time.Minute, 1*time.Minute)
	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)
	verifier.now = func() time.Time { return now }

	body := []byte(`{}`)
	ts := now.Add(-2 * time.Minute).Unix()
	nonce := "nonce"

	req := httptest.NewRequest("POST", "/managed/restore", nil)
	sig := SignManagedHookRequest("secret", req.Method, "/managed/restore", ts, nonce, body)
	req.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(HeaderManagedNonce, nonce)
	req.Header.Set(HeaderManagedSignature, sig)

	err := verifier.VerifyRequest(req, body)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManagedSignatureExpired)
}
