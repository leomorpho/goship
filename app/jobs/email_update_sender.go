package tasks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/views/emails/gen"
	"github.com/leomorpho/goship/app/views/web/layouts/gen"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/app/web/viewmodels"
	dbqueries "github.com/leomorpho/goship/db/queries"
	"github.com/leomorpho/goship/framework/domain"
	"log/slog"
)

type UpdateEmailSender struct {
	// TODO: feels kinda weird pass container here, but refactor later.
	container  *foundation.Container
	postgresql bool
}

func NewUpdateEmailSender(container *foundation.Container) *UpdateEmailSender {
	d := strings.ToLower(strings.TrimSpace(container.Config.Adapters.DB))
	return &UpdateEmailSender{
		container:  container,
		postgresql: d == "postgres" || d == "postgresql" || d == "pgx",
	}
}

// GetAudience returns the audience who should receive update emails.
// Partner and daily updates are always given together, such that if someone has partner notifs
// turned on, if they receive an email, it will contain a daily update.
// How many days in a row a person with "partner notifs" turned on will receive an email is determined
// by the function that consumes GetAudience. The ideais to only send them an email if they have a new notif in the last n days.
// On the other hand, if someone has daily update turned on, they will receive an email EVERY day,
// which contains new questions, as well as any partner updates.
func (e *UpdateEmailSender) GetAudience(ctx context.Context) ([]int, error) {
	if e.container == nil || e.container.Database == nil {
		return nil, errors.New("database not configured")
	}

	oneDayAgo := time.Now().Add(-24 * time.Hour)

	alreadySentDaily := domain.NotificationPermissionDailyReminder.Value
	alreadySentPartner := domain.NotificationPermissionNewFriendActivity.Value

	dailyAudienceQuery, err := dbqueries.Get("select_daily_update_audience")
	if err != nil {
		return nil, err
	}
	// Get all profiles who gave permission for daily updates and did not receive a daily/partner update email in last 24h.
	dailyRows, err := e.container.Database.QueryContext(ctx, e.bind(dailyAudienceQuery), domain.NotificationPermissionDailyReminder.Value, oneDayAgo, alreadySentDaily, alreadySentPartner)
	if err != nil {
		return nil, err
	}
	defer dailyRows.Close()

	profilesWithDailyUpdates := make([]int, 0)
	for dailyRows.Next() {
		var profileID int
		if err := dailyRows.Scan(&profileID); err != nil {
			return nil, err
		}
		profilesWithDailyUpdates = append(profilesWithDailyUpdates, profileID)
	}
	if err := dailyRows.Err(); err != nil {
		return nil, err
	}

	partnerAudienceQuery, err := dbqueries.Get("select_partner_update_audience")
	if err != nil {
		return nil, err
	}
	// Get all profiles with partner updates (excluding those with daily permission), unread partner-activity notif,
	// and no daily/partner update email sent in last 24h.
	partnerRows, err := e.container.Database.QueryContext(ctx, e.bind(partnerAudienceQuery), domain.NotificationPermissionNewFriendActivity.Value, domain.NotificationPermissionDailyReminder.Value, false, domain.NotificationTypeConnectionEngagedWithQuestion.Value, oneDayAgo, alreadySentDaily, alreadySentPartner)
	if err != nil {
		return nil, err
	}
	defer partnerRows.Close()

	profilesWithOnlyPartnerUpdates := make([]int, 0)
	for partnerRows.Next() {
		var profileID int
		if err := partnerRows.Scan(&profileID); err != nil {
			return nil, err
		}
		profilesWithOnlyPartnerUpdates = append(profilesWithOnlyPartnerUpdates, profileID)
	}
	if err := partnerRows.Err(); err != nil {
		return nil, err
	}

	profileIDs := mapset.NewSet[int]()

	for _, profileID := range profilesWithDailyUpdates {
		profileIDs.Add(profileID)
	}

	for _, profileID := range profilesWithOnlyPartnerUpdates {
		profileIDs.Add(profileID)
	}

	profileIDsSlice := profileIDs.ToSlice()

	if len(profileIDsSlice) != (len(profilesWithDailyUpdates) + len(profilesWithOnlyPartnerUpdates)) {
		slog.Error("there is an error in the queries to get the daily updates as the amounts do not match",
			"len(profileIDsSlice)", len(profileIDsSlice),
			"len(profilesWithDailyUpdates)", len(profilesWithDailyUpdates),
			"len(profilesWithOnlyPartnerUpdates)", len(profilesWithOnlyPartnerUpdates),
		)
	}

	return profileIDsSlice, nil
}

func (e *UpdateEmailSender) PrepareAndSendUpdateEmailForAll(ctx context.Context) error {
	_ = ctx
	return nil
}

