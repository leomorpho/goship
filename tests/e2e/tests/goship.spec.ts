import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test";

async function expectAnonymousRedirect(
  request: APIRequestContext,
  path: string,
  expectedLocation: string,
  seamName: string
) {
  const response = await request.get(path, { maxRedirects: 0 });
  expect(
    response.status(),
    `${seamName} should redirect anonymous users instead of serving a partial page`
  ).toBe(303);
  expect(
    response.headers()["location"],
    `${seamName} should redirect to the canonical login entrypoint`
  ).toBe(expectedLocation);
}

async function expectVisible(locator: Locator, message: string) {
  await expect(locator, message).toBeVisible();
}

async function expectLandingPage(page: Page) {
  await test.step("landing page exposes the scaffolded auth entrypoints", async () => {
    await page.goto("/");
    await expectVisible(
      page.locator('[data-component="landing-page"]'),
      "landing route should render the canonical landing page shell"
    );
    await expectVisible(
      page.getByRole("link", { name: "Create account" }),
      "landing page should expose the registration CTA"
    );
    await expectVisible(
      page.getByRole("link", { name: "Log in" }),
      "landing page should expose the login CTA"
    );
  });
}

async function expectRegisterPage(page: Page) {
  await test.step("register entrypoint serves the scaffolded form contract", async () => {
    await page.goto("/user/register");
    const registerForm = page.locator('form[data-component="register"]');
    await expectVisible(registerForm, "register page should render the canonical register form");
    await expectVisible(
      registerForm.getByLabel("Display Name"),
      "register form should expose the display-name field"
    );
    await expectVisible(
      registerForm.getByLabel("Email address"),
      "register form should expose the email field"
    );
    await expectVisible(
      registerForm.getByLabel("Birthdate (you need to be 18"),
      "register form should expose the age-gated birthdate field"
    );
    await expectVisible(
      registerForm.getByRole("button", { name: "Register" }),
      "register form should expose the canonical submit button"
    );
  });
}

async function expectLoginPage(page: Page) {
  await test.step("login entrypoint serves the scaffolded form contract", async () => {
    await page.goto("/user/login");
    const loginForm = page.locator('form[data-component="login"]');
    await expectVisible(loginForm, "login page should render the canonical login form");
    await expectVisible(
      loginForm.getByLabel("Email address"),
      "login form should expose the email field"
    );
    await expectVisible(
      loginForm.getByLabel("Password"),
      "login form should expose the password field"
    );
    await expectVisible(
      loginForm.getByRole("button", { name: "Log in" }),
      "login form should expose the canonical submit button"
    );
  });
}

async function expectCounterIsland(page: Page, islandName: string, mountedComponent: string) {
  const island = page.locator(`[data-island="${islandName}"]`);
  await expectVisible(
    island,
    `${islandName} should be rendered in the framework islands demo markup`
  );
  await expect(
    island,
    `${islandName} should mount through the islands runtime rather than staying server-only`
  ).toHaveAttribute("data-island-mounted", "true");
  await expectVisible(
    island.locator(`[data-component="${mountedComponent}"]`),
    `${islandName} should hydrate into its framework-specific counter component`
  );
}

test("goship golden flows: public and auth entrypoints match the scaffold", async ({
  page,
  request,
}) => {
  const health = await request.get("/up");
  expect(health.ok(), "/up should stay healthy before the golden browser journey starts").toBeTruthy();

  await expectLandingPage(page);
  await expectRegisterPage(page);
  await expectLoginPage(page);
});

test("goship golden flows: anonymous users are redirected at protected seams", async ({
  request,
}) => {
  await expectAnonymousRedirect(request, "/auth/realtime", "/user/login", "realtime seam");
  await expectAnonymousRedirect(request, "/auth/admin", "/user/login", "admin seam");
});

test("goship golden flows: islands demo proves the runtime mount contract", async ({
  page,
  request,
}) => {
  const response = await request.get("/demo/islands");
  expect(response.ok(), "/demo/islands should stay routable as the framework islands demo").toBeTruthy();

  await page.goto("/demo/islands");
  await expectVisible(
    page.locator('[data-component="islands-demo-page"]'),
    "islands demo route should render the canonical framework islands page"
  );
  await expectVisible(
    page.locator('[data-slot="islands-regression-note"]'),
    "islands demo should keep the regression note that explains why these counters exist"
  );

  await expectCounterIsland(page, "VanillaCounter", "counter-vanilla");
  await expectCounterIsland(page, "ReactCounter", "counter-react");
  await expectCounterIsland(page, "VueCounter", "counter-vue");
  await expectCounterIsland(page, "SvelteCounter", "counter-svelte");
});
