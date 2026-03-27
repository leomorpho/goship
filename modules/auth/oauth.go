package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/dberrors"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/modules/authsupport"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"golang.org/x/oauth2"
	ghoauth "golang.org/x/oauth2/github"
	goauth "golang.org/x/oauth2/google"
)

var errOAuthProviderDisabled = errors.New("oauth provider disabled")
var errOAuthStateInvalid = errors.New("invalid oauth state")

type OAuthProvider interface {
	Name() string
	Config() *oauth2.Config
	FetchUser(ctx context.Context, client *http.Client) (*OAuthUser, error)
}

type OAuthUser struct {
	Provider   string
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

type OAuthLoginResult struct {
	UserID                int
	ProfileID             int
	UserEmail             string
	NewUser               bool
	ProfileFullyOnboarded bool
}

type OAuthProviderLink struct {
	Name  string
	Label string
}

type oauthProvider struct {
	name      string
	label     string
	config    oauth2.Config
	fetchUser func(ctx context.Context, client *http.Client) (*OAuthUser, error)
}

func (p oauthProvider) Name() string {
	return p.name
}

func (p oauthProvider) Config() *oauth2.Config {
	cfg := p.config
	return &cfg
}

func (p oauthProvider) FetchUser(ctx context.Context, client *http.Client) (*OAuthUser, error) {
	return p.fetchUser(ctx, client)
}

type OAuthService struct {
	db                            *sql.DB
	dbDialect                     string
	auth                          oauthAuthClient
	profileService                profilesvc.ProfileService
	subscriptionsService          *paidsubscriptions.Service
	notificationPermissionService *notifications.NotificationPermissionService
	providers                     map[string]OAuthProvider
	exchangeCode                  func(ctx context.Context, provider OAuthProvider, code string) (*oauth2.Token, error)
	httpClient                    *http.Client
	secretKey                     string
}

type oauthAuthClient interface {
	RandomToken(length int) (string, error)
	HashPassword(password string) (string, error)
	GetIdentityByUserID(ctx context.Context, userID int) (*authsupport.AuthIdentity, error)
}

func NewOAuthService(cfg *config.Config, db *sql.DB, auth oauthAuthClient, deps Deps) *OAuthService {
	service := &OAuthService{
		db:                            db,
		dbDialect:                     cfg.Adapters.DB,
		auth:                          auth,
		profileService:                deps.ProfileService,
		subscriptionsService:          deps.SubscriptionsService,
		notificationPermissionService: deps.NotificationPermissionService,
		providers:                     make(map[string]OAuthProvider),
		httpClient:                    http.DefaultClient,
		secretKey:                     oauthSecretKey(cfg),
	}
	service.exchangeCode = func(ctx context.Context, provider OAuthProvider, code string) (*oauth2.Token, error) {
		return provider.Config().Exchange(ctx, code)
	}

	for _, provider := range buildOAuthProviders(cfg) {
		service.providers[provider.Name()] = provider
	}

	return service
}

func oauthSecretKey(cfg *config.Config) string {
	key := strings.TrimSpace(cfg.App.AppEncryptionKey)
	if key != "" && key != "=" {
		return key
	}
	return cfg.App.EncryptionKey
}

func buildOAuthProviders(cfg *config.Config) []OAuthProvider {
	baseURL := strings.TrimRight(cfg.HTTP.Domain, "/")
	providers := make([]OAuthProvider, 0, 3)

	if strings.TrimSpace(cfg.OAuth.GitHub.ClientID) != "" {
		providers = append(providers, oauthProvider{
			name:  "github",
			label: "Continue with GitHub",
			config: oauth2.Config{
				ClientID:     cfg.OAuth.GitHub.ClientID,
				ClientSecret: cfg.OAuth.GitHub.ClientSecret,
				RedirectURL:  baseURL + "/auth/oauth/github/callback",
				Scopes:       []string{"read:user", "user:email"},
				Endpoint:     ghoauth.Endpoint,
			},
			fetchUser: fetchGitHubUser,
		})
	}

	if strings.TrimSpace(cfg.OAuth.Google.ClientID) != "" {
		providers = append(providers, oauthProvider{
			name:  "google",
			label: "Continue with Google",
			config: oauth2.Config{
				ClientID:     cfg.OAuth.Google.ClientID,
				ClientSecret: cfg.OAuth.Google.ClientSecret,
				RedirectURL:  baseURL + "/auth/oauth/google/callback",
				Scopes:       []string{"openid", "email", "profile"},
				Endpoint:     goauth.Endpoint,
			},
			fetchUser: fetchGoogleUser,
		})
	}

	if strings.TrimSpace(cfg.OAuth.Discord.ClientID) != "" {
		providers = append(providers, oauthProvider{
			name:  "discord",
			label: "Continue with Discord",
			config: oauth2.Config{
				ClientID:     cfg.OAuth.Discord.ClientID,
				ClientSecret: cfg.OAuth.Discord.ClientSecret,
				RedirectURL:  baseURL + "/auth/oauth/discord/callback",
				Scopes:       []string{"identify", "email"},
				Endpoint: oauth2.Endpoint{
					AuthURL:  "https://discord.com/api/oauth2/authorize",
					TokenURL: "https://discord.com/api/oauth2/token",
				},
			},
			fetchUser: fetchDiscordUser,
		})
	}

	return providers
}

func (s *OAuthService) EnabledProviders() []OAuthProviderLink {
	links := make([]OAuthProviderLink, 0, len(s.providers))
	for _, key := range []string{"github", "google", "discord"} {
		provider, ok := s.providers[key]
		if !ok {
			continue
		}
		typed, _ := provider.(oauthProvider)
		links = append(links, OAuthProviderLink{
			Name:  provider.Name(),
			Label: typed.label,
		})
	}
	return links
}

func (s *OAuthService) AuthorizationURL(providerName, state string) (string, error) {
	provider, ok := s.providers[strings.ToLower(strings.TrimSpace(providerName))]
	if !ok {
		return "", errOAuthProviderDisabled
	}
	return provider.Config().AuthCodeURL(state), nil
}

func (s *OAuthService) HandleCallback(ctx context.Context, providerName, code string) (*OAuthLoginResult, error) {
	provider, ok := s.providers[strings.ToLower(strings.TrimSpace(providerName))]
	if !ok {
		return nil, errOAuthProviderDisabled
	}
	if s.db == nil {
		return nil, authsupport.ErrAuthStoreUnavailable
	}

	token, err := s.exchangeCode(ctx, provider, code)
	if err != nil {
		return nil, err
	}
	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	if s.httpClient != nil {
		client.Timeout = s.httpClient.Timeout
	}

	oauthUser, err := provider.FetchUser(ctx, client)
	if err != nil {
		return nil, err
	}
	oauthUser.Provider = provider.Name()
	oauthUser.Email = strings.ToLower(strings.TrimSpace(oauthUser.Email))
	if oauthUser.Email == "" {
		return nil, errors.New("oauth provider did not return an email address")
	}

	encryptedToken, err := encryptOAuthToken(s.secretKey, token.AccessToken)
	if err != nil {
		return nil, err
	}

	if existingUserID, err := s.oauthUserIDByProvider(ctx, oauthUser.Provider, oauthUser.ProviderID); err == nil {
		if err := s.upsertOAuthAccount(ctx, existingUserID, oauthUser, encryptedToken); err != nil {
			return nil, err
		}
		return s.resultForUser(ctx, existingUserID, false)
	} else if !errors.Is(err, sql.ErrNoRows) && !dberrors.IsNotFound(err) {
		return nil, err
	}

	if existingUserID, err := s.userIDByEmail(ctx, oauthUser.Email); err == nil {
		if err := s.upsertOAuthAccount(ctx, existingUserID, oauthUser, encryptedToken); err != nil {
			return nil, err
		}
		return s.resultForUser(ctx, existingUserID, false)
	} else if !errors.Is(err, sql.ErrNoRows) && !dberrors.IsNotFound(err) {
		return nil, err
	}

	passwordHash, err := s.syntheticPasswordHash()
	if err != nil {
		return nil, err
	}
	registration, err := s.profileService.RegisterUserWithProfile(
		ctx,
		fallbackDisplayName(oauthUser),
		oauthUser.Email,
		passwordHash,
		domain.DefaultBirthdate,
		s.subscriptionsService,
	)
	if err != nil {
		return nil, err
	}
	if err := s.markUserVerifiedByUserID(ctx, registration.UserID); err != nil {
		return nil, err
	}
	if err := s.markProfileFullyOnboarded(ctx, registration.ProfileID); err != nil {
		return nil, err
	}
	if err := s.createDefaultNotificationPermissions(ctx, registration.ProfileID); err != nil {
		return nil, err
	}
	if err := s.upsertOAuthAccount(ctx, registration.UserID, oauthUser, encryptedToken); err != nil {
		return nil, err
	}
	return s.resultForUser(ctx, registration.UserID, true)
}

func (s *OAuthService) resultForUser(ctx context.Context, userID int, newUser bool) (*OAuthLoginResult, error) {
	identity, err := s.auth.GetIdentityByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := &OAuthLoginResult{UserID: userID, NewUser: newUser}
	if identity != nil {
		result.UserEmail = identity.UserEmail
		result.ProfileFullyOnboarded = identity.ProfileFullyOnboarded
		if identity.HasProfile {
			result.ProfileID = identity.ProfileID
		}
	}
	return result, nil
}

func (s *OAuthService) syntheticPasswordHash() (string, error) {
	password, err := s.auth.RandomToken(32)
	if err != nil {
		return "", err
	}
	return s.auth.HashPassword(password)
}

func (s *OAuthService) createDefaultNotificationPermissions(ctx context.Context, profileID int) error {
	if s.notificationPermissionService == nil {
		return nil
	}
	for _, perm := range notifications.Permissions.Members() {
		platform := notifications.PlatformEmail
		if err := s.notificationPermissionService.CreatePermission(ctx, profileID, perm, &platform); err != nil {
			return err
		}
	}
	return nil
}

func (s *OAuthService) oauthUserIDByProvider(ctx context.Context, provider, providerID string) (int, error) {
	query := `SELECT user_id FROM oauth_accounts WHERE provider = ` + placeholder(s.dbDialect, 1) + ` AND provider_id = ` + placeholder(s.dbDialect, 2)
	var userID int
	err := s.db.QueryRowContext(ctx, query, provider, providerID).Scan(&userID)
	return userID, err
}

func (s *OAuthService) userIDByEmail(ctx context.Context, email string) (int, error) {
	query := `SELECT id FROM users WHERE lower(email) = lower(` + placeholder(s.dbDialect, 1) + `) LIMIT 1`
	var userID int
	err := s.db.QueryRowContext(ctx, query, email).Scan(&userID)
	return userID, err
}

func (s *OAuthService) upsertOAuthAccount(ctx context.Context, userID int, user *OAuthUser, encryptedToken string) error {
	query := `INSERT INTO oauth_accounts (user_id, provider, provider_id, email, token)
VALUES (` + placeholder(s.dbDialect, 1) + `, ` + placeholder(s.dbDialect, 2) + `, ` + placeholder(s.dbDialect, 3) + `, ` + placeholder(s.dbDialect, 4) + `, ` + placeholder(s.dbDialect, 5) + `)
ON CONFLICT(provider, provider_id) DO UPDATE SET
	user_id = excluded.user_id,
	email = excluded.email,
	token = excluded.token`
	_, err := s.db.ExecContext(ctx, query, userID, user.Provider, user.ProviderID, user.Email, encryptedToken)
	return err
}

func (s *OAuthService) markUserVerifiedByUserID(ctx context.Context, userID int) error {
	query := `UPDATE users SET verified = true WHERE id = ` + placeholder(s.dbDialect, 1)
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}

func (s *OAuthService) markProfileFullyOnboarded(ctx context.Context, profileID int) error {
	err := s.profileService.MarkProfileFullyOnboarded(ctx, profileID)
	if err == nil || errors.Is(err, profilesvc.ErrProfileDBNotConfigured) {
		return nil
	}
	return err
}

func placeholder(dialect string, index int) string {
	if strings.Contains(strings.ToLower(strings.TrimSpace(dialect)), "post") {
		return fmt.Sprintf("$%d", index)
	}
	return "?"
}

func fallbackDisplayName(user *OAuthUser) string {
	if strings.TrimSpace(user.Name) != "" {
		return user.Name
	}
	if at := strings.Index(user.Email, "@"); at > 0 {
		return user.Email[:at]
	}
	return "OAuth User"
}

func encryptOAuthToken(secret, plaintext string) (string, error) {
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func fetchGitHubUser(ctx context.Context, client *http.Client) (*OAuthUser, error) {
	type gitHubUser struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	user := gitHubUser{}
	if err := fetchOAuthJSON(ctx, client, "https://api.github.com/user", &user); err != nil {
		return nil, err
	}
	email := user.Email
	if strings.TrimSpace(email) == "" {
		var emails []struct {
			Email    string `json:"email"`
			Primary  bool   `json:"primary"`
			Verified bool   `json:"verified"`
		}
		if err := fetchOAuthJSON(ctx, client, "https://api.github.com/user/emails", &emails); err != nil {
			return nil, err
		}
		for _, candidate := range emails {
			if candidate.Primary && candidate.Verified {
				email = candidate.Email
				break
			}
		}
		if strings.TrimSpace(email) == "" && len(emails) > 0 {
			email = emails[0].Email
		}
	}
	name := user.Name
	if strings.TrimSpace(name) == "" {
		name = user.Login
	}
	return &OAuthUser{
		ProviderID: fmt.Sprintf("%d", user.ID),
		Email:      email,
		Name:       name,
		AvatarURL:  user.AvatarURL,
	}, nil
}

func fetchGoogleUser(ctx context.Context, client *http.Client) (*OAuthUser, error) {
	var user struct {
		Sub     string `json:"sub"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := fetchOAuthJSON(ctx, client, "https://openidconnect.googleapis.com/v1/userinfo", &user); err != nil {
		return nil, err
	}
	return &OAuthUser{
		ProviderID: user.Sub,
		Email:      user.Email,
		Name:       user.Name,
		AvatarURL:  user.Picture,
	}, nil
}

func fetchDiscordUser(ctx context.Context, client *http.Client) (*OAuthUser, error) {
	var user struct {
		ID            string `json:"id"`
		Username      string `json:"username"`
		GlobalName    string `json:"global_name"`
		Email         string `json:"email"`
		Avatar        string `json:"avatar"`
		Discriminator string `json:"discriminator"`
	}
	if err := fetchOAuthJSON(ctx, client, "https://discord.com/api/users/@me", &user); err != nil {
		return nil, err
	}
	name := user.GlobalName
	if strings.TrimSpace(name) == "" {
		name = user.Username
	}
	avatarURL := ""
	if user.Avatar != "" {
		avatarURL = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", user.ID, user.Avatar)
	}
	return &OAuthUser{
		ProviderID: user.ID,
		Email:      user.Email,
		Name:       name,
		AvatarURL:  avatarURL,
	}, nil
}

func fetchOAuthJSON(ctx context.Context, client *http.Client, url string, target any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("oauth user lookup failed: %s", strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(resp.Body).Decode(target)
}
