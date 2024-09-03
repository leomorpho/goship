// import { expect, test } from "@playwright/test";
// import { Page } from "playwright";
// import { config, defaultConfig } from "../config.js";
// const environment = process.env.NODE_ENV || "development";
// const { WEBSITE_URL } = config[environment] || defaultConfig;

// /*
// NOTE that before running these tests, you need to have amie running:
// $ export PAGODA_APP_NAME: "Amie"
// $ make run
// */

// test("Landing page", async ({ page }) => {
//   await page.goto("http://localhost:8000/");
//   await page.getByRole("link", { name: "Login" }).click();
//   await page.goto("http://localhost:8000/");
//   await page
//     .locator("#hero-background")
//     .getByRole("link", { name: "Get on the Friends List" })
//     .click();
//   await page.getByLabel("Locate my position").click();
//   await page.getByPlaceholder("Enter email").click();
//   await page.getByPlaceholder("Enter email").fill("test@s.com");
//   await page.locator("#subscribe-form-submit").click();
//   const page1Promise = page.waitForEvent("popup");
//   await page.getByRole("link", { name: "Click here to check out our" }).click();
//   const page1 = await page1Promise;
// });

// test("Onboard users", async ({ browser }) => {
//   for (const email of ["alice@test.com", "bob@test.com", "sandrine@test.com"]) {
//     const context = await browser.newContext();
//     const page = await context.newPage();
//     await page.goto("http://localhost:8000/");
//     await page.getByRole("link", { name: "Login" }).click();
//     await page.getByLabel("Email address").click();
//     await page.getByLabel("Email address").fill(email);
//     await page.getByLabel("Email address").press("Tab");
//     await page.getByPlaceholder("*******").fill("password");
//     await page.getByRole("button", { name: "Log in" }).click();
//     await page.getByRole("button", { name: "×" }).click();

//     // Check if the "Finish onboarding" button is visible
//     const isOnboardingButtonVisible = await page
//       .getByRole("button", { name: "Finish onboarding" })
//       .isVisible();

//     // Skip this user if the onboarding button is not visible
//     if (!isOnboardingButtonVisible) {
//       await context.close();
//       continue;
//     }

//     await page.getByPlaceholder("Tell us about yourself...").click();
//     await page.getByPlaceholder("Tell us about yourself...").press("Meta+a");
//     await page
//       .getByPlaceholder("Tell us about yourself...")
//       .fill("Change the bio");
//     await page.getByLabel("Select my gender").nth(1).selectOption("Man");
//     await page.getByLabel("Select my gender").nth(1).selectOption("Woman");
//     await page
//       .getByRole("checkbox", { name: "Interested in gender: Woman" })
//       .uncheck();
//     await page.getByLabel("Select minimum age").nth(1).selectOption("29");
//     await page.getByLabel("Select maximum age").nth(1).selectOption("52");
//     await page.locator("#map").nth(1).click();
//     await page.getByLabel("Adjust search radius").nth(1).fill("502000");
//     await page.getByLabel("Adjust search radius").nth(1).click();
//     await page.getByRole("button", { name: "Finish onboarding" }).click();
//     await page.getByRole("link", { name: "Icon Amie" }).click();
//     await page.getByRole("link", { name: "Logout" }).click();
//   }
// });

// test("Log-in as a user", async ({ page }) => {
//   await page.goto("http://localhost:8000/");
//   await page.getByRole("link", { name: "Login" }).click();
//   await page.getByLabel("Email address").click();
//   await page.getByLabel("Email address").fill("alice@test.com");
//   await page.getByPlaceholder("*******").click();
//   await page.getByPlaceholder("*******").fill("password");
//   await page.getByRole("button", { name: "Log in" }).click();
//   await page.getByRole("link", { name: "Logout" }).click();
// });

// test("Post a private message", async ({ page }) => {
//   const randomMessage = generateRandomText();

