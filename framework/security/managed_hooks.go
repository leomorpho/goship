package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// HeaderManagedTimestamp carries the unix timestamp used for managed hook signatures.
	HeaderManagedTimestamp = "X-GoShip-Timestamp"
	// HeaderManagedNonce carries the single-use nonce used for replay protection.
	HeaderManagedNonce = "X-GoShip-Nonce"
	// HeaderManagedSignature carries the request HMAC signature.
	HeaderManagedSignature = "X-GoShip-Signature"
)

var (
	ErrManagedSecretNotConfigured = errors.New("managed hook secret is not configured")
	ErrManagedMissingTimestamp    = errors.New("managed hook timestamp header is required")
	ErrManagedMissingNonce        = errors.New("managed hook nonce header is required")
	ErrManagedMissingSignature    = errors.New("managed hook signature header is required")
	ErrManagedInvalidTimestamp    = errors.New("managed hook timestamp is invalid")
	ErrManagedSignatureExpired    = errors.New("managed hook timestamp is outside the allowed window")
	ErrManagedSignatureMismatch   = errors.New("managed hook signature does not match")
	ErrManagedReplayDetected      = errors.New("managed hook nonce already used")
)

// ManagedHookSignatureVector captures a canonical managed-hook signing fixture.
type ManagedHookSignatureVector struct {
	Method            string `json:"method"`
	Path              string `json:"path"`
	Timestamp         int64  `json:"timestamp"`
	Nonce             string `json:"nonce"`
	Body              []byte `json:"body,omitempty"`
	ExpectedSignature string `json:"expected_signature"`
}

// ManagedHookSignatureVectors is the shared vector registry for managed-hook signing fixtures.
var ManagedHookSignatureVectors = []ManagedHookSignatureVector{
	{
		Method:            http.MethodGet,
		Path:              "/managed/status?verbose=true",
		Timestamp:         1710000000,
		Nonce:             "nonce-123",
		ExpectedSignature: "e55f9ca752736c0787742009ae01b495a47ebd98252eff077f46689e6ba5d859",
	},
}

// CronRequest captures the signed internal cron endpoint(s) request contract.
type CronRequest struct {
	Method    string
	Path      string
	Timestamp int64
	Nonce     string
	Body      []byte
}

// NonceStore records managed-hook nonce+timestamp tuples for replay protection.
type NonceStore interface {
	Consume(key string, now time.Time, ttl time.Duration) bool
}

// ManagedHookVerifier verifies signed managed hook requests with replay protection.
type ManagedHookVerifier struct {
	secret   []byte
	maxSkew  time.Duration
	nonceTTL time.Duration
	now      func() time.Time

	nonceStore NonceStore
}

// CronRequestVerifier verifies signed cron entrypoint contract requests.
type CronRequestVerifier struct {
	managed *ManagedHookVerifier
}

// NewManagedHookVerifier constructs a verifier for managed hook requests.
func NewManagedHookVerifier(secret string, maxSkew, nonceTTL time.Duration) *ManagedHookVerifier {
	if maxSkew <= 0 {
		maxSkew = 5 * time.Minute
	}
	if nonceTTL <= 0 {
		nonceTTL = maxSkew
	}

	return &ManagedHookVerifier{
		secret:     []byte(strings.TrimSpace(secret)),
		maxSkew:    maxSkew,
		nonceTTL:   nonceTTL,
		now:        time.Now,
		nonceStore: newInMemoryNonceStore(),
	}
}

// NewCronRequestVerifier constructs a verifier for signed cron entrypoint requests.
func NewCronRequestVerifier(secret string, maxSkew, nonceTTL time.Duration) *CronRequestVerifier {
	return &CronRequestVerifier{
		managed: NewManagedHookVerifier(secret, maxSkew, nonceTTL),
	}
}

// WithNonceStore overrides the replay-protection store, allowing shared/distributed implementations.
func (v *ManagedHookVerifier) WithNonceStore(store NonceStore) *ManagedHookVerifier {
	if v == nil || store == nil {
		return v
	}
	v.nonceStore = store
	return v
}

// WithNonceStore overrides the replay-protection store for cron request verification.
func (v *CronRequestVerifier) WithNonceStore(store NonceStore) *CronRequestVerifier {
	if v == nil || v.managed == nil {
		return v
	}
	v.managed.WithNonceStore(store)
	return v
}

