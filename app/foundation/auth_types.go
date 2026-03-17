package foundation

// AuthIdentity is the authenticated user shape exposed to request middleware.
type AuthIdentity struct {
	UserID                int
	UserName              string
	UserEmail             string
	HasProfile            bool
	ProfileID             int
	ProfileFullyOnboarded bool
}

// AuthUserRecord is the user lookup shape exposed to web controllers.
type AuthUserRecord struct {
	UserID     int
	Name       string
	Email      string
	Password   string
	IsVerified bool
}
