package testutil

import (
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func extractCSRFTokenFromPath(s *TestServer, path string, opts ...RequestOpt) string {
	if s == nil || s.client == nil {
		return ""
	}

	req, err := http.NewRequest(http.MethodGet, s.url(path), nil)
	if err != nil {
		s.t.Fatalf("build CSRF token GET request: %v", err)
	}
	if err := applyRequestOpts(req, opts...); err != nil {
		s.t.Fatalf("apply CSRF token request options: %v", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		s.t.Fatalf("run CSRF token GET request: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		s.t.Fatalf("parse CSRF token response body: %v", err)
	}

	token, ok := doc.Find(`input[name="csrf"]`).First().Attr("value")
	if !ok {
		return ""
	}
	return strings.TrimSpace(token)
}