//   await page.goto("http://localhost:8000/");
//   await page.getByRole("link", { name: "Login" }).click();
//   await page.getByLabel("Email address").click();
//   await page.getByLabel("Email address").fill("alice@test.com");
//   await page.getByLabel("Email address").press("Tab");
//   await page.getByPlaceholder("*******").fill("password");
//   await page.getByRole("button", { name: "Log in" }).click();
//   // Attempt to finish onboarding if the button is present within 2 seconds
//   try {
//     await page.waitForSelector("text=Finish onboarding", { timeout: 2000 });
//     await page.getByRole("button", { name: "Finish onboarding" }).click();
//   } catch (error) {
//     console.log(
//       "User is already onboarded, or the button did not appear within 2 seconds."
//     );
//   }
//   await page.getByText("Messages").nth(1).click();
//   await page.getByRole("link", { name: "Bob Lupin" }).click();
//   await page.getByPlaceholder("Write your message here...").click();
//   await page.getByPlaceholder("Write your message here...").fill(randomMessage);
//   await page.locator("#publish-button").click();
//   await page.getByText(randomMessage).click();
// });

// test("Convo: test convo between two different accounts", async ({
//   browser,
// }) => {
//   const homeUrl = "http://localhost:8000/";

//   // Create two separate browser contexts for two different sessions
//   const context1 = await browser.newContext();
//   const context2 = await browser.newContext();

//   // Open new pages in these contexts
//   const page1 = await context1.newPage();
//   const page2 = await context2.newPage();

//   // Simulate User 1 actions
//   await page1.goto(homeUrl);
//   await page1.getByRole("link", { name: "Login" }).click();
//   await page1.getByLabel("Email address").click();
//   await page1.getByLabel("Email address").fill("alice@test.com");
//   await page1.getByLabel("Email address").press("Tab");
//   await page1.getByPlaceholder("*******").fill("password");
//   await page1.getByRole("button", { name: "Log in" }).click();
//   // Attempt to finish onboarding if the button is present within 2 seconds
//   try {
//     await page2.waitForSelector("text=Finish onboarding", { timeout: 2000 });
//     await page2.getByRole("button", { name: "Finish onboarding" }).click();
//   } catch (error) {
//     console.log(
//       "User is already onboarded, or the button did not appear within 2 seconds."
//     );
//   }
//   await page1.getByText("Messages").nth(1).click();
//   await page1.getByRole("link", { name: "Bob Lupin" }).click();

//   // Simulate User 2 actions
//   await page2.goto(homeUrl);
//   await page2.getByRole("link", { name: "Login" }).click();
//   await page2.getByLabel("Email address").click();
//   await page2.getByLabel("Email address").fill("bob@test.com");
//   await page2.getByLabel("Email address").press("Tab");
//   await page2.getByPlaceholder("*******").fill("password");
//   await page2.getByRole("button", { name: "Log in" }).click();
//   // Attempt to finish onboarding if the button is present within 2 seconds
//   try {
//     await page2.waitForSelector("text=Finish onboarding", { timeout: 2000 });
//     await page2.getByRole("button", { name: "Finish onboarding" }).click();
//   } catch (error) {
//     console.log(
//       "User is already onboarded, or the button did not appear within 2 seconds."
//     );
//   }
//   await page2.getByText("Messages").nth(1).click();
//   await page2.getByRole("link", { name: "Alice Bonjovi" }).click();

//   // Check SSE connection for page1
//   try {
//     await checkSSEConnection(page1, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for page1");
//   } catch (error) {
//     console.error("Error verifying SSE connection for page1:", error);
//   }

//   // Check SSE connection for page2
//   try {
//     await checkSSEConnection(page2, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for page2");
//   } catch (error) {
//     console.error("Error verifying SSE connection for page2:", error);
//   }

//   // Make Alice post a new message
//   let randomMessage = generateRandomText();
//   await page1.getByPlaceholder("Write your message here...").click();
//   await page1
//     .getByPlaceholder("Write your message here...")
//     .fill(randomMessage);
//   await page1.locator("#publish-button").click();

//   // Check that both users can see the message
//   await page1.getByText(randomMessage).click();
//   await page2.getByText(randomMessage).click();

