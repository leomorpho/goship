package mailer

import (
	"bytes"
	"context"
	"errors"
	stdhtml "html"
	"regexp"
	"strings"

	"github.com/a-h/templ"
)

var (
	scriptTagPattern = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	styleTagPattern  = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	htmlTagPattern   = regexp.MustCompile(`(?s)<[^>]*>`)
)

// RenderEmail renders a templ component into HTML plus a plain-text fallback.
func RenderEmail(ctx context.Context, component templ.Component) (html string, text string, err error) {
	if component == nil {
		return "", "", errors.New("email component is required")
	}

	var buf bytes.Buffer
	if err := component.Render(ctx, &buf); err != nil {
		return "", "", err
	}

	html = strings.TrimSpace(buf.String())
	text = strings.TrimSpace(stripHTML(html))
	return html, text, nil
}

func stripHTML(value string) string {
	withoutScript := scriptTagPattern.ReplaceAllString(value, " ")
	withoutStyleAndScript := styleTagPattern.ReplaceAllString(withoutScript, " ")
	withoutTags := htmlTagPattern.ReplaceAllString(withoutStyleAndScript, " ")
	decoded := stdhtml.UnescapeString(withoutTags)
	return strings.Join(strings.Fields(decoded), " ")
}
