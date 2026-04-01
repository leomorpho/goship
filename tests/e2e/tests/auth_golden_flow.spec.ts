import { expect, test } from "@playwright/test";

test("framework repo auth golden flow: register, logout, protected redirect, and login return", async ({ page }) => {
  test.setTimeout(60_000);

  const seed = Date.now().toString();
  const displayName = `Playwright ${seed}`;
  const email = `playwright+${seed}@goship.test`;
  const password = "Password123!";

  await test.step("register submits successfully and lands on an authenticated dashboard surface", async () => {
    await page.goto("/user/register");
    await page.getByLabel("Display Name").fill(displayName);
    await page.getByLabel("Email address").fill(email);
    await page.getByLabel("Password").fill(password);
    await page.locator("#birthdate").fill("1990-01-01");
    await page.locator("form[data-component='register']").evaluate((form) => {
      let hidden = form.querySelector<HTMLInputElement>('input[name="relationship_status"]');
      if (!hidden) {
        hidden = document.createElement("input");
        hidden.type = "hidden";
        hidden.name = "relationship_status";
        form.appendChild(hidden);
      }
      hidden.value = "single";
    });
    await page.getByRole("button", { name: "Register" }).click();

    await page.goto("/auth/preferences");
    await expect(page).toHaveURL(/\/auth\/preferences(\?|$)/);
  });

  await test.step("logout clears session; protected route redirects to login", async () => {
    await page.goto("/auth/logout");
    await page.goto("/auth/profile");
    await expect(page).toHaveURL(/\/user\/login(\?|$)/);
  });

  await test.step("login returns to an authenticated route after protected-route redirect", async () => {
    await page.getByLabel("Email address").fill(email);
    await page.getByLabel("Password").fill(password);
    await page.locator("#login-button").click();

    await expect(page).toHaveURL(/(\/auth\/profile|\/welcome\/preferences)(\?|$)/);
  });
});
