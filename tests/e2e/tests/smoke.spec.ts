import { expect, test } from "@playwright/test";

test("smoke: server starts and serves landing page", async ({ page, request }) => {
  const health = await request.get("/up");
  expect(health.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator('[data-component="landing-page"]')).toBeVisible();
  await expect(page.getByRole("link", { name: "Create account" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Log in" })).toBeVisible();

  const login = await request.get("/user/login");
  expect(login.ok()).toBeTruthy();

  await page.goto("/user/login");
  await expect(page.locator('[data-component="login"]')).toBeVisible();
  await expect(page.getByRole("button", { name: "Log in" })).toBeVisible();
});
