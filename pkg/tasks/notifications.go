package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/notification"
	"github.com/mikestefanello/pagoda/pkg/domain"
	"github.com/mikestefanello/pagoda/pkg/repos/notifierrepo"
	"github.com/mikestefanello/pagoda/pkg/repos/profilerepo"
	"github.com/mikestefanello/pagoda/pkg/repos/subscriptions"
	"github.com/mikestefanello/pagoda/pkg/services"
	"github.com/rs/zerolog/log"
)

var dailyNotifText = []string{
	"A good relationship is when someone accepts your past, supports your present, and texts you back quickly.",
	"Strong relationships are built on trust, honesty, and the ability to laugh at each other’s jokes.",
	"The most important thing in communication is hearing what isn’t said – and pretending you understood.",
	"A lasting relationship is about how two people communicate, laugh, and trust each other completely – and knowing when to stop talking.",
	"Communication works for those who work at it – and those who know how to bribe with chocolate.",
	"In a relationship, you need somebody who’s going to call you out, and still love you after you call them back.",
	"Communicate even when it’s uncomfortable – like when you’re talking about who left the toilet seat up.",
	"The best relationships are built on trust, honesty, and knowing when to stop talking.",
	"A great relationship starts with great communication – and ends with great snacks.",
	"Love is not about how much you say 'I love you,' but how often you can say 'I’m sorry, what did you say?'",
	"True love is about growing as a couple and not needing to grow up.",
	"The goal in a relationship is not to think alike, but to think together – while disagreeing completely.",
	"Communication is like Wi-Fi – you can’t see it, but it connects you to what you need.",
	"A strong relationship is about loving each other even when you want to kill each other.",
	"Talking about your feelings is like opening a can of worms – but it’s necessary if you want to go fishing.",
	"The biggest problem in communication is thinking you’re good at it.",
	"Speak when you’re angry, and you’ll make the best speech you’ll ever regret – so maybe text instead.",
	"A real relationship is where you can tell each other anything – including when you’ve run out of chocolate.",
	"Communication is just talking with an extra layer of patience.",
	"The best conversations are the ones that make no sense, yet make perfect sense to both of you.",
	"Arguing with your partner is just having a heated discussion with the person who knows all your weaknesses.",
	"If we both agree all the time, one of us isn't communicating enough.",
	"A relationship without communication is like a car without gas – you’re not going anywhere fast.",
	"Love is talking to your partner and realizing halfway through that you were right all along.",
	"Communication is key, but good luck finding the lock.",
	"Communication works for those who work at it.",
	"A good relationship is when someone accepts your past, supports your present, and encourages your future.",
	"The most important thing in communication is hearing what isn’t said.",
	"Communicate even when it's uncomfortable or uneasy. One of the best ways to heal is simply getting everything out.",
	"Love is not about how much you say 'I love you,' but how much you can prove that it's true through actions and communication.",
	"In relationships, communication is not about speaking first. It’s about listening first.",
	"The goal in marriage is not to think alike, but to think together.",
	"Communication to a relationship is like oxygen to life – without it, it dies.",
	"A strong relationship requires choosing to love each other, even in those moments when you struggle to like each other.",
	"In a relationship, it's not about finding the perfect person, but about creating perfect communication.",
	"The single biggest problem in communication is the illusion that it has taken place.",
	"Communication is the solvent of all problems and is the foundation for personal development.",
	"Listening is loving in action.",
	"In the end, we just need someone who gets us.",
	"A great relationship is about two things: First, appreciating the similarities, and second, respecting the differences.",
	"In a relationship, when communication starts to fade, everything else follows.",
	"Good communication is as stimulating as black coffee, and just as hard to sleep after.",
	"Arguing with your partner is simply talking with passion.",
	"Love is understanding even when you don’t understand.",
	"A relationship without communication is like a phone without a signal – you just end up playing games.",
	"A happy marriage is the union of two good forgivers.",
	"Marriage means commitment. Of course, so does insanity.",
	"Marriage is the alliance of two people, one of whom never remembers birthdays and the other who never forgets them.",
	"Marriage is a great institution for those who like institutions.",
	"Marriage is like twirling a baton, turning a handspring, or eating with chopsticks: it looks easy until you try it.",
	"The best thing to hold onto in life is each other.",
	"A successful marriage is falling in love with the same person multiple times.",
	"A good marriage is like a good recipe: it needs time, patience, and a dash of spice.",
	"Marriage is a wonderful invention: then again, so is the bicycle repair kit.",
	"Marriage is a workshop where the husband works, and the wife shops.",
	"Marriage is like vitamins: we supplement each other’s minimum daily requirements.",
	"I love being married. It's so great to find that one special person you want to annoy for the rest of your life.",
	"Marriage is not just about love; it's about teamwork, respect, and patience.",
	"A true relationship is two unperfect people refusing to give up on each other.",
	"Marriage is a wonderful institution, but who wants to live in an institution?",
	"Marriage: an endless sleepover with your favorite weirdo.",
	"A good marriage is like a good relationship with your Wi-Fi: it works best when it’s strong, secure, and uninterrupted.",
	"Behind every great man is a woman rolling her eyes.",
	"The first to apologize is the bravest. The first to forgive is the strongest. The first to forget is the happiest.",
	"Marriage lets you annoy one special person for the rest of your life.",
	"A good marriage is one where each partner secretly suspects they got the better deal.",
	"Marriage is not just spiritual communion; it is also remembering to take out the trash.",
	"You know you're in love when you can't fall asleep because reality is finally better than your dreams.",
	"Love is telling someone their hair extensions are showing.",
	"Marriage is a relationship in which one person is always right and the other is the husband.",
	"Being married means mostly shouting 'What?' from other rooms.",
	"Marriage is the only war where you sleep with the enemy.",
	"The secret of a happy marriage remains a secret.",
	"Happily ever after is not a fairy tale. It’s a choice.",
	"Marriage is like a deck of cards. In the beginning, all you need is two hearts and a diamond. By the end, you wish you had a club and a spade.",
	"Love is being stupid together.",
	"Marriage is when a man and woman become one. The trouble starts when they try to decide which one.",
	"A good marriage is like a casserole, only those responsible for it really know what goes in it.",
	"Never go to bed mad. Stay up and fight.",
	"Love is blind, but marriage is a real eye-opener.",
	"A successful marriage requires falling in love many times, always with the same person.",
	"The best way to remember your wife's birthday is to forget it once.",
	"Love is sharing your popcorn.",
	"Marriage is not about finding someone to live with; it's about finding someone you can't live without.",
	"Chérie: Because your relationship deserves more than 'Did you eat?'",
	"Find love or deepen it – with Chérie, every conversation is a chance to grow.",
	"Chérie: Turning 'We need to talk' into 'Let's answer prompts!'",
	"Connect, answer, share, chat offline – Chérie has your relationship covered from sparks to lasting bonds.",
	"Chérie: Where 'How was your day?' meets meaningful conversations.",
	"Chérie isn’t just about finding love—it’s about growing it, one question at a time.",
	"Whether dating or committed, Chérie helps you go from small talk to deep talk.",
	"Answer prompts, share only mutual answers – Chérie ensures both voices are heard.",
	"Chérie: Because guessing games belong in board games, not relationships.",
	"For dating or committed couples, Chérie turns 'What do you want to talk about?' into an easy decision.",
	"Chérie: Making ‘I didn’t know you felt that way’ a thing of the past.",
	"Chérie: Turning 'I wish you’d open up' into 'Wow, I didn’t know that!'",
	"Deepen your bond with Chérie – because every great relationship starts with great conversations.",
	"Chérie: The app that takes your relationship from 'just talking' to 'truly connecting.'",
	"Only shared answers are revealed – with Chérie, mutual understanding is guaranteed.",
	"Chérie: Where every answered prompt brings you closer.",
	"From the first spark to deep, enduring bonds – Chérie is your relationship conversation assist.",
	"Chérie: Because your relationship deserves conversations that matter.",
	"Chérie makes sure 'I never knew that about you' becomes 'Tell me more!'",
	"Find love or deepen it – Chérie ensures your relationship flourishes.",
	"Chérie: Because deep, meaningful exchanges beat small talk any day.",
	"Chérie isn’t just about digital conversations—it’s about cultivating lasting commitments.",
	"Answering prompts with Chérie means never running out of things to talk about.",
	"Chérie: Because every great relationship deserves great communication.",
	"From dating mode to committed love – Chérie supports your relationship journey.",
}