//   // Make Bob post a new message
//   randomMessage = generateRandomText();
//   await page2.getByPlaceholder("Write your message here...").click();
//   await page2
//     .getByPlaceholder("Write your message here...")
//     .fill(randomMessage);
//   await page2.locator("#publish-button").click();

//   // Check that both users can see the message
//   await page1.getByText(randomMessage).click();
//   await page2.getByText(randomMessage).click();

//   // TODO: Answer a new question for Alice
//   // await page.locator("div > div > div > a").first().click();
//   // In Bob's convo view he has with Alice, assert the new answer is visible

//   // Clean up: close the pages and contexts
//   await page1.close();
//   await page2.close();
//   await context1.close();
//   await context2.close();
// });

// test("Publish answer", async ({ browser }) => {
//   const homeUrl = "http://localhost:8000/";

//   // Create two separate browser contexts for two different sessions
//   const context1 = await browser.newContext();
//   const context2 = await browser.newContext();

//   // Open new pages in these contexts
//   const page1 = await context1.newPage();
//   const page2 = await context2.newPage();

//   // Simulate User 1 actions
//   await page1.goto(homeUrl);
//   await page1.getByRole("link", { name: "Login" }).click();
//   await page1.getByLabel("Email address").click();
//   await page1.getByLabel("Email address").fill("alice@test.com");
//   await page1.getByLabel("Email address").press("Tab");
//   await page1.getByPlaceholder("*******").fill("password");
//   await page1.getByRole("button", { name: "Log in" }).click();
//   await page1.getByText("Messages").nth(1).click();
//   await page1.getByRole("link", { name: "You" }).click();

//   // Simulate User 2 actions
//   await page2.goto(homeUrl);
//   await page2.getByRole("link", { name: "Login" }).click();
//   await page2.getByLabel("Email address").click();
//   await page2.getByLabel("Email address").fill("alice@test.com");
//   await page2.getByLabel("Email address").press("Tab");
//   await page2.getByPlaceholder("*******").fill("password");
//   await page2.getByRole("button", { name: "Log in" }).click();
//   await page2.getByText("Feed").nth(1).click();

//   // Answer a question
//   await page2.locator(".flex > a").first().click();
//   await page2.getByPlaceholder("Type your answer here...").click();
//   let randomMessage = generateRandomText(500);
//   await page2.getByPlaceholder("Type your answer here...").fill(randomMessage);
//   await page2.getByRole("button", { name: "Publish" }).click();
//   // Close the confirmation message saying the answer was published.
//   await page2.getByRole("button", { name: "×" }).click();

//   // Check SSE connection for page1
//   try {
//     await checkSSEConnection(page1, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for page1");
//   } catch (error) {
//     console.error("Error verifying SSE connection for page1:", error);
//   }
//   // Check that SSE message was received
//   await page1.getByText(randomMessage).click();

//   // Clean up: close the pages and contexts
//   await page1.close();
//   await page2.close();
//   await context1.close();
//   await context2.close();
// });

// test("New message notifications", async ({ browser }) => {
//   const homeUrl = "http://localhost:8000/";

//   // Create two separate browser contexts for two different sessions
//   const context1 = await browser.newContext();
//   const context2 = await browser.newContext();

//   // Open new pages in these contexts
//   const alice = await context1.newPage();
//   const bob = await context2.newPage();

//   // Simulate User 1 actions
//   await alice.goto(homeUrl);
//   await alice.getByRole("link", { name: "Login" }).click();
//   await alice.getByLabel("Email address").click();
//   await alice.getByLabel("Email address").fill("alice@test.com");
//   await alice.getByLabel("Email address").press("Tab");
//   await alice.getByPlaceholder("*******").fill("password");
//   await alice.getByRole("button", { name: "Log in" }).click();
//   await alice.getByText("Messages").nth(1).click();
//   await alice.getByRole("link", { name: "You" }).click();
//   await alice.getByText("Messages").nth(1).click();

