import { faker } from "@faker-js/faker";
import { Page, expect } from "@playwright/test";
import { config, defaultConfig } from "../config.js";
const environment = process.env.NODE_ENV || "development";
const { WEBSITE_URL, EMAIL_URL } = config[environment] || defaultConfig;

export async function registerUser(
  page,
  fullname,
  email,
  password,
  birthdate,
  isMobileView,
  setDatingPreferencesFunc
): Promise<string> {
  console.log(birthdate);

  await page.goto(`${WEBSITE_URL}/`);

  await page.getByRole("button", { name: "Log in" }).click();
  // await page
  //   .locator("a")
  //   .filter({ hasText: /^Login$/ })
  //   .click();
  // }
  await page.getByText("Create an account").click();
  if (setDatingPreferencesFunc) {
    await page.getByText("✅🌹 I am looking to date").click();
  } else {
    await page.getByText("✅💋 I'm in a committed").click();
  }
  console.log(`Registering with fullname ${fullname}`);
  await page.getByPlaceholder("JohnWatts123").fill(fullname);

  console.log(`Registering with email ${email}`);
  await page.getByPlaceholder("steamyjohn@diesel.com").fill(email);

  console.log(`Registering with password ${password}`);
  await page.getByPlaceholder("•••••••••").fill(password);

  console.log("Checking birthday");
  await page.getByLabel("Birthdate (you need to be 18").fill(birthdate);

  console.log("Before register");
  await page.getByRole("button", { name: "Register", exact: true }).click();
  console.log("After register");

  await page.getByText("SuccessYour account has been").click();
  await page.getByText("InfoAn email was sent to you").click();

  if (setDatingPreferencesFunc) {
    await setDatingPreferencesFunc(page);
  }

  await page.getByLabel("Finish onboarding").click();

  if (setDatingPreferencesFunc) {
    await completeOnboardingDriverJSFlow(page, false);
  } else {
    await completeOnboardingDriverJSFlow(page, true);
  }

  return email;
}

