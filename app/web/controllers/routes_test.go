//go:build integration

package controllers_test

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/app"
	"github.com/leomorpho/goship/app/foundation"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"github.com/leomorpho/goship/config"
	storagerepo "github.com/leomorpho/goship/framework/repos/storage"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

var (
	srv *httptest.Server
	c   *foundation.Container
)

func TestMain(m *testing.M) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("failed to resolve current test file path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
	if err := os.Chdir(repoRoot); err != nil {
		panic(err)
	}

	// Set the environment to test
	config.SwitchEnvironment(config.EnvTest)

	// Start a new container
	c = foundation.NewContainer()
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
		PubSub:                              foundation.AdaptNotificationsPubSub(c.CorePubSub),
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
		panic(err)
	}

	// Start a test HTTP server
	if err := goship.BuildRouter(c, goship.RouterModules{
		PaidSubscriptions: paidSubscriptions,
		Notifications:     notificationServices,
	}); err != nil {
		panic(err)
	}
	srv = httptest.NewServer(c.Web)

	// Run tests
	exitVal := m.Run()

	// Shutdown the container and test server
	if err := c.Shutdown(); err != nil {
		panic(err)
	}
	srv.Close()

	os.Exit(exitVal)
}

type httpRequest struct {
	route  string
	client http.Client
	body   url.Values
	t      *testing.T
}

func request(t *testing.T) *httpRequest {
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	r := httpRequest{
		t:    t,
		body: url.Values{},
		client: http.Client{
			Jar: jar,
		},
	}
	return &r
}

func (h *httpRequest) setClient(client http.Client) *httpRequest {
	h.client = client
	return h
}

func (h *httpRequest) setRoute(route string, params ...any) *httpRequest {
	h.route = srv.URL + c.Web.Reverse(route, params)
	return h
}

func (h *httpRequest) setBody(body url.Values) *httpRequest {
	h.body = body
	return h
}

func (h *httpRequest) get() *httpResponse {
	resp, err := h.client.Get(h.route)
	require.NoError(h.t, err)
	r := httpResponse{
		t:        h.t,
		Response: resp,
	}
	return &r
}

func (h *httpRequest) post() *httpResponse {
	// Make a get request to get the CSRF token
	doc := h.get().
		assertStatusCode(http.StatusOK).
		toDoc()

	// Extract the CSRF and include it in the POST request body
	csrf := doc.Find(`input[name="csrf"]`).First()
	token, exists := csrf.Attr("value")
	assert.True(h.t, exists)
	h.body["csrf"] = []string{token}

	// Make the POST requests
	resp, err := h.client.PostForm(h.route, h.body)
	require.NoError(h.t, err)
	r := httpResponse{
		t:        h.t,
		Response: resp,
	}
	return &r
}

type httpResponse struct {
	*http.Response
	t *testing.T
}

func (h *httpResponse) assertStatusCode(code int) *httpResponse {
	assert.Equal(h.t, code, h.Response.StatusCode)
	return h
}

func (h *httpResponse) assertRedirect(t *testing.T, route string, params ...any) *httpResponse {
	assert.Equal(t, c.Web.Reverse(route, params), h.Header.Get("Location"))
	return h
}

func (h *httpResponse) toDoc() *goquery.Document {
	doc, err := goquery.NewDocumentFromReader(h.Body)
	require.NoError(h.t, err)
	err = h.Body.Close()
	assert.NoError(h.t, err)
	return doc
}