// ----------------------------------------------------------

const TypeAllDailyConvoNotifications = "notification.all_daily_conversation"

type (
	AllDailyConvoNotificationsProcessor struct {
		orm                     *ent.Client
		profileRepo             *profilerepo.ProfileRepo
		taskRunner              *services.TaskClient
		timespanInMinutes       int
		plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo
	}
)

func NewAllDailyConvoNotificationsProcessor(
	orm *ent.Client,
	profileRepo *profilerepo.ProfileRepo,
	plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo,
	taskRunner *services.TaskClient,
	timespanInMinutes int,
) *AllDailyConvoNotificationsProcessor {
	return &AllDailyConvoNotificationsProcessor{
		orm:                     orm,
		profileRepo:             profileRepo,
		plannedNotificationRepo: plannedNotificationRepo,
		taskRunner:              taskRunner,
		timespanInMinutes:       timespanInMinutes,
	}
}

func (d *AllDailyConvoNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {
	err := d.plannedNotificationRepo.CreateNotificationTimeObjects(
		ctx, domain.NotificationTypeDailyConversationReminder, domain.NotificationPermissionDailyReminder)
	if err != nil {
		return err
	}

	profileIDs, err := d.plannedNotificationRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, time.Now(), domain.NotificationTypeDailyConversationReminder, nil)
	if err != nil {
		return err
	}

	// Start tasks in batches to notify users who should received the
	// daily notification for the current hour.
	batchSize := 50

	for i := 0; i < len(profileIDs); i += batchSize {
		end := i + batchSize
		if end > len(profileIDs) {
			end = len(profileIDs)
		}
		batch := profileIDs[i:end]

		if err := d.taskRunner.
			New(TypeDailyConvoNotification).
			Payload(DailyConvoNotificationsPayload{ProfileIDs: batch}).
			Timeout(120 * time.Second).
			Retain(24 * time.Hour).
			Save(); err != nil {
			log.Error().Err(err).
				Msg("failed to start TypeDailyConvoNotification task")
		}
	}

	return nil
}

