package viewmodels

type LoginOAuthData struct {
	Providers []LoginOAuthProvider
}

type LoginOAuthProvider struct {
	Name  string
	Label string
}
