package helpers

import frameworkhelpers "github.com/leomorpho/goship/framework/web/helpers/gen"

var (
	CacheBuster       = frameworkhelpers.CacheBuster
	File              = frameworkhelpers.File
	ServiceWorkerFile = frameworkhelpers.ServiceWorkerFile
	Link              = frameworkhelpers.Link
	UnsafeHTML        = frameworkhelpers.UnsafeHTML
	ToJSON            = frameworkhelpers.ToJSON
	ToJS              = frameworkhelpers.ToJS
)

type Fn = frameworkhelpers.Fn
