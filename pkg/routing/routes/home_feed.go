package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/pkg/controller"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/routing/routenames"
	"github.com/mikestefanello/pagoda/pkg/types"
	"github.com/mikestefanello/pagoda/templates"
	"github.com/mikestefanello/pagoda/templates/layouts"
	"github.com/mikestefanello/pagoda/templates/pages"
	"github.com/rs/zerolog/log"
)

type (
	homeFeed struct {
		ctr         controller.Controller
		profileRepo profilerepo.ProfileRepo
		pageSize    *int
	}
)

func NewHomeFeedRoute(
	ctr controller.Controller,
	profileRepo profilerepo.ProfileRepo,
	pageSize *int,
) homeFeed {

	return homeFeed{
		ctr:         ctr,
		profileRepo: profileRepo,
		pageSize:    pageSize,
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

	page := controller.NewPage(ctx)
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

	data := types.HomeFeedData{
		NextPageURL:           nextPageURL,
		SupportEmail:          c.ctr.Container.Config.App.SupportEmail,
		JustFinishedOnboarded: justFinishedOnboarded,
	}

	page.Data = data
	page.HTMX.Request.Boosted = true
	page.ShowBottomNavbar = true
	page.SelectedBottomNavbarItem = domain.BottomNavbarItemHome

	return c.ctr.RenderPage(ctx, page)
}

func (c *homeFeed) GetHomeButtons(ctx echo.Context) error {

	page := controller.NewPage(ctx)
	page.Layout = layouts.Main
	page.Component = pages.HomeFeedButtons(&page)
	page.Name = templates.PageHomeFeed

	var numWaitingOnPartner int

	numDrafts := 2

	numLiked := 4

	waitingOnYou := 2

	data := types.HomeFeedButtonsData{
		NumDrafts:           numDrafts,
		NumLikedQuestions:   numLiked,
		NumWaitingOnPartner: numWaitingOnPartner,
		NumWaitingOnYou:     waitingOnYou,
	}

	page.Data = data

	return c.ctr.RenderPage(ctx, page)
}