//   // Simulate User 2 actions
//   await bob.goto(homeUrl);
//   await bob.getByRole("link", { name: "Login" }).click();
//   await bob.getByLabel("Email address").click();
//   await bob.getByLabel("Email address").fill("bob@test.com");
//   await bob.getByLabel("Email address").press("Tab");
//   await bob.getByPlaceholder("*******").fill("password");
//   await bob.getByRole("button", { name: "Log in" }).click();
//   await bob.getByText("Messages").nth(1).click();
//   await bob.getByRole("link", { name: "Alice Bonjovi" }).click();

//   // Capture the initial notification count
//   let numUnseenMessagesForAliceSideBarInt = await alice
//     .locator("#message-notifications-count")
//     .nth(0)
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   let numUnseenMessagesForAliceConvosInt = await alice
//     .locator("div:has-text('Bob Lupin') .unseen-notification-count")
//     .first()
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   // Make Bob post a new message
//   let randomMessage = generateRandomText();
//   await bob.getByPlaceholder("Write your message here...").click();
//   await bob.getByPlaceholder("Write your message here...").fill(randomMessage);
//   await bob.locator("#publish-button").click();

//   // Check SSE connection for alice
//   try {
//     await checkSSEConnection(alice, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for alice");
//   } catch (error) {
//     console.error("Error verifying SSE connection for alice:", error);
//   }

//   // Introduce a sleep to wait for the notification to be processed
//   await alice.waitForTimeout(1000);

//   const numNewUnseenMessagesForAliceSideBarInt = await alice
//     .locator("#message-notifications-count")
//     .nth(0)
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   const numNewUnseenMessagesForAliceConvosInt = await alice
//     .locator("div:has-text('Bob Lupin') .unseen-notification-count")
//     .first()
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   // Check the notifications in the side menu
//   expect(numUnseenMessagesForAliceSideBarInt + 1).toEqual(
//     numNewUnseenMessagesForAliceSideBarInt
//   );

//   // Check the notifications in the convos page
//   expect(numUnseenMessagesForAliceConvosInt + 1).toEqual(
//     numNewUnseenMessagesForAliceConvosInt
//   );

//   // Clean up: close the pages and contexts
//   await alice.close();
//   await bob.close();
//   await context1.close();
//   await context2.close();
// });

// test("Unfriend/refriend", async ({ browser }) => {
//   const homeUrl = "http://localhost:8000/";

//   // Create two separate browser contexts for two different sessions
//   const context1 = await browser.newContext();
//   const context2 = await browser.newContext();

//   // Open new pages in these contexts
//   const alice = await context1.newPage();
//   const bob = await context2.newPage();

//   await alice.goto(homeUrl);
//   await alice.getByRole("link", { name: "Login" }).click();
//   await alice.getByLabel("Email address").click();
//   await alice.getByLabel("Email address").fill("alice@test.com");
//   await alice.getByLabel("Email address").press("Tab");
//   await alice.getByPlaceholder("*******").fill("password");
//   await alice.getByRole("button", { name: "Log in" }).click();

//   await bob.goto(homeUrl);
//   await bob.getByRole("link", { name: "Login" }).click();
//   await bob.getByLabel("Email address").click();
//   await bob.getByLabel("Email address").fill("bob@test.com");
//   await bob.getByLabel("Email address").press("Tab");
//   await bob.getByPlaceholder("*******").fill("password");
//   await bob.getByRole("button", { name: "Log in" }).click();

//   // Clean up notifications and invitations before test
//   await deleteAllNotifications(alice);
//   await deleteAllNotifications(bob);
//   await deleteAllInvitations(alice);
//   await deleteAllInvitations(bob);

//   // Alice unfriends Bob. Not sure why, but clicking once on Messages
//   // sometimes doesn't work
//   await alice.getByText("Messages").nth(1).click();

//   // Alice checks if Bob is in her friend list
//   const bobInFriendList = await alice
//     .locator('a:has-text("Bob Lupin")')
//     .first()
//     .isVisible();

//   console.log(`has bob lupin: ${bobInFriendList}`);
//   // If Bob is found, proceed to unfriend him
//   if (bobInFriendList) {
//     await alice
//       .locator("#convos div")
//       .filter({ hasText: "Bob Lupin Unfriend" })
//       .locator("svg")
//       .click();
//     await alice
//       .locator("#convos")
//       .getByRole("listitem")
//       .getByText("Unfriend")
//       .click();
//     await alice.getByRole("button", { name: "OK" }).click();
//   }