func (e *UpdateEmailSender) SendUpdateEmail(
	ctx context.Context, profileID int, email, selfName, partnerName string,
	questionsAnswered, questionsNotAnswered []viewmodels.QuestionInEmail,
	dailyUpdatePermissionToken, partnerUpdatePermissionToken string, numPartnerNotifications int,
) error {

	if len(questionsAnswered) > 0 && partnerUpdatePermissionToken == "" {
		return errors.New("partner update permission token missing")
	} else if len(questionsNotAnswered) > 0 && dailyUpdatePermissionToken == "" {
		return errors.New("daily update permission token missing")
	}

	title := "New questions to answer!"
	questionsAnsweredByFriendButNotSelfTitle := ""

	// TODO: if partner in committed mode has answers waiting, use their name.
	if len(questionsAnswered) > 0 {
		answerText := "answers"
		if len(questionsAnswered) == 1 {
			answerText = "answer"
		}

		questionText := "questions"
		if len(questionsAnswered) == 1 {
			questionText = "question"
		}

		if partnerName == "" {
			title = fmt.Sprintf(
				"%d new %s you can read as soon as you've answered them yourself!", len(questionsAnswered), answerText)
			questionsAnsweredByFriendButNotSelfTitle = fmt.Sprintf(
				"You can read %d new %s from your matches for the following:", len(questionsAnswered), answerText)
		} else {
			title = fmt.Sprintf(
				"%s answered %d new %s you can read as soon as you've answered them yourself!", partnerName, len(questionsAnswered), questionText)
			questionsAnsweredByFriendButNotSelfTitle = fmt.Sprintf(
				"Christa answered %d %s you have not answered:", len(questionsAnswered), questionText)

		}
	}

	// Create a new Echo instance
	ech := echo.New()

	// Create a dummy request and response writer
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Create a new Echo context
	echoCtx := ech.NewContext(req, rec)

	url := e.container.Web.Reverse(routenames.RouteNameDeleteEmailSubscriptionWithToken,
		domain.NotificationPermissionDailyReminder.Value, dailyUpdatePermissionToken)
	unsubscribeDailyUpdatesLink := fmt.Sprintf("%s%s", e.container.Config.HTTP.Domain, url)

	url = e.container.Web.Reverse(routenames.RouteNameDeleteEmailSubscriptionWithToken,
		domain.NotificationPermissionNewFriendActivity.Value, partnerUpdatePermissionToken)
	unsubscribePartnerActivityLink := fmt.Sprintf("%s%s", e.container.Config.HTTP.Domain, url)

	page := ui.NewPage(echoCtx)
	page.Layout = layouts.Main
	data := viewmodels.NewEmailUpdate()
	data.SelfName = selfName
	data.AppName = string(e.container.Config.App.Name)
	data.SupportEmail = e.container.Config.Mail.FromAddress
	data.Domain = e.container.Config.HTTP.Domain
	data.PartnerName = partnerName
	data.NumNewNotifications = numPartnerNotifications
	data.QuestionsAnsweredByFriendButNotSelfTitle = questionsAnsweredByFriendButNotSelfTitle
	data.NumQuestionsAnsweredByFriendButNotSelf = len(questionsAnswered)
	data.QuestionsAnsweredByFriendButNotSelf = questionsAnswered
	data.QuestionsNotAnsweredInSocialCircle = questionsNotAnswered
	data.UnsubscribeDailyUpdatesLink = unsubscribeDailyUpdatesLink
	data.UnsubscribePartnerActivityLink = unsubscribePartnerActivityLink
	page.Data = data

	err := e.container.Mail.
		Compose().
		To(email).
		Subject(title).
		TemplateLayout(layouts.Email).
		Component(emails.EmailUpdate(&page)).
		Send(ctx)
	if err != nil {
		return err
	}

	emailType := domain.NotificationPermissionDailyReminder
	if len(questionsAnswered) > 0 {
		emailType = domain.NotificationPermissionNewFriendActivity
	}
	now := time.Now().UTC()
	insertSentEmailQuery, lookupErr := dbqueries.Get("insert_sent_email")
	if lookupErr != nil {
		return lookupErr
	}
	_, err = e.container.Database.ExecContext(ctx, e.bind(insertSentEmailQuery), now, now, emailType.Value, profileID)

	return err
}

func (e *UpdateEmailSender) bind(query string) string {
	if !e.postgresql || strings.Count(query, "?") == 0 {
		return query
	}
	var b strings.Builder
	b.Grow(len(query) + 8)
	arg := 1
	for _, r := range query {
		if r == '?' {
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(arg))
			arg++
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
