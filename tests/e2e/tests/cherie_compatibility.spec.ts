import { expect, test } from "@playwright/test";

test("framework repo cherie compatibility: boot, auth, realtime, and admin baseline", async ({ page, request }) => {
  const health = await request.get("/up");
  expect(health.ok()).toBeTruthy();

  await page.goto("/");
  await expect(page.locator('[data-component="landing-page"]')).toBeVisible();
  await expect(page.getByRole("link", { name: "Create account" })).toBeVisible();
  await expect(page.getByRole("link", { name: "Log in" })).toBeVisible();

  const login = await request.get("/user/login");
  expect(login.ok()).toBeTruthy();

  await page.goto("/user/login");
  const loginPage = page.locator('[data-component="login"]');
  await expect(loginPage).toBeVisible();
  await expect(loginPage.getByRole("button", { name: "Log in" })).toBeVisible();

  const realtime = await request.get("/auth/realtime", { maxRedirects: 0 });
  expect(realtime.status()).toBe(303);
  expect(realtime.headers()["location"]).toBe("/user/login");

  const admin = await request.get("/auth/admin", { maxRedirects: 0 });
  expect(admin.status()).toBe(303);
  expect(admin.headers()["location"]).toBe("/user/login");
});