//   // Capture the initial notification count
//   let numUnreadNotificationsForAlice = await alice
//     .locator("#normal-notifications-count")
//     .nth(0)
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   let numUnreadNotificationsForBob = await bob
//     .locator("#normal-notifications-count")
//     .nth(0)
//     .innerText()
//     .then((text) => parseInt(text, 10));

//   // Alice creates an invitation for Bob
//   await alice.getByText("Add friend").nth(1).click();
//   await alice.getByPlaceholder("Type name here").click();
//   await alice.getByPlaceholder("Type name here").fill(generateRandomText(6));
//   await alice.getByRole("button", { name: "Create invitation" }).click();

//   // Locate the invite link on the page
//   const inviteLinkLocator = alice.locator(".inviteText").first();
//   const invitationText = await inviteLinkLocator.textContent();

//   if (!invitationText) {
//     throw new Error("Invitation text not found");
//   }
//   console.log(invitationText); // for debugging, to see the content

//   // Regular expression to match the URL
//   const urlRegex = /http:\/\/localhost:8000\/auth\/i\/[\w\-_.~]+/;

//   // Extract the URL
//   const match = invitationText.match(urlRegex);
//   const extractedUrl = match ? match[0] : null;

//   await bob.waitForSelector("text=Messages", { state: "attached" });
//   await bob.getByText("Messages").nth(1).click();

//   // Let Bob accept Alice's invitation
//   if (extractedUrl) {
//     // Let Bob accept Alice's invitation
//     await bob.goto(extractedUrl);
//   } else {
//     throw new Error("Invitation URL not found");
//   }

//   await bob.getByRole("heading", { name: "Alice Bonjovi" }).click();

//   // Check SSE connection for bob
//   try {
//     await checkSSEConnection(bob, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for bob");
//   } catch (error) {
//     console.error("Error verifying SSE connection for bob:", error);
//   }
//   // Check SSE connection for alice
//   try {
//     await checkSSEConnection(alice, "http://localhost:8000/auth/realtime");
//     console.log("SSE connection verified for alice");
//   } catch (error) {
//     console.error("Error verifying SSE connection for alice:", error);
//   }

//   // // TODO: hack, wait for notification
//   // await bob.waitForTimeout(1000);
//   // await alice.waitForTimeout(1000);

//   // // There should be no notifications for Bob
//   // // Select all notifications with a class starting with "notification-card-"
//   // await bob.getByText("Notifications").nth(1).click();
//   // const bobNotifications = await bob.locator(".notification-card");

//   // // Count the number of notifications
//   // let count = await bobNotifications.count();

//   // // Assert that there are notifications
//   // expect(count).toBe(0);
//   // Capture the new notification count
//   // let newNumUnreadNotificationsForAlice = await alice
//   //   .locator("#normal-notifications-count")
//   //   .nth(0)
//   //   .innerText()
//   //   .then((text) => parseInt(text, 10));

//   // let newNumUnreadNotificationsForBob = await bob
//   //   .locator("#normal-notifications-count")
//   //   .nth(0)
//   //   .innerText()
//   //   .then((text) => parseInt(text, 10));

//   // // Check the notifications in the side menu
//   // expect(numUnreadNotificationsForAlice).toEqual(
//   //   newNumUnreadNotificationsForAlice + 1
//   // );

//   // // Check the notifications in the convos page
//   // expect(numUnreadNotificationsForBob).toEqual(newNumUnreadNotificationsForBob);

//   // There should be exactly 1 notification for Alice, go take a look
//   await bob.waitForSelector("text=Notifications", { state: "attached" });
//   await alice.getByText("Notifications").nth(1).click();
//   await expect(
//     alice.getByText("You are now friends with Bob Lupin")
//   ).toBeVisible();

//   const aliceNotifications = await alice.locator(".notification-card");

