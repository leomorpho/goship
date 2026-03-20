// TODO: this file has not yet been updated for GoShip (it's currently set up for Chérie).
import { faker } from "@faker-js/faker";
import { devices, expect, test } from "@playwright/test";
import { config, defaultConfig } from "../config.js";
import {
  acceptInvitationByLink,
  addEmojiToFirstVisibleAnswer,
  checkBottomNavbarNotificationCount,
  checkLatestNotification,
  checkNavbarNotificationCount,
  clickOnMailpitEmail,
  completeOnboardingDriverJSFlow,
  createDraftAndVerifyExists,
  logUserOut,
  publishAnswer,
  registerUser,
  publishQuizAnswers,
  registerUserWithDefaultInfo,
  removeEmojiFromFirstAnswer,
  seeSelfConvo,
  setDatingPreferences,
  verifyEmojiDoesNotExist,
  verifyEmojiExists,
} from "./helper.ts";
const environment = process.env.NODE_ENV || "development";
const { WEBSITE_URL, EMAIL_CAPTURE_PORTAL_URL } =
  config[environment] || defaultConfig;
/*
NOTE that before running these tests, you need to have Goship running.
Note that the "Cherie" should have no accent when set to PAGODA_APP_NAME.
$ export PAGODA_APP_NAME: "Cherie"
$ make watch
*/

// TODO: add tests for
// - the change experience binary toggler, it's used on the landing page, the registration page and the settings page of a logged in user.

