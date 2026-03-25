package security

import (
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManagedHookVerifierVerifyRequest(t *testing.T) {
	useIsolatedNonceStore(t)
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
	useIsolatedNonceStore(t)
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

func TestManagedHookVerifierRejectsReplayAcrossVerifiersWhenNonceStoreIsShared(t *testing.T) {
	useIsolatedNonceStore(t)
	store := newInMemoryNonceStore()

	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)
	verifierA := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute).WithNonceStore(store)
	verifierB := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute).WithNonceStore(store)
	verifierA.now = func() time.Time { return now }
	verifierB.now = func() time.Time { return now }

	body := []byte(`{}`)
	ts := now.Unix()
	nonce := "shared-nonce"

	req1 := httptest.NewRequest("GET", "/managed/status", nil)
	sig := SignManagedHookRequest("secret", req1.Method, "/managed/status", ts, nonce, body)
	req1.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req1.Header.Set(HeaderManagedNonce, nonce)
	req1.Header.Set(HeaderManagedSignature, sig)
	require.NoError(t, verifierA.VerifyRequest(req1, body))

	req2 := httptest.NewRequest("GET", "/managed/status", nil)
	req2.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req2.Header.Set(HeaderManagedNonce, nonce)
	req2.Header.Set(HeaderManagedSignature, sig)
	err := verifierB.VerifyRequest(req2, body)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManagedReplayDetected)
}

func TestManagedHookVerifierRejectsClockSkew(t *testing.T) {
	useIsolatedNonceStore(t)
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

func TestManagedHookVerifierRejectsReplayAcrossVerifierRestartWithDurableStore(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "nonces.json")
	t.Setenv(ManagedHooksNonceStorePathEnv, storePath)

	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)
	body := []byte(`{}`)
	ts := now.Unix()
	nonce := "durable-nonce"
	sig := SignManagedHookRequest("secret", "GET", "/managed/status", ts, nonce, body)

	verifierA := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)
	verifierA.now = func() time.Time { return now }
	reqA := httptest.NewRequest("GET", "/managed/status", nil)
	reqA.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	reqA.Header.Set(HeaderManagedNonce, nonce)
	reqA.Header.Set(HeaderManagedSignature, sig)
	require.NoError(t, verifierA.VerifyRequest(reqA, body))

	verifierB := NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)
	verifierB.now = func() time.Time { return now }
	reqB := httptest.NewRequest("GET", "/managed/status", nil)
	reqB.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	reqB.Header.Set(HeaderManagedNonce, nonce)
	reqB.Header.Set(HeaderManagedSignature, sig)
	err := verifierB.VerifyRequest(reqB, body)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrManagedReplayDetected)
}

func TestFileNonceStoreConsumeConcurrentSingleWinner(t *testing.T) {
	store := &fileNonceStore{
		path: filepath.Join(t.TempDir(), "concurrency-nonces.json"),
	}
	now := time.Date(2026, time.March, 16, 1, 20, 0, 0, time.UTC)

	const workers = 24
	var accepted atomic.Int32
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			if store.Consume("same-nonce", now, 5*time.Minute) {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()

	if got := accepted.Load(); got != 1 {
		t.Fatalf("accepted consumes = %d, want 1", got)
	}
}

func TestManagedHookVerifier_AcceptsActiveAndPreviousKeyVersions(t *testing.T) {
	useIsolatedNonceStore(t)
	now := time.Date(2026, time.March, 25, 7, 0, 0, 0, time.UTC)
	body := []byte(`{"action":"restore"}`)

	verifier := NewManagedHookVerifier("active-secret", 5*time.Minute, 5*time.Minute).
		WithPreviousSecret("previous-secret").
		WithKeyVersions("v2", "v1")
	verifier.now = func() time.Time { return now }

	timestamp := now.Unix()

	reqActive := httptest.NewRequest("POST", "/managed/restore", nil)
	reqActive.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(timestamp, 10))
	reqActive.Header.Set(HeaderManagedNonce, "nonce-active")
	reqActive.Header.Set(HeaderManagedKeyVersion, "v2")
	reqActive.Header.Set(
		HeaderManagedSignature,
		SignManagedHookRequest("active-secret", reqActive.Method, "/managed/restore", timestamp, "nonce-active", body),
	)
	require.NoError(t, verifier.VerifyRequest(reqActive, body))

	reqPrevious := httptest.NewRequest("POST", "/managed/restore", nil)
	reqPrevious.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(timestamp, 10))
	reqPrevious.Header.Set(HeaderManagedNonce, "nonce-previous")
	reqPrevious.Header.Set(HeaderManagedKeyVersion, "v1")
	reqPrevious.Header.Set(
		HeaderManagedSignature,
		SignManagedHookRequest("previous-secret", reqPrevious.Method, "/managed/restore", timestamp, "nonce-previous", body),
	)
	require.NoError(t, verifier.VerifyRequest(reqPrevious, body))
}

func TestManagedHookVerifier_RotateSecretsWithoutDowntime(t *testing.T) {
	useIsolatedNonceStore(t)
	now := time.Date(2026, time.March, 25, 9, 0, 0, 0, time.UTC)
	body := []byte(`{"action":"status"}`)
	timestamp := now.Unix()

	verifier := NewManagedHookVerifier("old-secret", 5*time.Minute, 5*time.Minute).WithKeyVersions("v1", "")
	verifier.now = func() time.Time { return now }
	verifier.RotateSecrets("new-secret", "v2")

	reqOld := httptest.NewRequest("POST", "/managed/status", nil)
	reqOld.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(timestamp, 10))
	reqOld.Header.Set(HeaderManagedNonce, "nonce-old-after-rotation")
	reqOld.Header.Set(HeaderManagedKeyVersion, "v1")
	reqOld.Header.Set(
		HeaderManagedSignature,
		SignManagedHookRequest("old-secret", reqOld.Method, "/managed/status", timestamp, "nonce-old-after-rotation", body),
	)
	require.NoError(t, verifier.VerifyRequest(reqOld, body))

	reqNew := httptest.NewRequest("POST", "/managed/status", nil)
	reqNew.Header.Set(HeaderManagedTimestamp, strconv.FormatInt(timestamp, 10))
	reqNew.Header.Set(HeaderManagedNonce, "nonce-new-after-rotation")
	reqNew.Header.Set(HeaderManagedKeyVersion, "v2")
	reqNew.Header.Set(
		HeaderManagedSignature,
		SignManagedHookRequest("new-secret", reqNew.Method, "/managed/status", timestamp, "nonce-new-after-rotation", body),
	)
	require.NoError(t, verifier.VerifyRequest(reqNew, body))
}

func useIsolatedNonceStore(t *testing.T) {
	t.Helper()
	t.Setenv(ManagedHooksNonceStorePathEnv, filepath.Join(t.TempDir(), "isolated-nonces.json"))
}
