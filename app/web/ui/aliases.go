package ui

import frameworkui "github.com/leomorpho/goship/framework/web/ui"

type (
	Controller      = frameworkui.Controller
	Page            = frameworkui.Page
	LayoutComponent = frameworkui.LayoutComponent
	AuthUserView    = frameworkui.AuthUserView
	Pager           = frameworkui.Pager
	FormSubmission  = frameworkui.FormSubmission
)

const (
	DefaultItemsPerPage = frameworkui.DefaultItemsPerPage
	PageQueryKey        = frameworkui.PageQueryKey
)

var (
	NewController = frameworkui.NewController
	NewPage       = frameworkui.NewPage
	NewPager      = frameworkui.NewPager
)