// ----------------------------------------------------------

const TypeDailyConvoNotification = "notification.subset_daily_conversation"

type (
	DailyConvoNotificationsProcessor struct {
		orm                     *ent.Client
		notifierRepo            *notifierrepo.NotifierRepo
		echoServer              *echo.Echo
		subscriptionRepo        *subscriptions.SubscriptionsRepo
		plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo
	}

	DailyConvoNotificationsPayload struct {
		ProfileIDs []int
	}
)

func NewDailyConvoNotificationsProcessor(
	notifierRepo *notifierrepo.NotifierRepo,
	e *echo.Echo,
	subscriptionRepo *subscriptions.SubscriptionsRepo,
	plannedNotificationRepo *notifierrepo.PlannedNotificationsRepo,
) *DailyConvoNotificationsProcessor {

	return &DailyConvoNotificationsProcessor{
		notifierRepo:            notifierRepo,
		echoServer:              e,
		subscriptionRepo:        subscriptionRepo,
		plannedNotificationRepo: plannedNotificationRepo,
	}
}
func (d *DailyConvoNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	var p DailyConvoNotificationsPayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		fmt.Printf("Error unmarshalling payload: %v\n", err)
		return err
	}

	wantedProfileIDs, err := d.plannedNotificationRepo.ProfileIDsCanGetPlannedNotificationNow(
		ctx, time.Now(), domain.NotificationTypeDailyConversationReminder, &p.ProfileIDs,
	)
	if err != nil {
		return err
	}

	for _, profileID := range wantedProfileIDs {

		// Generate a random index
		randomIndex := rand.Intn(len(dailyNotifText))
		// Select a random item from the list
		randomDailyNotifText := dailyNotifText[randomIndex]

		prod, _, _, err := d.subscriptionRepo.GetCurrentlyActiveProduct(ctx, profileID)
		if err != nil {
			log.Error().
				Err(err).
				Int("profileID", profileID).
				Msg("failed to get currently active plan")
			return err
		}
		var title string

		if prod == &domain.ProductTypeFree {
			title = "🌤 Today's free question!"
		} else {
			title = "🌤 Today's question!"
		}

		url := d.echoServer.Reverse("home_feed")
		err = d.notifierRepo.PublishNotification(ctx, domain.Notification{
			Type:                      domain.NotificationTypeDailyConversationReminder,
			ProfileID:                 profileID,
			Title:                     title,
			Text:                      randomDailyNotifText,
			ReadInNotificationsCenter: true,
			Link:                      url,
		}, true, true)
		if err != nil {
			log.Error().
				Err(err).
				Int("profileID", profileID).
				Str("type", domain.NotificationTypeDailyConversationReminder.Value).
				Msg("failed to send notification")
		}
	}

	return nil
}

// ----------------------------------------------------------
const TypeDeleteStaleNotifications = "notification.recycling"

type (
	DeleteStaleNotificationsProcessor struct {
		orm     *ent.Client
		numDays int
	}
)

func NewDeleteStaleNotificationsProcessor(orm *ent.Client, numDays int) *DeleteStaleNotificationsProcessor {
	return &DeleteStaleNotificationsProcessor{
		orm:     orm,
		numDays: numDays,
	}
}
func (d *DeleteStaleNotificationsProcessor) ProcessTask(
	ctx context.Context, t *asynq.Task,
) error {

	_, err := d.orm.Notification.
		Delete().
		Where(
			notification.CreatedAtLT(time.Now().Add(time.Hour * -24 * time.Duration(d.numDays))),
		).
		Exec(ctx)

	if err != nil {
		return err
	}

	// Delete all daily notifications that are older than 48h
	_, err = d.orm.Notification.
		Delete().
		Where(
			notification.CreatedAtLT(time.Now().Add(time.Hour*-48)),
			notification.TypeIn(
				notification.Type(domain.NotificationTypeDailyConversationReminder.Value),
				notification.Type(domain.NotificationTypeNewHomeFeedQA.Value),
			),
		).
		Exec(ctx)

	if err != nil {
		return err
	}

	return nil
}