//   // Count the number of notifications
//   let count = await aliceNotifications.count();
//   console.log(`received ${count} notifications`);
//   expect(count).toBe(1);

//   const notificationSelector =
//     '.notification-card p:has-text("You are now friends with Bob Lupin")';
//   await expect(alice.locator(notificationSelector)).toBeVisible();

//   // await alice.getByRole("link", { name: "More info" }).nth(1).click();
//   // await expect(
//   //   alice.getByRole("heading", { name: "Alice Bonjovi" })
//   // ).toBeVisible();

//   // Notifications for alice should now be all read
//   // let finalNumUnreadNotificationsForAlice = await alice
//   //   .locator("#normal-notifications-count")
//   //   .nth(0)
//   //   .innerText()
//   //   .then((text) => parseInt(text, 10));
//   // expect(finalNumUnreadNotificationsForAlice).toEqual(0);

//   await deleteAllNotifications(alice);
//   await deleteAllNotifications(bob);
// });

// async function deleteAllNotifications(userPage) {
//   await userPage.waitForSelector("text=Notifications", { state: "attached" });
//   await userPage.getByText("Notifications").nth(1).click();

//   const notifications = await userPage.locator(".notification-card");
//   const count = await notifications.count();

//   for (let i = 0; i < count; i++) {
//     const notification = notifications.nth(i);
//     await notification.getByRole("img").click();
//     await notification.getByText("Delete").click();
//   }
// }

// async function deleteAllInvitations(userPage) {
//   await userPage.waitForSelector("text=Add Friend", { state: "attached" });
//   await userPage.getByText("Add Friend").nth(1).click();
//   // Find all invitation links within #convos
//   const invitations = await userPage.locator("#convos a");
//   const count = await invitations.count();
//   console.log("deleting all invitations");
//   // Loop through each invitation and delete it
//   for (let i = 0; i < count; i++) {
//     await invitations.nth(i).click();
//     await userPage.getByRole("button", { name: "OK" }).click();
//   }
// }

// async function deleteAllDrafts(page) {
//   try {
//     // Wait for the first delete button to be loaded in the DOM or timeout after 5 seconds
//     await page.waitForSelector(".deleteDraft", {
//       state: "attached",
//       timeout: 5000,
//     });

//     // Now, get all delete buttons. Assuming they are loaded by the time the first one appears.
//     const deleteButtons = await page.$$(".deleteDraft");

//     for (const button of deleteButtons) {
//       // Click the delete button to trigger the confirmation dialog
//       // Use waitForSelector to ensure the button is clickable
//       await button.click();

//       // Wait for the confirmation dialog to appear
//       // This assumes a specific role and name for the "OK" button, adjust accordingly.
//       // Note: The previous code snippet was incorrect for clicking the OK button.
//       // The correct way is below, assuming the OK button has a specific accessible name or text.
//       await page.locator("button", { hasText: "OK" }).click();
//     }
//   } catch (error) {
//     // Handle the case where the delete buttons did not load within 5 seconds
//     console.error(
//       "No delete items loaded within 5 seconds or other error: ",
//       error
//     );
//   }
// }

// test("Drafts create/delete", async ({ page }) => {
//   await page.goto("http://localhost:8000/");
//   await page.getByText("Amie Login Go to Chérie for").click();
//   await page.getByRole("link", { name: "Login" }).click();
//   await page.getByLabel("Email address").click();
//   await page.getByLabel("Email address").fill("alice@test.com");
//   await page.getByLabel("Email address").press("Tab");
//   await page.getByPlaceholder("*******").fill("password");
//   await page.getByPlaceholder("*******").press("Enter");
//   await page.locator("a").filter({ hasText: "Feed" }).nth(1).click();

//   await deleteAllDrafts(page);

//   // Reload draft page by navigating back to it
//   await page.locator("a").filter({ hasText: "Feed" }).nth(1).click();
//   await page.getByRole("button", { name: "Drafts" }).click();
//   await page.getByText("You don't have any drafts yet 🦗🦗🦗").click();

//   // Click on a question to answer it
//   await page.locator("a").filter({ hasText: "Feed" }).nth(1).click();
//   await page.locator(".flex > a").first().click();
//   await page.getByPlaceholder("Type your answer here...").click();

