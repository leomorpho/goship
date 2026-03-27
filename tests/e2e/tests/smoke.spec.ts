import { expect, test } from "@playwright/test";

test("smoke: server starts and serves landing page", async ({ page, request }) => {
  const health = await request.get("/up");
  expect(health.ok()).toBeTruthy();

  const landing = await request.get("/");
  expect(landing.ok()).toBeTruthy();

  const login = await request.get("/user/login");
  expect(login.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page).toHaveURL(/\/$/);

  const loginPage = await page.goto("/user/login");
  expect(loginPage?.ok()).toBeTruthy();
  await expect(page).toHaveURL(/\/user\/login$/);
});
