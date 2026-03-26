package ui

import (
	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/domain"
	frameworkpage "github.com/leomorpho/goship/framework/web/page"
	templates "github.com/leomorpho/goship/framework/web/templates"
	i18nmodule "github.com/leomorpho/goship/modules/i18n"
)

type (
	LayoutComponent func(content templ.Component, page *Page) templ.Component
	AuthUserView    struct {
		ID    int
		Name  string
		Email string
	}
)

// Page consists of all data that will be used to render a page response for a given webui.
// While it's not required for a controller to render a Page on a route, this is the common data
// object that will be passed to the templates, making it easy for all controllers to share
// functionality both on the back and frontend. The Page can be expanded to include anything else
// your app wants to support.
// Methods on this page also then become available in the templates, which can be more useful than
// the funcmap if your methods require data stored in the page, such as the appcontext.
type Page struct {
	// Base stores app-agnostic page fields/behavior owned by framework.
	frameworkpage.Base

	// Layout stores the templ component layout base function which will be used when the page is rendered.
	Layout LayoutComponent

	// Name stores the name of the page as well as the name of the template file which will be used to render
	// the content portion of the layout template.
	// This should match a template file located within the pages directory inside the templates directory.
	// The template extension should not be included in this value.
	Name templates.Page

	IsNavBarSticky bool

	// IsFullyOnboarded indicates whether the user is fully onboarded
	IsFullyOnboarded bool

	// AuthUser stores the authenticated user
	AuthUser *AuthUserView

	AuthUserProfilePicURL string

	// ActiveProduct stores the active product for the profile (limited to 1 for now)
	ActiveProduct domain.ProductType

	// Metatags stores metatag values
	Metatags struct {
		// Description stores the description metatag value
		Description string

		// Keywords stores the keywords metatag values
		Keywords []string
	}

	// Pager stores a pager which can be used to page lists of results
	Pager Pager

	// Bottom navbar is only shown if this is set. It allows flexibility for a native-like experience.
	ShowBottomNavbar         bool
	SelectedBottomNavbarItem domain.BottomNavbarItem
}

// NewPage creates and initiatizes a new Page for a given request context
func NewPage(ctx echo.Context) Page {
	base := frameworkpage.NewBase(ctx)
	p := Page{
		Base:  base,
		Pager: NewPager(ctx, DefaultItemsPerPage),
	}

	if u := ctx.Get(appcontext.AuthenticatedUserIDKey); u != nil {
		userID, ok := u.(int)
		if ok && userID > 0 {
			p.AuthUser = &AuthUserView{
				ID: userID,
			}
			if userName, ok := ctx.Get(appcontext.AuthenticatedUserNameKey).(string); ok {
				p.AuthUser.Name = userName
			}
			if userEmail, ok := ctx.Get(appcontext.AuthenticatedUserEmailKey).(string); ok {
				p.AuthUser.Email = userEmail
			}
		}
		if fullyOnboarded := ctx.Get(appcontext.ProfileFullyOnboarded); fullyOnboarded != nil {
			p.IsFullyOnboarded = fullyOnboarded.(bool)
		} else {
			p.IsFullyOnboarded = false
		}
	}
	if u := ctx.Get(appcontext.AuthenticatedUserProfilePicURL); u != nil {
		p.AuthUserProfilePicURL = u.(string)
	}

	p.ShowBottomNavbar = false

	return p
}

func (p Page) Language() string {
	if p.Context == nil || p.Context.Request() == nil {
		return "en"
	}
	lang := i18nmodule.LanguageFromContext(p.Context.Request().Context())
	if lang == "" {
		return "en"
	}
	return lang
}

func (p Page) StarterPageClass() string {
	return recipeClass(RecipePage)
}

func (p Page) StarterPanelClass() string {
	return recipeClass(RecipePanel)
}

func (p Page) StarterTitleClass() string {
	return recipeClass(RecipeTitle)
}

func (p Page) StarterTextClass() string {
	return recipeClass(RecipeText)
}

func (p Page) StarterPrimaryActionClass() string {
	return buttonClass(true)
}

func (p Page) StarterSecondaryActionClass() string {
	return buttonClass(false)
}

func (p Page) StarterKickerClass() string {
	return recipeClass(RecipeKicker)
}

func (p Page) StarterStackClass() string {
	return recipeClass(RecipeStack)
}

func (p Page) StarterMutedColorClass() string {
	return recipeClass(RecipeMutedColor)
}

func (p Page) StarterElevationClass() string {
	return recipeClass(RecipeElevationFloat)
}

func (p Page) StarterCardClass() string {
	return cardClass()
}

func (p Page) StarterNavClass() string {
	return navClass()
}

func (p Page) StarterNavItemClass(active bool) string {
	return navItemClass(active)
}

func (p Page) StarterAlertClass(variant string) string {
	return alertClass(variant)
}

func (p Page) StarterLayoutShellClass() string {
	return layoutShellClass()
}

func (p Page) StarterLayoutHeaderClass() string {
	return layoutHeaderClass()
}

func (p Page) StarterLayoutContentClass() string {
	return layoutContentClass()
}

func (p Page) StarterLayoutFooterClass() string {
	return layoutFooterClass()
}

func (p Page) StarterIslandMountClass() string {
	return islandMountClass()
}