async function runCommittedRelationshipRegistrationTest(
  browser,
  isMobile: boolean
) {
  let deviceOptions;
  if (isMobile) {
    deviceOptions = devices["iPhone 12"];
  } else {
    deviceOptions = {
      viewport: { width: 1280, height: 720 },
    };
  }

  const context1 = await browser.newContext({ ...deviceOptions });
  const context2 = await browser.newContext({ ...deviceOptions });
  const pageUser1 = await context1.newPage();
  const pageUser2 = await context2.newPage();

  let fullname1 = faker.person.fullName();
  let fullname2 = faker.person.fullName();
  let email1 = faker.internet.email();
  let email2 = faker.internet.email();
  let password = faker.internet.password({ length: 20 });

  let birthdateDateObject = new Date(
    faker.date.birthdate({ min: 18, max: 65, mode: "age" })
  );
  const formattedDateString = birthdateDateObject
    .toISOString()
    .split("T")[0]
    .replace(/\//g, "-");

  // Register User 1
  await registerUser(
    pageUser1,
    fullname1,
    email1,
    password,
    formattedDateString,
    isMobile,
    null
  );

  // Register User 2
  await registerUser(
    pageUser2,
    fullname2,
    email2,
    password,
    formattedDateString,
    isMobile,
    null
  );

  // User 2 sends an invitation to User 1 and User 1 accepts it
  await acceptInvitationByLink(pageUser2, pageUser1, isMobile);

  // User 1 accepted the invite, user 2 receives the notification
  if (isMobile) {
    await checkBottomNavbarNotificationCount(pageUser1, 0, 1000);
    await checkBottomNavbarNotificationCount(pageUser2, 1);
  } else {
    await checkNavbarNotificationCount(pageUser1, 0, 1000);
    await checkNavbarNotificationCount(pageUser2, 1);
  }
  // link it contains should lead to the other person's profile.
  await checkLatestNotification(
    pageUser2,
    "You are now using the committed relationship mode of the app with",
    isMobile
  );

  // If we fully reload the page before having clicked on the notification, the notification count
  // should still show as before.
  pageUser1.reload();
  pageUser2.reload();
  if (isMobile) {
    await checkBottomNavbarNotificationCount(pageUser1, 0, 100);
    await checkBottomNavbarNotificationCount(pageUser2, 1);
  } else {
    await checkNavbarNotificationCount(pageUser1, 0, 100);
    await checkNavbarNotificationCount(pageUser2, 1);
  }

  // Click on notification this time, which should mark it as read
  await checkLatestNotification(
    pageUser2,
    "You are now using the committed relationship mode of the app with",
    isMobile,
    true
  );
  // await expect(
  //   pageUser2.getByRole("button", { name: "Shared Answers" })
  // ).toBeVisible();

  // Notification should be consumed with an SSE update
  if (isMobile) {
    await checkBottomNavbarNotificationCount(pageUser1, 0, 100);
    await checkBottomNavbarNotificationCount(pageUser2, 0, 100);
  } else {
    await checkNavbarNotificationCount(pageUser1, 0, 100);
    await checkNavbarNotificationCount(pageUser2, 0, 100);
  }

  // Notifications from BE reload should be correct
  pageUser1.reload();
  pageUser2.reload();
  if (isMobile) {
    await checkBottomNavbarNotificationCount(pageUser1, 0);
    await checkBottomNavbarNotificationCount(pageUser2, 0);
  } else {
    await checkNavbarNotificationCount(pageUser1, 0);
    await checkNavbarNotificationCount(pageUser2, 0);
  }
}

test.describe("Committed relationship workflows", () => {
  test("Desktop view invitation handling", async ({ browser }) => {
    test.setTimeout(120000);
    await runCommittedRelationshipRegistrationTest(browser, false);
  });

  test("Mobile view invitation handling", async ({ browser }) => {
    test.setTimeout(120000);
    await runCommittedRelationshipRegistrationTest(browser, true);
  });

  test("Accept invite: not logged in", async ({ browser }) => {
    test.setTimeout(120000);

    // When a user accepts another invitation's but isn't first logged in (or registered),
    // they are redirected to the login flow. TODO: there is no smart redirect for now, but
    // it would make their experience easier if we stored the next page address in a key param
    // and navigated to it after successful login/registration.
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      null
    );

    // Register User 2
    await registerUser(
      pageUser2,
      fullname2,
      email2,
      password,
      formattedDateString,
      false,
      null
    );

    await logUserOut(pageUser2);

    // User 2 sends an invitation to User 1 and User 1 accepts it
    await acceptInvitationByLink(pageUser1, pageUser2, false, false);

    // User 2 will be redirected to login page
    await expect(
      pageUser2.getByRole("heading", { name: "Log in" })
    ).toBeVisible();

    await pageUser2.getByPlaceholder("johny@hey.com").fill(email2);
    await pageUser2.getByPlaceholder("•••••••••").fill(password);
    await pageUser2.locator("#login-button").click();

    // After loggin in, user 2 should be automatically connected to user 1
    await pageUser2.getByText("Invitation accepted").click();
  });

  test("Accept invite: not registered", async ({ browser }) => {
    // When a user accepts another invitation's but isn't first logged in (or registered),
    // they are redirected to the login flow. TODO: there is no smart redirect for now, but
    // it would make their experience easier if we stored the next page address in a key param
    // and navigated to it after successful login/registration.
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      null
    );

    // User 2 sends an invitation to User 1 and User 1 accepts it
    await acceptInvitationByLink(pageUser1, pageUser2, false, false);

    // User 2 will be redirected to login page
    await expect(
      pageUser2.getByRole("heading", { name: "Log in" })
    ).toBeVisible();

    // User 2 will want to register
    await pageUser2.getByText("Create an account").click();
    await pageUser2.getByText("✅💋 I'm in a committed").click();
    await pageUser2.getByPlaceholder("JohnWatts123").fill(fullname2);
    await pageUser2.getByPlaceholder("steamyjohn@diesel.com").fill(email2);
    await pageUser2.getByLabel("Password", { exact: true }).fill(password);
    await pageUser2
      .getByLabel("Birthdate (you need to be 18")
      .fill(formattedDateString);
    await pageUser2
      .getByRole("button", { name: "Register", exact: true })
      .click();

    await pageUser2.getByText("SuccessYour account has been").click();

    await pageUser2.getByText("InfoAn email was sent to you").click();

    // After registration in, user 2 should be automatically connected to user 1
    await pageUser2.getByText("Invitation accepted").click();
  });

  test("Publish text answer", async ({ browser }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    await registerUserWithDefaultInfo(pageUser1);
    await registerUserWithDefaultInfo(pageUser2);

    await acceptInvitationByLink(pageUser2, pageUser1);
    // Consume notification from invitation
    await checkLatestNotification(
      pageUser2,
      "You are now using the committed relationship mode of the app with",
      false,
      true
    );

    await publishAnswer(pageUser1, 1);
    await expect(
      pageUser1.getByText(
        "Successfully published your answer. Your significant other will be able to read it once they've also answered the question."
      )
    ).toBeVisible();
    await pageUser1.getByLabel("Close").click();

    // user 2 will receive a notification about user 1 engaging with a question.
    await checkNavbarNotificationCount(pageUser2, 1);
    // Have user 1 navigate to the new shared answer through their notification
    await checkLatestNotification(
      pageUser2,
      "Your partner answered a new question. Answer it too to see their thoughts on it!",
      false,
      true,
      "Answer"
    );

    await publishAnswer(pageUser2, 1);
    // Check notification
    await expect(
      pageUser2.getByText(
        "Successfully published your answer and notified your significant other."
      )
    ).toBeVisible();
    await pageUser2.getByLabel("Close").click();

    // Check answer is visible by looking for the "shared answer" tag at top of card
    await expect(pageUser2.getByText("Shared answer")).toBeVisible();
    await pageUser2.getByRole("button", { name: "You" }).click();

    await pageUser2.locator(".delete-draft").click();
    await pageUser2.getByRole("button", { name: "OK" }).click();

    await checkNavbarNotificationCount(pageUser1, 1);
    await checkNavbarNotificationCount(pageUser2, 0);

    // Have user 1 navigate to the new shared answer through their notification
    await checkLatestNotification(
      pageUser1,
      "answered the same question as you and you can read their answer!",
      false,
      true,
      "See answer"
    );
    // Check answer is visible by looking for the "shared answer" tag at top of card
    await expect(pageUser1.getByText("Shared answer")).toBeVisible();
    await pageUser1.getByRole("button", { name: "You" }).click();

    await pageUser1.locator(".delete-draft").click();
    // await page.locator('.delete-draft > .icon').first().click();
    await pageUser1.getByRole("button", { name: "OK" }).click();

    const emojiLeftByUser1 = "🤣";
    const emojiLeftByUser2 = "🙂";

    // Test adding/removing an emoji
    await addEmojiToFirstVisibleAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiExists(pageUser1, emojiLeftByUser1);
    await removeEmojiFromFirstAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiDoesNotExist(pageUser1, emojiLeftByUser1, 3000);

    // Ok, just add an emoji for each user to the other's answer
    // NOTE: emoji notifications are only sent every 5 min, so removing the emoji won't remove the previously created notif, and adding a new one won't create another notif.
    await addEmojiToFirstVisibleAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiExists(pageUser1, emojiLeftByUser1);

    await addEmojiToFirstVisibleAnswer(pageUser2, emojiLeftByUser2);
    await verifyEmojiExists(pageUser2, emojiLeftByUser2);

    // Both users should have received notifications for these emojis
    await checkNavbarNotificationCount(pageUser1, 1);
    await checkNavbarNotificationCount(pageUser2, 1);

    // Have user 1 navigate to the new shared answer through their notification
    await checkLatestNotification(
      pageUser1,
      "reacted to one of your answers!",
      false,
      true,
      "See reaction"
    );
    await expect(pageUser1.getByText(emojiLeftByUser2)).toBeVisible();
    await verifyEmojiExists(pageUser1, emojiLeftByUser2);

    // NOTE: emoji notifications are only sent every 5 min, so removing the emoji won't remove the previously created notif, and adding a new one won't create another notif.
    // await checkLatestNotification(
    //   pageUser2,
    //   "reacted to one of your answers!",
    //   false,
    //   true,
    //   "See reaction"
    // );
    // await expect(pageUser2.getByText(emojiLeftByUser1)).toBeVisible();
    // await verifyEmojiExists(pageUser2, emojiLeftByUser1);

    console.log("done!");
  });

  test("Publish quiz answers", async ({ browser }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    await registerUserWithDefaultInfo(pageUser1);
    await registerUserWithDefaultInfo(pageUser2);

    await acceptInvitationByLink(pageUser2, pageUser1);
    // Consume notification from invitation
    await checkLatestNotification(
      pageUser2,
      "You are now using the committed relationship mode of the app with",
      false,
      true
    );

    const questionID = await publishQuizAnswers(pageUser1);
    await expect(
      pageUser1.getByText(
        "Successfully published your quiz. Your significant other will be able to read it once they've completed it too."
      )
    ).toBeVisible();

    // user 2 will receive a notification about user 1 engaging with a question.
    await checkNavbarNotificationCount(pageUser2, 1);
    // Have user 1 navigate to the new shared answer through their notification

    await checkLatestNotification(
      pageUser2,
      "Your partner answered a new quiz. Answer it too to see their answers!",
      false,
      true,
      "Answer"
    );

    await publishQuizAnswers(pageUser2, questionID);

    // Check notification
    await expect(
      pageUser2.getByText(
        "Successfully published your quiz and notified your significant other."
      )
    ).toBeVisible();

    // Check answer is visible by looking for the "shared answer" tag at top of card
    await expect(pageUser2.getByText("Shared answer")).toBeVisible();

    await pageUser2.locator(".delete-draft").click();
    await pageUser2.getByRole("button", { name: "OK" }).click();

    await checkNavbarNotificationCount(pageUser1, 1);
    await checkNavbarNotificationCount(pageUser2, 0);

    // Have user 1 navigate to the new shared answer through their notification
    await checkLatestNotification(
      pageUser1,
      "completed the same quiz as you and you can see their answers!",
      false,
      true,
      "See answer"
    );
    // Check answer is visible by looking for the "shared answer" tag at top of card
    await expect(pageUser1.getByText("Shared answer")).toBeVisible();

    await pageUser1.locator(".delete-draft").click();
    await pageUser1.getByRole("button", { name: "OK" }).click();

    const emojiLeftByUser1 = "🤣";
    const emojiLeftByUser2 = "🙂";

    // Test adding/removing an emoji
    await addEmojiToFirstVisibleAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiExists(pageUser1, emojiLeftByUser1);

    await removeEmojiFromFirstAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiDoesNotExist(pageUser1, emojiLeftByUser1, 3000);

    // Ok, just add an emoji for each user to the other's answer
    // NOTE: emoji notifications are only sent every 5 min, so removing the emoji won't remove the previously created notif, and adding a new one won't create another notif.
    await addEmojiToFirstVisibleAnswer(pageUser1, emojiLeftByUser1);
    await verifyEmojiExists(pageUser1, emojiLeftByUser1);

    await addEmojiToFirstVisibleAnswer(pageUser2, emojiLeftByUser2);
    await verifyEmojiExists(pageUser2, emojiLeftByUser2);

    // Both users should have received notifications for these emojis
    await checkNavbarNotificationCount(pageUser1, 1);
    await checkNavbarNotificationCount(pageUser2, 1);

    // Have user 1 navigate to the new shared answer through their notification
    await checkLatestNotification(
      pageUser1,
      "reacted to one of your answers!",
      false,
      true,
      "See reaction"
    );

    await verifyEmojiExists(pageUser1, emojiLeftByUser2);

    // NOTE: emoji notifications are only sent every 5 min, so removing the emoji won't remove the previously created notif, and adding a new one won't create another notif.
    // await checkLatestNotification(
    //   pageUser2,
    //   "reacted to one of your answers!",
    //   false,
    //   true,
    //   "See reaction"
    // );
    // await expect(pageUser2.getByText(emojiLeftByUser1)).toBeVisible();
    // await verifyEmojiExists(pageUser2, emojiLeftByUser1);
  });

  test("Profile views", async ({ browser }) => {
    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    await registerUserWithDefaultInfo(pageUser1);
    await registerUserWithDefaultInfo(pageUser2);

    // Connect them
    await acceptInvitationByLink(pageUser2, pageUser1);

    await pageUser1.locator("#navbar-profile-menu").first().click();
    await pageUser1.getByRole("list").getByText("Profile").click();

    await pageUser1.getByRole("button", { name: "Upload photo" }).click();
    await pageUser1.getByText("Drop files here").click();
    await pageUser1.getByRole("button", { name: "Select a file→" }).click();
    await pageUser1
      .getByRole("button", { name: "Select a file→" })
      .press("Escape");
    await pageUser1.getByRole("button", { name: "Upload photo" }).click();
    await pageUser1
      .getByRole("button", { name: "Select a file→" })
      .press("Escape");
  });

  // TODO: verify profile view of partner (from current user POV) faced a nil ptr deref there before TODO TODO TODO
  test("Self convo view", async ({ browser }) => {
    const context1 = await browser.newContext();
    const pageUser1 = await context1.newPage();

    await registerUserWithDefaultInfo(pageUser1);

    await publishAnswer(pageUser1, 1);
    await expect(
      pageUser1.getByText(
        "Successfully published your answer. Your significant other will be able to read it once they've also answered the question."
      )
    ).toBeVisible();
    await pageUser1.getByLabel("Close").click();

    await pageUser1.getByRole("link", { name: "Logo Goship" }).click();

    await pageUser1.getByRole("button", { name: "All Conversations" }).click();
    await pageUser1.locator("a").filter({ hasText: "You" }).click();
  });

  test("Waiting on partner view", async ({ browser }) => {
    const context1 = await browser.newContext();
    const pageUser1 = await context1.newPage();

    await registerUserWithDefaultInfo(pageUser1);

    await publishAnswer(pageUser1, 1);
    await expect(
      pageUser1.getByText(
        "Successfully published your answer. Your significant other will be able to read it once they've also answered the question."
      )
    ).toBeVisible();
    await pageUser1.getByLabel("Close").click();

    await pageUser1.getByRole("link", { name: "Logo Goship" }).click();

    await pageUser1.getByRole("button", { name: "Waiting on partner" }).click();
    await expect(
      pageUser1.locator('[id*="ask-partner-to-answer-question"]')
    ).toBeVisible();
    await pageUser1
      .locator('[id*="ask-partner-to-answer-question"]')
      .getByText("Ask partner to answer")
      .click();

    await expect(
      pageUser1.locator(".upvote-outline-button").first()
    ).toBeVisible();
    await expect(
      pageUser1.locator(".downvote-outline-button").first()
    ).toBeVisible();
  });

  test("Waiting on you view", async ({ browser }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    await registerUserWithDefaultInfo(pageUser1);
    await registerUserWithDefaultInfo(pageUser2);

    await acceptInvitationByLink(pageUser2, pageUser1);
    // Consume notification from invitation
    await checkLatestNotification(
      pageUser2,
      "You are now using the committed relationship mode of the app with",
      false,
      true
    );
    // await expect(
    //   pageUser2.getByRole("button", { name: "Shared Answers" })
    // ).toBeVisible();

    await publishAnswer(pageUser1, 1);
    await expect(
      pageUser1.getByText(
        "Successfully published your answer. Your significant other will be able to read it once they've also answered the question."
      )
    ).toBeVisible();
    await pageUser1.getByLabel("Close").click();

    await pageUser2.getByRole("link", { name: "Logo Goship" }).click();
    await pageUser2.getByRole("button", { name: "Waiting on You" }).click();
    await expect(pageUser2.locator(".textQuestion").nth(1)).toBeVisible();
  });

  test("Cannot publish right now", async ({ browser }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext();
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    await registerUserWithDefaultInfo(pageUser1);
    await registerUserWithDefaultInfo(pageUser2);

    await acceptInvitationByLink(pageUser2, pageUser1);
    // Consume notification from invitation
    await checkLatestNotification(
      pageUser2,
      "You are now using the committed relationship mode of the app with",
      false,
      true
    );

    for (let i = 1; i <= 5; i++) {
      await publishAnswer(pageUser1, i);
      await expect(
        pageUser1.getByText(
          "Successfully published your answer. Your significant other will be able to read it once they've also answered the question."
        )
      ).toBeVisible();
      await pageUser1.getByLabel("Close").click();
    }

    await expect(
      pageUser1.getByRole("button", { name: "Waiting on Partner 5/5" })
    ).toBeVisible();
    await expect(
      pageUser1.getByText(
        "You have 5 questions waiting for your partner to answer. Until they respond to at least one, you won’t be able to publish more. But feel free to keep writing drafts in the meantime 😉. You can also use the app's feature to give them a nudge to answer one of your questions!"
      )
    ).toBeVisible();

    await pageUser2.getByRole("link", { name: "Logo Goship" }).click();
    await pageUser2.getByRole("button", { name: "Waiting on You" }).click();
    await expect(
      pageUser2.getByRole("button", { name: "Waiting on You 5/5" })
    ).toBeVisible();

    await expect(
      pageUser2.getByText(
        "You have 5 unanswered questions from your partner. You'll need to respond to some of these before you can publish answers to any new ones. But feel free to keep drafting new answers in the meantime 😉"
      )
    ).toBeVisible();

    await pageUser1.getByRole("link", { name: "Logo Goship" }).click();

    pageUser1.locator(".textQuestion").first().click();
    await expect(
      pageUser1.getByText(
        "You have 5 questions waiting for your partner to answer. Until they respond to at least one, you won’t be able to publish more. But feel free to keep writing drafts in the meantime 😉. You can also use the app's feature to give them a nudge to answer one of your questions!"
      )
    ).toBeVisible();

    // Check for user 2
    // --- text question
    await pageUser2.getByRole("link", { name: "Logo Goship" }).click();

    pageUser2.locator(".textQuestion").nth(6).click();
    await expect(
      pageUser2.getByText(
        "You have 5 unanswered questions from your partner. You'll need to respond to some of these before you can publish answers to any new ones. But feel free to keep drafting new answers in the meantime 😉"
      )
    ).toBeVisible();
    // const publishButton2 = pageUser2.getByRole("button", { name: "Publish" });
    // console.log(publishButton2)
    // await expect(publishButton2).toBeDisabled();

    // --- quiz question
    await pageUser2.getByRole("link", { name: "Logo Goship" }).click();
    await pageUser2.locator(".quizQuestion").first().click();
    await expect(
      pageUser2.getByText(
        "You have 5 unanswered questions from your partner. You'll need to respond to some of these before you can publish answers to any new ones. But feel free to keep drafting new answers in the meantime 😉"
      )
    ).toBeVisible();
    // while (
    //   await publishButton2.getByRole("button", { name: "next" }).isVisible()
    // ) {
    //   await publishButton2.getByRole("button", { name: "next" }).click();
    // }
    // const submitButton2 = pageUser2.getByRole("button", { name: "Submit" });
    // await expect(submitButton2).toBeDisabled();

    // Check for user 1
    await pageUser1.getByRole("link", { name: "Logo Goship" }).click();
    pageUser1.locator(".textQuestion").nth(6).click();
    await expect(
      pageUser1.getByText(
        "You have 5 questions waiting for your partner to answer. Until they respond to at least one, you won’t be able to publish more. But feel free to keep writing drafts in the meantime 😉. You can also use the app's feature to give them a nudge to answer one of your questions!"
      )
    ).toBeVisible();
    await pageUser1.getByRole("link", { name: "Logo Goship" }).click();
    await pageUser1.locator(".quizQuestion").first().click();
    await expect(
      pageUser1.getByText(
        "You have 5 questions waiting for your partner to answer. Until they respond to at least one, you won’t be able to publish more. But feel free to keep writing drafts in the meantime 😉. You can also use the app's feature to give them a nudge to answer one of your questions!"
      )
    ).toBeVisible();
  });
});

