package ui

import appui "github.com/leomorpho/goship/app/web/ui"

type (
	Controller      = appui.Controller
	Page            = appui.Page
	LayoutComponent = appui.LayoutComponent
	AuthUserView    = appui.AuthUserView
	Pager           = appui.Pager
	FormSubmission  = appui.FormSubmission
)

const (
	DefaultItemsPerPage = appui.DefaultItemsPerPage
	PageQueryKey        = appui.PageQueryKey
)

var (
	NewController = appui.NewController
	NewPage       = appui.NewPage
	NewPager      = appui.NewPager
)
