package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	goship "github.com/leomorpho/goship/app"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/config"
	frameworkbootstrap "github.com/leomorpho/goship/framework/bootstrap"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"
	profilesvc "github.com/leomorpho/goship/modules/profile"
)

type RequestOpt func(*requestConfig) error

type requestConfig struct {
	headers map[string]string
	cookies []*http.Cookie
}

type TestServer struct {
	Server    *httptest.Server
	Container *foundation.Container
	t         testing.TB
	client    *http.Client
}

func NewTestServer(t testing.TB) *TestServer {
	t.Helper()
	config.SwitchEnvironment(config.EnvTest)

	c := foundation.NewContainer()

	paidSubscriptions := paidsubscriptions.New(paidsubscriptions.NewSQLStore(
		c.Database,
		c.Config.Adapters.DB,
		c.Config.App.OperationalConstants.ProTrialTimespanInDays,
		c.Config.App.OperationalConstants.PaymentFailedGracePeriodInDays,
	))
	storageClient := storagerepo.NewStorageClient(c.Config, c.Database, c.Config.Adapters.DB)
	profileService := profilesvc.NewProfileServiceWithDBDeps(
		c.Database,
		c.Config.Adapters.DB,
		storageClient,
		paidSubscriptions,
		profilesvc.NewSQLNotificationCountStore(c.Database, c.Config.Adapters.DB),
	)

	var firebaseJSONAccessKeys *[]byte
	if len(c.Config.App.FirebaseJSONAccessKeys) > 0 {
		firebaseJSONAccessKeys = &c.Config.App.FirebaseJSONAccessKeys
	}
	notificationServices, err := notifications.New(notifications.RuntimeDeps{
		DB:                                  c.Database,
		DBDialect:                           c.Config.Adapters.DB,
		PubSub:                              frameworkbootstrap.AdaptNotificationsPubSub(c.CorePubSub),
		SubscriptionService:                 paidSubscriptions,
		VapidPublicKey:                      c.Config.App.VapidPublicKey,
		VapidPrivateKey:                     c.Config.App.VapidPrivateKey,
		MailFromAddress:                     c.Config.Mail.FromAddress,
		FirebaseJSONAccessKeys:              firebaseJSONAccessKeys,
		SMSRegion:                           c.Config.Phone.Region,
		SMSSenderID:                         c.Config.Phone.SenderID,
		SMSValidationCodeExpirationMinutes:  c.Config.Phone.ValidationCodeExpirationMinutes,
		GetNumNotificationsForProfileByIDFn: profileService.GetCountOfUnseenNotifications,
	})
	if err != nil {
		t.Fatalf("build notifications module: %v", err)
	}

	if err := goship.BuildRouter(c, goship.RouterModules{
		PaidSubscriptions: paidSubscriptions,
		Notifications:     notificationServices,
	}); err != nil {
		t.Fatalf("build router: %v", err)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}

	srv := httptest.NewServer(c.Web)
	t.Cleanup(func() {
		srv.Close()
		_ = c.Shutdown()
	})

	return &TestServer{
		Server:    srv,
		Container: c,
		t:         t,
		client: &http.Client{
			Jar: jar,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (s *TestServer) Get(path string, opts ...RequestOpt) *TestResponse {
	req, err := http.NewRequest(http.MethodGet, s.url(path), nil)
	if err != nil {
		s.t.Fatalf("build GET request: %v", err)
	}
	if err := applyRequestOpts(req, opts...); err != nil {
		s.t.Fatalf("apply GET request options: %v", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("run GET request: %v", err)
	}
	return &TestResponse{Response: resp, t: s.t}
}

func (s *TestServer) PostForm(path string, form url.Values, opts ...RequestOpt) *TestResponse {
	if form == nil {
		form = url.Values{}
	}
	token := extractCSRFTokenFromPath(s, path, opts...)
	if token != "" && form.Get("csrf") == "" {
		form.Set("csrf", token)
	}

	req, err := http.NewRequest(http.MethodPost, s.url(path), strings.NewReader(form.Encode()))
	if err != nil {
		s.t.Fatalf("build POST request: %v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
		query := req.URL.Query()
		query.Set("csrf", token)
		req.URL.RawQuery = query.Encode()
	}
	if err := applyRequestOpts(req, opts...); err != nil {
		s.t.Fatalf("apply POST request options: %v", err)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("run POST request: %v", err)
	}
	return &TestResponse{Response: resp, t: s.t}
}

func WithHeader(key, value string) RequestOpt {
	return func(cfg *requestConfig) error {
		if cfg == nil {
			return nil
		}
		cfg.headers[key] = value
		return nil
	}
}

func WithCookie(cookie *http.Cookie) RequestOpt {
	return func(cfg *requestConfig) error {
		if cfg == nil || cookie == nil {
			return nil
		}
		cfg.cookies = append(cfg.cookies, cookie)
		return nil
	}
}

func (s *TestServer) url(path string) string {
	if path == "" {
		return s.Server.URL
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return s.Server.URL + path
}

func applyRequestOpts(req *http.Request, opts ...RequestOpt) error {
	cfg := &requestConfig{
		headers: map[string]string{},
		cookies: []*http.Cookie{},
	}
	for _, opt := range opts {
		if opt != nil {
			if err := opt(cfg); err != nil {
				return err
			}
		}
	}
	for key, value := range cfg.headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range cfg.cookies {
		req.AddCookie(cookie)
	}
	return nil
}

type TestResponse struct {
	*http.Response
	t       testing.TB
	bodyRaw []byte
}

func (r *TestResponse) AssertStatus(code int) *TestResponse {
	r.t.Helper()
	if r.StatusCode != code {
		r.t.Errorf("expected status %d, got %d", code, r.StatusCode)
	}
	return r
}

func (r *TestResponse) AssertRedirectsTo(path string) *TestResponse {
	r.t.Helper()
	if r.StatusCode < 300 || r.StatusCode > 399 {
		r.t.Errorf("expected redirect status code, got %d", r.StatusCode)
	}
	location := r.Header.Get("Location")
	if location != path {
		r.t.Errorf("expected redirect to %q, got %q", path, location)
	}
	return r
}

func (r *TestResponse) AssertContains(text string) *TestResponse {
	r.t.Helper()
	body := r.body()
	if !strings.Contains(string(body), text) {
		r.t.Errorf("response body does not contain %q", text)
	}
	return r
}

func (r *TestResponse) AssertJSON(v any) *TestResponse {
	r.t.Helper()
	body := r.body()
	if err := jsonUnmarshal(body, v); err != nil {
		r.t.Errorf("failed to decode JSON response: %v", err)
	}
	return r
}

func (r *TestResponse) body() []byte {
	if r.bodyRaw != nil {
		return r.bodyRaw
	}
	defer r.Body.Close()
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		r.t.Fatalf("read response body: %v", err)
	}
	r.bodyRaw = payload
	return payload
}

func jsonUnmarshal(data []byte, v any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