test.describe("Dating workflows", () => {
  test("User registration", async ({ browser }) => {
    const context1 = await browser.newContext();
    const page = await context1.newPage();

    let fullname1 = faker.person.fullName();
    let email1 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      page,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Profile").click();
    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Settings").click();

    // Check notification permissions are loading
    await expect(
      page.locator("#notification-permissions-toggles")
    ).toContainText("Daily conversation");
    await expect(
      page
        .locator("#notification-permissions-toggles")
        .getByRole("button")
        .first()
    ).toBeVisible();
    await expect(page.getByText("Partner activity")).toBeVisible();
    await expect(
      page
        .locator("#notification-permissions-toggles")
        .getByRole("button")
        .nth(2)
    ).toBeVisible();

    // TODO: below is commented out because meeting people on our platform is disabled for now. Once enabled, uncomment.
    // // Check the multiselect is loading for preferred genders
    // await expect(
    //   page.getByLabel("selected options").getByRole("textbox")
    // ).toBeVisible();

    // // Check dropdowns are loading
    // await expect(page.getByLabel("Select minimum age")).toBeVisible();
    // await expect(page.getByLabel("Select maximum age")).toBeVisible();

    // // Check map is loading
    // await expect(page.getByLabel("Map", { exact: true })).toBeVisible();
    // await expect(page.getByLabel("Zoom in")).toBeVisible();
    // await expect(
    //   page.getByLabel("Map marker").locator("path").first()
    // ).toBeVisible();
  });

  test("Profile views", async ({ browser }) => {
    const context1 = await browser.newContext();
    const pageUser1 = await context1.newPage();

    let fullname1 = faker.person.fullName();
    let email1 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    await pageUser1.locator("#navbar-profile-menu").first().click();
    await pageUser1.getByRole("list").getByText("Profile").click();
    await pageUser1.locator("#navbar-profile-menu").first().click();
    await expect(
      pageUser1.getByRole("list").getByText("Settings")
    ).toBeVisible();
    await pageUser1.getByRole("button", { name: "Upload photo" }).click();
    await pageUser1.getByText("Drop files here").click();
    await pageUser1.getByRole("button", { name: "Select a file→" }).click();
    await pageUser1
      .getByRole("button", { name: "Select a file→" })
      .press("Escape");
    await pageUser1.getByRole("button", { name: "Upload photo" }).click();
    await pageUser1
      .getByRole("button", { name: "Select a file→" })
      .press("Escape");
  });

  test("Direct invitations", async ({ browser }) => {
    test.setTimeout(120000);
    const context1 = await browser.newContext({
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const context2 = await browser.newContext();
    const context3 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();
    const pageUser3 = await context3.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let fullname3 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let email3 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    // Register User 2
    await registerUser(
      pageUser2,
      fullname2,
      email2,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    // Register User 3
    await registerUser(
      pageUser3,
      fullname3,
      email3,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    ////////////////////////////
    // Connect users 1 and 2
    ////////////////////////////
    await pageUser1
      .getByRole("button", { name: "Invite someone directly" })
      .click();

    // Evaluate the clipboard content within a user gesture context
    let clipboardContent = await pageUser1.evaluate(async () => {
      // Ensure the action is performed after a user gesture
      await new Promise((resolve) => setTimeout(resolve, 100)); // Small delay to simulate real user interaction
      return navigator.clipboard.readText();
    });

    const urlRegex = /\bhttps?:\/\/\S+/g;
    expect(clipboardContent).not.toBeNull();

    if (!clipboardContent) throw new Error("clipboard empty!");

    let extractedUrl = clipboardContent.match(urlRegex)[0];

    await pageUser2.goto(extractedUrl);
    await pageUser2.getByText("Invitation accepted").click();

    ////////////////////////////
    // Connect users 1 and 3
    ////////////////////////////
    await pageUser1
      .getByRole("button", { name: "Invite someone directly" })
      .click();

    // Evaluate the clipboard content within a user gesture context
    clipboardContent = await pageUser1.evaluate(async () => {
      // Ensure the action is performed after a user gesture
      await new Promise((resolve) => setTimeout(resolve, 100)); // Small delay to simulate real user interaction
      return navigator.clipboard.readText();
    });

    expect(clipboardContent).not.toBeNull();

    if (!clipboardContent) throw new Error("clipboard empty!");

    extractedUrl = clipboardContent.match(urlRegex)[0];

    await pageUser3.goto(extractedUrl);
    await pageUser3.getByText("Invitation accepted").click();

    ////////////////////////////
    // Check that user 1 is connected to user 2 and 3
    ////////////////////////////
    await pageUser1.getByRole("button", { name: "All Conversations" }).click();

    await expect(
      pageUser1.locator("a").filter({ hasText: fullname2 })
    ).toBeVisible();

    await expect(
      pageUser1.locator("a").filter({ hasText: fullname3 })
    ).toBeVisible();
  });

  test("Can always publish right now", async ({ browser }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext({
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const context2 = await browser.newContext();
    const context3 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    // Register User 2
    await registerUser(
      pageUser2,
      fullname2,
      email2,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    for (let i = 1; i <= 8; i++) {
      await publishAnswer(pageUser1, i);
      await expect(
        pageUser1.getByText(
          "Successfully published your answer. "
        )
      ).toBeVisible();
      await pageUser1.getByLabel("Close").click();
    }

    for (let i = 1; i <= 8; i++) {
      await publishAnswer(pageUser2, i);
      await expect(
        pageUser2.getByText(
          "Successfully published your answer. "
        )
      ).toBeVisible();
      await pageUser2.getByLabel("Close").click();
    }

    await expect(
      pageUser1.getByRole("button", { name: "Waiting on You" })
    ).toBeVisible();

    await expect(
      pageUser2.getByRole("button", { name: "Waiting on You" })
    ).toBeVisible();
  });
});

test.describe("Shared workflows", () => {
  test("Test committed/dating selector", async ({ page }) => {
    await page.goto(`${WEBSITE_URL}/`);
    await page.getByRole("link", { name: "💋 I'm in a committed" }).click();
    await expect(page.getByRole("heading", { name: "Register" })).toBeVisible();
    await page.getByRole("link", { name: "🌹 I am looking to date" }).click();
    await expect(page.getByText("We're launching swipes")).toBeVisible();
    await page.getByRole("heading", { name: "Register" }).click();
    await page.getByRole("link", { name: "Logo Goship" }).click();
    await page.getByRole("link", { name: "🌹 I am looking to date" }).click();
    await expect(page.getByRole("heading", { name: "Register" })).toBeVisible();
  });

  test("CRUD draft", async ({ page }) => {
    await registerUserWithDefaultInfo(page);
    const draftText = await createDraftAndVerifyExists(page, 1);

    // Use a locator to find a container that includes the draft text and contains a "downvote-question" button
    await page.locator(".delete-draft").click();
    await page.getByRole("button", { name: "OK" }).click();

    // Wait for potential asynchronous operations that might delay the removal of the draft
    await page.waitForTimeout(1000);

    // Check if the element with the draft text still exists
    const draftLocator = page.locator(
      `.temporalized-item-container:has-text("${draftText}")`
    );

    await expect(draftLocator).toHaveCount(0, { timeout: 5000 });

    // Optionally, log a message if the test passes
    console.log("Draft has been successfully deleted.");

    // page.reload();
    await page
      .getByRole("button", { name: "Drafts" })
      .waitFor({ state: "visible" });
    await page.getByRole("button", { name: "Drafts" }).click();

    await expect(
      page.getByText("You don't have any drafts yet 🦗🦗🦗")
    ).toBeVisible();
  });

  test("See self-convo", async ({ page }) => {
    await registerUserWithDefaultInfo(page);
    await seeSelfConvo(page);
  });

  test("Switch experience types", async ({ page }) => {
    test.setTimeout(120000);

    await registerUserWithDefaultInfo(page);

    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Settings").click();

    // Move to dating mode
    await page.getByText("✅🌹 I am looking to date").click();
    await page.getByRole("button", { name: "Cancel" }).click();
    await page.getByText("✅🌹 I am looking to date").click();
    await page.getByRole("button", { name: "OK" }).click();

    await page.getByLabel("Finish onboarding").click();
    await completeOnboardingDriverJSFlow(page, false);
    await expect(
      page.getByRole("navigation").locator("a").filter({ hasText: "Meet" })
    ).toBeVisible();

    // Move back to committed mode...though it won't switch until the other person
    // accepts the invite.
    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Settings").click();

    await page.getByText("✅💋 I'm in a committed").click();
    await page.getByRole("button", { name: "OK" }).click();
    await expect(
      page.getByRole("heading", { name: "👥 Change to Committed" })
    ).toBeVisible();
    await page
      .getByRole("heading", {
        name: "🔗 Almost connected with your partner! 💙🎉🤗",
      })
      .click();

    await page.getByRole("button", { name: "Copy to Clipboard" }).click();

    // TODO: would be nice to have matches in dating mode and check that the match shows in the dropdown

    // TODO extend this to check that the page is scrollable after switching mode, and that the menus are as expected,
    // namely onboarding mode when going to dating and fully onboarded mode when going to committed.
  });

  // We don't want a committed profile to be able to invite a dating profile as the committed profile
  // could be in that mode by mistake, and could cause the dating profile to lose all of their matches.
  test("Committed profile invites dating profile to connect", async ({
    browser,
  }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext({
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const context2 = await browser.newContext();
    const pageCommitted = await context1.newPage();
    const pageDater = await context2.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageCommitted,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      null
    );

    // Register User 2
    await registerUser(
      pageDater,
      fullname2,
      email2,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    await pageCommitted
      .getByRole("button", { name: "Copy to Clipboard" })
      .click();

    // Evaluate the clipboard content within a user gesture context
    const clipboardContent = await pageCommitted.evaluate(async () => {
      // Ensure the action is performed after a user gesture
      await new Promise((resolve) => setTimeout(resolve, 100)); // Small delay to simulate real user interaction
      return navigator.clipboard.readText();
    });
    const urlRegex = /\bhttps?:\/\/\S+/g;
    expect(clipboardContent).not.toBeNull();

    if (!clipboardContent) throw new Error("clipboard empty!");

    const extractedUrl = clipboardContent.match(urlRegex)[0];

    await pageDater.goto(extractedUrl);
    await pageDater
      .getByText("You must switch to committed mode in your settings")
      .click();
  });

  test("Dating profile invites committed profile to connect", async ({
    browser,
  }) => {
    test.setTimeout(120000);

    const context1 = await browser.newContext({
      permissions: ["clipboard-read", "clipboard-write"],
    });
    const context2 = await browser.newContext();
    const pageUser1 = await context1.newPage();
    const pageUser2 = await context2.newPage();

    let fullname1 = faker.person.fullName();
    let fullname2 = faker.person.fullName();
    let email1 = faker.internet.email();
    let email2 = faker.internet.email();
    let password = faker.internet.password({ length: 20 });

    let birthdateDateObject = new Date(
      faker.date.birthdate({ min: 18, max: 65, mode: "age" })
    );
    const formattedDateString = birthdateDateObject
      .toISOString()
      .split("T")[0]
      .replace(/\//g, "-");

    // Register User 1
    await registerUser(
      pageUser1,
      fullname1,
      email1,
      password,
      formattedDateString,
      false,
      setDatingPreferences
    );

    // Register User 2
    await registerUser(
      pageUser2,
      fullname2,
      email2,
      password,
      formattedDateString,
      false,
      null
    );

    await pageUser1
      .getByRole("button", { name: "Invite someone directly" })
      .click();

    // Evaluate the clipboard content within a user gesture context
    const clipboardContent = await pageUser1.evaluate(async () => {
      // Ensure the action is performed after a user gesture
      await new Promise((resolve) => setTimeout(resolve, 200)); // Small delay to simulate real user interaction
      return navigator.clipboard.readText();
    });

    const urlRegex = /\bhttps?:\/\/\S+/g;
    expect(clipboardContent).not.toBeNull();

    if (!clipboardContent) throw new Error("clipboard empty!");

    const extractedUrl = clipboardContent.match(urlRegex)[0];

    await pageUser2.goto(extractedUrl);
    await pageUser2.getByText("Invitation accepted").click();
  });

  test("Dark/light modes", async ({ page }) => {
    await page.goto(`${WEBSITE_URL}/`);

    // Click to switch to dark mode
    await page.getByRole("button", { name: "🌚" }).click();
    // Wait for any necessary transitions or JavaScript that applies the class changes
    await page.waitForTimeout(500); // Ideally, use a more reliable synchronization like waitForFunction or expect

    // Verify dark mode
    const htmlElementDark = page.locator("html");
    await expect(htmlElementDark).toHaveClass(/min-h-screen/);
    await expect(htmlElementDark).toHaveClass(/dark/);
    await expect(htmlElementDark).toHaveAttribute("data-theme", "darkmode");

    // Click to switch to light mode
    await page.getByRole("button", { name: "🌞" }).click();
    // Wait for any necessary transitions or JavaScript that applies the class changes
    await page.waitForTimeout(500); // Ideally, use a more reliable synchronization like waitForFunction or expect

    // Verify light mode
    const htmlElementLight = page.locator("html");
    await expect(htmlElementLight).toHaveClass(/min-h-screen/);
    await expect(htmlElementLight).not.toHaveClass(/dark/);
    await expect(htmlElementLight).toHaveAttribute("data-theme", "lightmode");
  });

  test("Delete user data", async ({ page }) => {
    await registerUserWithDefaultInfo(page);

    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Settings").click();

    await expect(
      page.getByRole("heading", { name: "🦈 Dangerous Section" })
    ).toBeVisible();
    await page
      .getByRole("button", { name: "Delete my account and data" })
      .click();
    await page
      .getByRole("button", { name: "Delete account and data right now" })
      .click();

    await page.getByRole("button", { name: "Yes, delete it!" }).click();

    // Get the current URL
    const currentUrl = page.url();
    expect(currentUrl).toBe(`${WEBSITE_URL}/`);
  });
});

test.describe("Emails", () => {
  test("Registration email", async ({ page }) => {
    const emailAddress = await registerUserWithDefaultInfo(page);

    page.goto(EMAIL_CAPTURE_PORTAL_URL);

    clickOnMailpitEmail(page, emailAddress, "Confirm your email address");

    await expect(
      page
        .frameLocator("#preview-html")
        .getByRole("heading", { name: "Welcome to Goship!" })
    ).toBeVisible();
    // await expect(
    //   page.getByText("Confirm your email address", { exact: true })
    // ).toBeVisible();
    const page1Promise = page.waitForEvent("popup");
    await page
      .frameLocator("#preview-html")
      .getByRole("link", { name: "Confirm Email" })
      .click();
    const page1 = await page1Promise;
    await page1.getByText("Your email has been").click();
  });

  test("Password reset", async ({ page }) => {
    const emailAddress = await registerUserWithDefaultInfo(page);
    // Log user out
    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Sign out").click();

    // Go to login flow to request "password reset" email
    await page.getByRole("button", { name: "Log in" }).click();

    await page.getByText("Forgot password?").click();
    await page.getByPlaceholder("johny@hey.com").fill(emailAddress);
    await page.getByRole("button", { name: "Reset password" }).click();
    await expect(page.getByText("An email was sent to reset")).toBeVisible();

    page.goto(EMAIL_CAPTURE_PORTAL_URL);
    clickOnMailpitEmail(page, emailAddress, "Reset your password");

    await expect(
      page.frameLocator("#preview-html").getByText("You recently requested to")
    ).toBeVisible();

    const page2Promise = page.waitForEvent("popup");
    await page
      .frameLocator("#preview-html")
      .getByRole("link", { name: "Reset your password" })
      .click();
    const page2 = await page2Promise;

    const newPassword = faker.internet.password({ length: 20 });
    await page2.getByLabel("Password", { exact: true }).fill(newPassword);
    await page2.getByLabel("Confirm password").fill(newPassword);

    await page2.getByRole("button", { name: "Update password" }).click();
    await expect(page2.getByText("Your password has been")).toBeVisible();

    // Login in with new password
    await page2.goto(`${WEBSITE_URL}/`);
    await page2.getByRole("button", { name: "Log in" }).click();

    await page2.getByPlaceholder("johny@hey.com").fill(emailAddress);
    await page2.getByPlaceholder("•••••••••").fill(newPassword);
    await page2.locator("#login-button").click();

    // Verify user is logged in
    await page2.locator("#navbar-profile-menu").first().click();
    await expect(page2.getByText(emailAddress)).toBeVisible();
    await expect(page2.getByRole("list").getByText("Profile")).toBeVisible();
    await expect(page2.getByRole("list").getByText("Settings")).toBeVisible();
    await expect(page2.getByRole("list").getByText("Sign out")).toBeVisible();
  });
});

// TODO: add test for profile view in committed + dating mode. Check that user can upload images + profile pic
// TODO: test preference views from both committed + dating mode