export async function completeOnboardingDriverJSFlow(
  page: Page,
  committedMode: boolean
) {
  await expect(page.getByText("Welcome!")).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  await expect(page.getByText("Questions", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  await expect(page.getByText("Rate and answer questions")).toBeVisible();
  await expect(page.getByText("Please like or dislike")).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  await expect(page.getByText("New questions", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  await page.getByRole("button", { name: "Next →" }).click();
  await page.getByRole("button", { name: "Next →" }).click();
  await page.getByRole("button", { name: "Next →" }).click();
  await page.getByRole("button", { name: "Next →" }).click();
  if (committedMode) {
    await page.getByRole("button", { name: "Next →" }).click();
  }
  await page.getByRole("button", { name: "Next →" }).click();
  await expect(page.getByText("Almost done...to get back to")).toBeVisible();
  await page.getByRole("button", { name: "Next →" }).click();
  await expect(page.getByText("Happy Connecting!")).toBeVisible();
  await page.getByRole("button", { name: "Done" }).click();
}

export async function registerUserWithDefaultInfo(page: Page): Promise<string> {
  let birthdateDateObject = new Date(
    faker.date.birthdate({ min: 18, max: 65, mode: "age" })
  );
  const formattedDateString = birthdateDateObject
    .toISOString()
    .split("T")[0]
    .replace(/\//g, "-");

  const emailAddress = await registerUser(
    page,
    faker.person.fullName(),
    faker.internet.email(),
    faker.internet.password({ length: 20 }),
    formattedDateString,
    false,
    null
  );
  return emailAddress;
}

export async function setDatingPreferences(page: Page) {
  await page
    .getByPlaceholder("Share something interesting about yourself...")
    .fill("I am Samir");
  await page.getByLabel("Default select example").selectOption("CA");
  await page.getByPlaceholder("234 5678").fill("6043392552");
  // TODO: below is commented out because meeting people on our platform is disabled for now. Once enabled, uncomment.
  // await page.getByLabel("Select my gender").selectOption("Man");

  // await page
  //   .locator("#genders-multiselect-placeholder")
  //   .getByRole("img")
  //   .click();
  // await page
  //   .locator("#genders-multiselect-placeholder")
  //   .getByRole("option", { name: "Man", exact: true })
  //   .click();
  // await page
  //   .locator("#genders-multiselect-placeholder")
  //   .getByRole("option", { name: "Woman" })
  //   .click();
  // await page.getByLabel("Select minimum age").selectOption("18");
  // await page.getByLabel("Select maximum age").selectOption("60");
  // await page.getByLabel("Max distance: 50000 km").fill("2007");
}

export async function logUserOut(page, isMobileView: boolean = false) {
  if (isMobileView) {
    await page.locator("#bottom-navbar").getByRole("button").nth(2).click();
    await page.getByRole("button", { name: "Log out" }).click();
  } else {
    await page.locator("#navbar-profile-menu").first().click();
    await page.getByRole("list").getByText("Sign out").click();
  }
}

export async function logUserIn(
  page,
  emailAddress: string,
  password: string,
  isMobileView: boolean = false
) {
  await page.getByRole("button", { name: "Log in" }).click();
  await page.getByPlaceholder("johny@hey.com").fill(emailAddress);
  await page.getByPlaceholder("•••••••••").fill(password);
  await page.getByRole("button", { name: "Log in" }).click();
}

export async function acceptInvitationByLink(
  pageSender,
  pageReceiver,
  isMobileView: boolean = false,
  isReceiverLoggedIn: boolean = true
) {
  // user2 will share invitation with user1
  if (isMobileView) {
    await pageSender
      .locator("#bottom-navbar")
      .getByRole("button")
      .nth(2)
      .click();
    await pageSender.locator("#bottom-navbar-settings-button").click();
  } else {
    await pageSender.locator("#navbar-profile-menu").first().click();
    await pageSender.getByRole("list").getByText("Settings").click();
  }

  const invitationText = await pageSender
    .locator("#invitation-text")
    .first()
    .textContent();
  const urlRegex = /\bhttps?:\/\/\S+/g;
  const extractedUrl = invitationText.match(urlRegex)[0];

  if (!extractedUrl) throw new Error("Invitation URL not found");

  // user1 accepts user2's invitation
  await pageReceiver.goto(extractedUrl);

  if (isReceiverLoggedIn) {
    await pageReceiver.getByText("Invitation accepted").click();

    if (isMobileView) {
      await pageReceiver.locator("#bottom-navbar-settings-button").click();
      await pageSender.locator("#bottom-navbar-settings-button").click();
    } else {
      await pageReceiver.getByRole("link", { name: "Logo Goship" }).click();

      await pageSender.getByRole("link", { name: "Logo Goship" }).click();
    }
  }
}

export async function seeSelfConvo(page) {
  await page.getByRole("link", { name: "Logo Goship" }).click();

  await page.getByRole("button", { name: "All Conversations" }).click();
  await expect(page.locator("a").filter({ hasText: "You" })).toBeVisible();
}

export async function createDraftAndVerifyExists(
  page,
  questionID: number
): Promise<string> {
  // Check that user has no drafts right now
  await page.getByRole("link", { name: "Logo Goship" }).click();

  await page
    .getByRole("button", { name: "Drafts" })
    .waitFor({ state: "visible" });
  await page.getByRole("button", { name: "Drafts" }).click();

  await expect(
    page.getByText("You don't have any drafts yet 🦗🦗🦗")
  ).toBeVisible();

  const url = `${WEBSITE_URL}/auth/questions/${questionID}/text-draft`;
  await page.goto(url);

  // Create a draft
  const draftText = faker.lorem.paragraph();

  // We're now in the answer writing page
  await expect(page.getByRole("button", { name: "Publish" })).toBeVisible();
  await page
    .locator('#answer-writer-component .ProseMirror[contenteditable="true"]')
    .fill(draftText);
  await page.waitForTimeout(1000);

  // Go check out draft in drafts window
  // Check that user has the new draft
  await page.getByRole("link", { name: "Logo Goship" }).click();

  await page.getByRole("button", { name: "Drafts" }).click();
  await expect(page.getByText(draftText)).toBeVisible();

  return draftText;
}

export async function publishAnswer(page, questionID: number) {
  const url = `${WEBSITE_URL}/auth/questions/${questionID}/text-draft`;

  page.goto(url);

  // Create a draft
  const answerText = faker.lorem.paragraph();

  // We're now in the answer writing page
  await expect(page.getByRole("button", { name: "Publish" })).toBeVisible();

  // Get and print the current URL before submitting the answer
  const currentUrl = page.url();
  console.log("Current page URL before publishing the answer:", currentUrl);

  // Try to submit without inputting any text
  await page.getByRole("button", { name: "Publish" }).click();
  await expect(page.getByText("This field is required.")).toBeVisible();

  await page
    .locator('#answer-writer-component .ProseMirror[contenteditable="true"]')
    .fill(answerText);

  await page.getByRole("button", { name: "Publish" }).click();
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

export async function publishQuizAnswers(page, questionID?: number) {
  if (questionID) {
    await page.goto(`${WEBSITE_URL}/auth/questions/${questionID}/quiz-draft`);
  } else {
    await page.getByRole("link", { name: "Logo Goship" }).click();
    await page.locator(".quizQuestion").first().click();
  }

  // Create a draft
  await page.getByRole("button", { name: "Leave additional comment" }).click();
  await page.getByRole("slider").fill("63");
  await page.getByRole("button", { name: "Leave additional comment" }).click();
  await page.getByRole("textbox").fill(faker.lorem.paragraph());
  await page.getByRole("button", { name: "next" }).click();
  await page.getByRole("slider").fill("100");
  await page.getByRole("button", { name: "Leave additional comment" }).click();

  await page.getByRole("textbox").fill(faker.lorem.paragraph());
  await page.getByRole("button", { name: "next" }).click();
  await page.getByRole("slider").fill("83");
  await page.getByRole("button", { name: "Leave additional comment" }).click();

  await page.getByRole("textbox").fill(faker.lorem.paragraph());
  await page.getByRole("button", { name: "next" }).click();
  await page.getByRole("button", { name: "Leave additional comment" }).click();
  await page.getByRole("textbox").fill(faker.lorem.paragraph());

  await page.getByRole("button", { name: "next" }).click();
  await page.getByRole("slider").fill("69");
  await page.getByRole("button", { name: "Leave additional comment" }).click();

  await page.getByRole("textbox").fill(faker.lorem.paragraph());
  await sleep(2000);
  await page.getByRole("button", { name: "Submit" }).click();

  const currentUrl = page.url();
  // Regular expression to extract the integer from the URL
  const regex = /\/questions\/(\d+)\//;
  // Use the match method to find the integer
  const match = currentUrl.match(regex);

  const extractedInt = match && match[1] ? parseInt(match[1], 10) : questionID;
  return extractedInt;
}

async function downvoteQuestion(page) {
  await page.locator(".downvote-question").first().click();
  await page.getByRole("button", { name: "OK" }).click();
}

export async function checkNavbarNotificationCount(
  page,
  expectedCount: number,
  timeout: number = 500000
) {
  // Check navbar notifications
  // Wait for the element to be visible on the page before trying to interact with it
  const navNotificationCountLocator = page.locator(
    "#normal-notifications-count-navbar"
  );

  if (expectedCount === 0) {
    // Check that the notification count element does not exist or is not visible
    // This assumes the element is either not added to the DOM or is hidden via CSS when the count is zero
    await expect(navNotificationCountLocator).toHaveText(/^(0|)$/);
  } else {
    await navNotificationCountLocator.waitFor();
    await expect(navNotificationCountLocator).toHaveText(
      expectedCount.toString(),
      {
        timeout: timeout,
      }
    );
  }
}

export async function checkBottomNavbarNotificationCount(
  page,
  expectedCount: number,
  timeout: number = 200000
) {
  // Check navbar notifications
  // Wait for the element to be visible on the page before trying to interact with it
  const navNotificationCountLocator = page.locator(
    "#normal-notifications-count-bottom-navbar"
  );
  if (expectedCount === 0) {
    // Check that the notification count element does not exist or is not visible
    // This assumes the element is either not added to the DOM or is hidden via CSS when the count is zero
    await expect(navNotificationCountLocator).toHaveText(/^(0|)$/);
  } else {
    await navNotificationCountLocator.waitFor();
    await expect(navNotificationCountLocator).toHaveText(
      expectedCount.toString(),
      {
        timeout: timeout,
      }
    );
  }
}

export async function checkLatestNotification(
  page,
  notificationText: string,
  isMobileView: boolean = false,
  clickOnMoreInfo: boolean = false,
  buttonText: string = "See more"
) {
  if (isMobileView) {
    await page.locator("#bottom-navbar-notifications-button").click();
  } else {
    await page.locator("#notifications-navbar").click();
  }

  console.log("notificationText", notificationText);
  await expect(page.locator('[id^="notification-card"]').first()).toContainText(
    notificationText
  );

  if (clickOnMoreInfo) {
    // Locate the notification card that contains the expected text
    const notificationCardLocator = page
      .locator('[id^="notification-card"]')
      .filter({
        hasText: notificationText,
      });

    console.log("buttonText in checkLatestNotification", buttonText);
    // Within that specific card, find and click on the "More info" link
    const moreInfoLinkLocator = notificationCardLocator.getByRole("button", {
      name: buttonText,
    });
    await moreInfoLinkLocator.click();
  }
}

export async function addEmojiToFirstVisibleAnswer(page, emoji = "😀") {
  // Trigger the emoji picker
  let emojiPicker = page.locator('[id^="emoji-picker"]').first();
  await emojiPicker.click();

  // Wait for the emoji picker container to become visible.
  // Since the container is at the end of the document and might
  // have a specific structure, use a more global and precise selector.
  let emojiPickerContainer = page
    .locator(".picker-container:visible em-emoji-picker")
    .first();

  await emojiPickerContainer.waitFor({ timeout: 50000 }); // timeout is in milliseconds, 10000 ms = 10 seconds

  // Locate and click the specific emoji within the picker
  const emojiLocator = emojiPickerContainer.locator(`text="${emoji}"`).first();
  await emojiLocator.click();
  console.log("done checking emoji");
}

export async function removeEmojiFromFirstAnswer(page, emojiCharacter = "😁") {
  // Locate the specific emoji within the interactions container
  const emojiLocator = page.locator(
    `.emoji-interactions span:text("${emojiCharacter}")`
  );

  // Check if the emoji is visible and click it to remove
  if (await emojiLocator.isVisible()) {
    await emojiLocator.click();
    console.log(`Emoji '${emojiCharacter}' was clicked to remove.`);
  } else {
    console.log(`Emoji '${emojiCharacter}' not found or already removed.`);
  }
}

export async function verifyEmojiExists(page, emojiCharacter = "😁") {
  // Selector to locate the emoji interactions container and the specific emoji
  const emojiLocator = page.locator(
    `.emoji-interactions span:text("${emojiCharacter}")`
  );

  // Check if the emoji is visible in the interactions container
  try {
    await emojiLocator.waitFor({ state: "visible", timeout: 5000 });
    console.log(`Emoji '${emojiCharacter}' is present.`);
    return true; // Return true if the emoji is found and visible
  } catch (error) {
    console.error(`Emoji '${emojiCharacter}' is not present.`);
    return false; // Return false if the emoji is not found within the timeout
  }
}

export async function verifyEmojiDoesNotExist(
  page,
  emojiCharacter = "😁",
  timeout = 50000
) {
  // Selector to locate the emoji interactions container and the specific emoji
  const emojiLocator = page.locator(
    `.emoji-interactions span:text("${emojiCharacter}")`
  );

  // Check if the emoji is not visible in the interactions container
  try {
    await emojiLocator.waitFor({ state: "hidden", timeout: timeout });
    console.log(`Emoji '${emojiCharacter}' is not present.`);
    return true; // Return true if the emoji is confirmed not visible
  } catch (error) {
    console.error(`Emoji '${emojiCharacter}' is still present.`);
    return false; // Return false if the emoji is still visible after the timeout
  }
}

export async function clickOnMailpitEmail(
  page,
  emailAddress: string,
  subject: string
) {
  // Use a locator that finds an `a` element containing the specified email and confirmation text
  const linkLocator = page.locator(
    `a:has(.privacy:has-text("${emailAddress}")):has(.subject:has-text("${subject}"))`
  );

  // Click on the first matching link
  await linkLocator.first().click();
}
