package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/framework/domain"
	"github.com/leomorpho/goship/framework/web/routenames"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/web/viewmodels"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"log/slog"
)

type homeFeed struct {
	ctr            ui.Controller
	profileService profilesvc.ProfileService
	pageSize       *int
}

func NewHomeFeedRoute(
	ctr ui.Controller,
	profileService profilesvc.ProfileService,
	pageSize *int,
) homeFeed {
	return homeFeed{
		ctr:            ctr,
		profileService: profileService,
		pageSize:       pageSize,
	}
}

func (c *homeFeed) Get(ctx echo.Context) error {
	justFinishedOnboardedStr := ctx.QueryParam("just_finished_onboarding")
	var justFinishedOnboarded bool
	if justFinishedOnboardedStr != "" {
		justFinishedOnboardedStr = strings.ToLower(justFinishedOnboardedStr)
		switch justFinishedOnboardedStr {
		case "true":
			justFinishedOnboarded = true
		case "false":
			justFinishedOnboarded = false
		default:
			justFinishedOnboarded = false
		}
	}

	timestampParam := ctx.QueryParam("timestamp")
	var timestamp *time.Time
	if timestampParam != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, timestampParam)
		if err != nil {
			slog.Error("invalid timestamp format", "error", err, "convo", "invalid timestamp format")
			return echo.NewHTTPError(http.StatusBadRequest, "Invalid timestamp format")
		}
		timestamp = &parsedTime
	}

	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.HomeFeed(&page)
	page.Name = templates.PageHomeFeed

	var oldestAnswerTimestamp time.Time
	if timestamp != nil {
		oldestAnswerTimestamp = *timestamp
	} else {
		oldestAnswerTimestamp = time.Now()
	}

	nextPageURL := ctx.Echo().Reverse(routenames.RouteNameHomeFeed) + "?timestamp=" + oldestAnswerTimestamp.Format(time.RFC3339Nano)

	data := viewmodels.NewHomeFeedData()
	data.NextPageURL = nextPageURL
	data.SupportEmail = c.ctr.Container.Config.App.SupportEmail
	data.JustFinishedOnboarded = justFinishedOnboarded

	page.Data = data
	page.HTMX.Request.Boosted = true
	page.ShowBottomNavbar = true
	page.SelectedBottomNavbarItem = domain.BottomNavbarItemHome

	return c.ctr.RenderPage(ctx, page)
}

func (c *homeFeed) GetHomeButtons(ctx echo.Context) error {
	page := ui.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.HomeFeedButtons(&page)
	page.Name = templates.PageHomeFeed

	var numWaitingOnPartner int
	numDrafts := 2
	numLiked := 4
	waitingOnYou := 2

	data := viewmodels.NewHomeFeedButtonsData()
	data.NumDrafts = numDrafts
	data.NumLikedQuestions = numLiked
	data.NumWaitingOnPartner = numWaitingOnPartner
	data.NumWaitingOnYou = waitingOnYou

	page.Data = data
	return c.ctr.RenderPage(ctx, page)
}
