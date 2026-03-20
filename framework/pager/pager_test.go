package pager

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestNew(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?page=3", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	p := New(ctx, 25)
	if p.Page != 3 {
		t.Fatalf("page = %d, want 3", p.Page)
	}
	if p.PerPage != 25 {
		t.Fatalf("per-page = %d, want 25", p.PerPage)
	}
}

func TestPagerMath(t *testing.T) {
	p := Pager{Page: 2, PerPage: 10, Total: 35}
	if p.Offset() != 10 {
		t.Fatalf("offset = %d, want 10", p.Offset())
	}
	if p.Limit() != 10 {
		t.Fatalf("limit = %d, want 10", p.Limit())
	}
	if !p.HasNext() {
		t.Fatalf("expected has next page")
	}
	if !p.HasPrev() {
		t.Fatalf("expected has previous page")
	}
	if p.TotalPages() != 4 {
		t.Fatalf("total pages = %d, want 4", p.TotalPages())
	}
}