// VerifyRequest validates signature headers, timestamp skew, and replay constraints.
func (v *ManagedHookVerifier) VerifyRequest(r *http.Request, body []byte) error {
	if v == nil || len(v.secret) == 0 {
		return ErrManagedSecretNotConfigured
	}
	if r == nil {
		return ErrManagedSignatureMismatch
	}

	timestampRaw := strings.TrimSpace(r.Header.Get(HeaderManagedTimestamp))
	if timestampRaw == "" {
		return ErrManagedMissingTimestamp
	}
	nonce := strings.TrimSpace(r.Header.Get(HeaderManagedNonce))
	if nonce == "" {
		return ErrManagedMissingNonce
	}
	signature := strings.TrimSpace(r.Header.Get(HeaderManagedSignature))
	if signature == "" {
		return ErrManagedMissingSignature
	}

	timestamp, err := strconv.ParseInt(timestampRaw, 10, 64)
	if err != nil {
		return ErrManagedInvalidTimestamp
	}

	nowFn := v.now
	if nowFn == nil {
		nowFn = time.Now
	}
	now := nowFn().UTC()
	ts := time.Unix(timestamp, 0).UTC()
	if ts.After(now.Add(v.maxSkew)) || ts.Before(now.Add(-v.maxSkew)) {
		return ErrManagedSignatureExpired
	}

	path := managedRequestPath(r)
	expected := SignManagedHookRequest(string(v.secret), r.Method, path, timestamp, nonce, body)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return ErrManagedSignatureMismatch
	}

	if !v.consumeNonce(nonce, ts, now) {
		return ErrManagedReplayDetected
	}
	return nil
}

// VerifyCronRequest validates a signed cron request using the shared managed-hook verifier contract.
func (v *CronRequestVerifier) VerifyCronRequest(r *http.Request, body []byte) error {
	if v == nil || v.managed == nil {
		return ErrManagedSecretNotConfigured
	}
	return v.managed.VerifyRequest(r, body)
}

// VerifyCronRequest validates a signed cron request through the provided verifier.
func VerifyCronRequest(v *CronRequestVerifier, r *http.Request, body []byte) error {
	if v == nil {
		return ErrManagedSecretNotConfigured
	}
	return v.VerifyCronRequest(r, body)
}

// SignManagedHookRequest signs a managed hook request payload with HMAC-SHA256.
func SignManagedHookRequest(secret, method, path string, timestamp int64, nonce string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(CanonicalManagedHookPayload(method, path, timestamp, nonce, body))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignCronRequest signs a cron entrypoint request using the shared canonical payload library.
func SignCronRequest(secret string, req CronRequest) string {
	return SignManagedHookRequest(secret, req.Method, req.Path, req.Timestamp, req.Nonce, req.Body)
}

// CanonicalManagedHookPayload returns the canonical payload bytes for shared signature vectors.
func CanonicalManagedHookPayload(method, path string, timestamp int64, nonce string, body []byte) []byte {
	return canonicalManagedRequest(method, path, timestamp, nonce, body)
}

// CanonicalManagedHookPayloadFromRequest builds the canonical payload for a live request.
func CanonicalManagedHookPayloadFromRequest(r *http.Request, body []byte) []byte {
	if r == nil {
		return CanonicalManagedHookPayload("", "", 0, "", body)
	}
	timestamp, _ := strconv.ParseInt(strings.TrimSpace(r.Header.Get(HeaderManagedTimestamp)), 10, 64)
	return CanonicalManagedHookPayload(
		r.Method,
		managedRequestPath(r),
		timestamp,
		strings.TrimSpace(r.Header.Get(HeaderManagedNonce)),
		body,
	)
}

func canonicalManagedRequest(method, path string, timestamp int64, nonce string, body []byte) []byte {
	normalizedMethod := strings.ToUpper(strings.TrimSpace(method))
	normalizedPath := strings.TrimSpace(path)
	normalizedNonce := strings.TrimSpace(nonce)

	canonical := fmt.Sprintf("%s\n%s\n%d\n%s\n", normalizedMethod, normalizedPath, timestamp, normalizedNonce)
	return append([]byte(canonical), body...)
}

func managedRequestPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	trimmedPath := strings.TrimSpace(r.URL.Path)
	if trimmedPath == "" {
		trimmedPath = "/"
	}
	if strings.TrimSpace(r.URL.RawQuery) == "" {
		return trimmedPath
	}
	return trimmedPath + "?" + strings.TrimSpace(r.URL.RawQuery)
}

func (v *ManagedHookVerifier) consumeNonce(nonce string, timestamp, now time.Time) bool {
	key := strings.TrimSpace(nonce) + ":" + strconv.FormatInt(timestamp.Unix(), 10)
	store := v.nonceStore
	if store == nil {
		store = newInMemoryNonceStore()
		v.nonceStore = store
	}
	return store.Consume(key, now, v.nonceTTL)
}

type inMemoryNonceStore struct {
	mu         sync.Mutex
	seenNonces map[string]time.Time
}

func newInMemoryNonceStore() *inMemoryNonceStore {
	return &inMemoryNonceStore{
		seenNonces: map[string]time.Time{},
	}
}

func (s *inMemoryNonceStore) Consume(key string, now time.Time, ttl time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	for seenKey, expiresAt := range s.seenNonces {
		if !expiresAt.After(now) {
			delete(s.seenNonces, seenKey)
		}
	}
	if expiry, exists := s.seenNonces[key]; exists && expiry.After(now) {
		return false
	}

	s.seenNonces[key] = now.Add(ttl)
	return true
}