//   let text = generateRandomText(100);
//   await page.getByPlaceholder("Type your answer here...").fill(text);
//   await page.getByRole("button", { name: "Save" }).click();

//   // Go back to home feed
//   await page.locator("a").filter({ hasText: "Feed" }).nth(1).click();
//   await page.getByRole("button", { name: "Drafts" }).click();
//   await expect(page.getByText(text)).toBeVisible();

//   // Delete draft
//   await page.locator(".bg-yellow-200").click();
//   await page.getByRole("button", { name: "OK" }).click();

//   // Reload draft page by navigating back to it
//   await page.locator("a").filter({ hasText: "Feed" }).nth(1).click();
//   await page.getByRole("button", { name: "Drafts" }).click();
//   await page.getByText("You don't have any drafts yet 🦗🦗🦗").click();
// });

// // test("Test hiding/downvoting question", async ({ page }) => {
// //   await page.goto("http://localhost:8000/");
// //   await page.getByRole("link", { name: "Login" }).click();
// //   await page.getByLabel("Email address").click();
// //   await page.getByLabel("Email address").fill("alice@test.com");
// //   await page.getByLabel("Email address").press("Tab");
// //   await page.getByPlaceholder("*******").fill("password");
// //   await page.getByRole("button", { name: "Log in" }).click();
// //   await page.getByText("Feed").nth(1).click();

// //   // Get the text of the first question within the class 'questionCard'
// //   const firstQuestionLocator = page.locator(".questionCard .text-lg").first();
// //   const firstQuestionText = await firstQuestionLocator.textContent();

// //   // Click to hide/downvote the first question
// //   await page.locator(".questionCard .text-black > .w-6").first().click();

// //   // Confirm the action
// //   await expect(page.getByRole("heading", { name: "Confirm" })).toBeVisible();
// //   await page.getByText("Downvote and hide this").click();
// //   await page.getByRole("button", { name: "OK" }).click();

// //   // Wait for a moment to ensure the action completes
// //   await page.waitForTimeout(1250);

// //   // Check that the question text is no longer present on the page
// //   const pageContent = await page.content();
// //   expect(pageContent).not.toContain(firstQuestionText);
// // });

// // TODO: add test: delete answer (in self and shared thread)

// // test("Test changing profile pic", async ({ page }) => {
// //   await page.goto("http://localhost:8000/");
// //   await page
// //     .getByRole("navigation")
// //     .getByRole("link", { name: "Login" })
// //     .click();
// //   await page.getByLabel("Email address").click();
// //   await page.getByLabel("Email address").fill("alice@test.com");
// //   await page.getByLabel("Email address").press("Tab");
// //   await page.getByPlaceholder("*******").fill("password");
// //   await page.getByRole("button", { name: "Log in" }).click();
// //   await page.locator("#profile").click();
// //   await page.locator("#edit-profile-pic").click();
// //   await page.locator("#imageInput").click();
// //   await page.locator("#imageInput").click();
// //   await page.locator("#imageInput").setInputFiles("testdata/photos/1.jpg");
// //   await page.getByRole("button", { name: "Upload Photo" }).click();
// // });

// // TODO: test that drafts show up after being created.

// function generateRandomText(length = 100) {
//   const characters =
//     "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 ";
//   let result = "";
//   for (let i = 0; i < length; i++) {
//     result += characters.charAt(Math.floor(Math.random() * characters.length));
//   }
//   return result;
// }

// // Function to check SSE connection
// const checkSSEConnection = (page: Page, url: string): Promise<void> => {
//   return new Promise<void>((resolve, reject) => {
//     page.on("response", (response) => {
//       if (response.url() === url && response.status() === 200) {
//         console.log(`SSE connection established on ${url}`);
//         resolve();
//       }
//     });

//     // Set a timeout to reject the promise if SSE connection is not established
//     setTimeout(() => {
//       reject(`SSE connection not established on ${url}`);
//     }, 5000); // Adjust the timeout as needed
//   });
// };
