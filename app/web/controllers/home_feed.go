package controllers

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/views"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/views/web/pages/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	"github.com/leomorpho/goship/framework/domain"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"github.com/rs/zerolog/log"
)

type (
	homeFeed struct {
		ctr            ui.Controller
		profileService profilesvc.ProfileService
		pageSize       *int
	}
)

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
		// Convert the query parameter string to lowercase to handle case-insensitivity
		justFinishedOnboardedStr = strings.ToLower(justFinishedOnboardedStr)

		// Parse the string into a boolean value
		switch justFinishedOnboardedStr {
		case "true":
			justFinishedOnboarded = true
		case "false":
			justFinishedOnboarded = false
		default:
			// Handle invalid or unexpected values
			// You can set a default value or handle the error as needed
			justFinishedOnboarded = false // Or you might want to return an error
		}
	}

	timestampParam := ctx.QueryParam("timestamp")
	var timestamp *time.Time
	if timestampParam != "" {
		parsedTime, err := time.Parse(time.RFC3339Nano, timestampParam)
		if err != nil {
			log.Error().Str("convo", "invalid timestamp format")
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

	// NOTE: we're obviosuly not querying any home feed items with the timestamp, but feel free to create the appropriate repo method for it.
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
