//go:build integration

package controllers_test

import (
	"net/http"
	"strings"
	"testing"

	routeNames "github.com/leomorpho/goship/app/web/routenames"
	"github.com/stretchr/testify/assert"
)

func TestIslandsDemoPage_HasFrameworkCounterIslands(t *testing.T) {
	doc := request(t).
		setRoute(routeNames.RouteNameIslandsDemo).
		get().
		assertStatusCode(http.StatusOK).
		toDoc()

	assert.Equal(t, 1, doc.Find(`[data-island="VanillaCounter"]`).Length())
	assert.Equal(t, 1, doc.Find(`[data-island="ReactCounter"]`).Length())
	assert.Equal(t, 1, doc.Find(`[data-island="VueCounter"]`).Length())
	assert.Equal(t, 1, doc.Find(`[data-island="SvelteCounter"]`).Length())

	note := strings.Join(strings.Fields(doc.Find(`[data-slot="islands-regression-note"]`).Text()), " ")
	assert.Contains(t, note, "may be moved around")
	assert.Contains(t, note, "do not delete")
}
