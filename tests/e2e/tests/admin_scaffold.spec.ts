import { expect, test } from "@playwright/test";

async function registerAdminUser(page, email: string, password: string) {
  await page.goto("/user/register");
  await page.evaluate(() => {
    const form = document.querySelector('form[data-component="register"]');
    if (!form) {
      throw new Error("register form not found");
    }
    let input = form.querySelector('input[name="relationship_status"]');
    if (!input) {
      input = document.createElement("input");
      input.type = "hidden";
      input.name = "relationship_status";
      form.appendChild(input);
    }
    (input as HTMLInputElement).value = "committed";
  });
  await page.getByPlaceholder("JohnWatts123").fill("Admin User");
  await page.getByPlaceholder("steamyjohn@diesel.com").fill(email);
  await page.getByPlaceholder("•••••••••").fill(password);
  await page.getByLabel("Birthdate (you need to be 18").fill("1990-01-01");
  await page.evaluate(() => {
    const form = document.querySelector('form[data-component="register"]');
    if (!form) {
      throw new Error("register form not found");
    }
    (form as HTMLFormElement).submit();
  });
  await page.waitForTimeout(1000);

  if (page.url().endsWith("/user/login")) {
    await page.getByLabel("Email address").fill(email);
    await page.getByLabel("Password").fill(password);
    await page.locator("#login-button").click();
    await page.waitForTimeout(1000);
  }
}
test("admin scaffold: critical surfaces stay behind admin auth", async ({ page, request }) => {
  test.setTimeout(120000);

  const adminEmail = (process.env.PAGODA_ADMIN_EMAILS ?? "admin@goship.test")
    .split(",")[0]
    .trim();
  const password = "Adminpass12345!";

  const unauthorized = await request.get("/auth/admin", { maxRedirects: 0 });
  expect(unauthorized.status()).toBe(303);
  expect(unauthorized.headers()["location"]).toBe("/user/login");

  await registerAdminUser(page, adminEmail, password);

  await page.goto("/auth/admin");
  await expect(page).toHaveURL(/\/auth\/admin\/managed-settings/);
  await expect(
    page.getByRole("heading", { name: "Admin - Managed Runtime Settings" })
  ).toBeVisible();

  await page.goto("/auth/admin/flags");
  await expect(
    page.getByRole("heading", { name: "Admin - Feature Flags" })
  ).toBeVisible();
  await expect(page.getByText("No feature flags found.")).toBeVisible();

  await page.goto("/auth/admin/trash");
  await expect(page.getByRole("heading", { name: "Admin - Trash" })).toBeVisible();
  await expect(page.getByText("No soft-deleted rows found.")).toBeVisible();
});
