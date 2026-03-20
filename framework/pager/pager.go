package pager

import (
	"math"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

type Pager struct {
	Page    int
	PerPage int
	Total   int
}

func New(ctx echo.Context, perPage int) Pager {
	if perPage <= 0 {
		perPage = 20
	}
	p := Pager{
		Page:    1,
		PerPage: perPage,
	}
	if ctx == nil {
		return p
	}
	pageRaw := strings.TrimSpace(ctx.QueryParam("page"))
	if pageRaw == "" {
		return p
	}
	page, err := strconv.Atoi(pageRaw)
	if err != nil || page < 1 {
		return p
	}
	p.Page = page
	return p
}

func (p Pager) Offset() int {
	if p.Page < 1 {
		return 0
	}
	return (p.Page - 1) * p.Limit()
}

func (p Pager) Limit() int {
	if p.PerPage <= 0 {
		return 20
	}
	return p.PerPage
}

func (p Pager) HasNext() bool {
	return p.Page < p.TotalPages()
}

func (p Pager) HasPrev() bool {
	return p.Page > 1
}

func (p Pager) TotalPages() int {
	if p.Total <= 0 {
		return 1
	}
	return int(math.Ceil(float64(p.Total) / float64(p.Limit())))
}
