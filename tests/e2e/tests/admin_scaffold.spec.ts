import { execFileSync } from "node:child_process";
import path from "node:path";

import { expect, test } from "@playwright/test";

const repoRoot = path.resolve(process.cwd(), "..", "..");

type AuthCookie = {
  name: string;
  value: string;
  path: string;
  httpOnly: boolean;
};

function adminAuthCookie(email: string, password: string): AuthCookie {
  const raw = execFileSync("go", ["run", "./tests/e2e/scripts/admin_session.go"], {
    cwd: repoRoot,
    encoding: "utf8",
    env: {
      ...process.env,
      E2E_ADMIN_EMAIL: email,
      E2E_ADMIN_PASSWORD: password,
    },
  });

  return JSON.parse(raw) as AuthCookie;
}

test("admin scaffold: critical surfaces stay behind admin auth", async ({ request }) => {
  test.setTimeout(120000);

  const adminEmail = (process.env.PAGODA_ADMIN_EMAILS ?? "admin@goship.test")
    .split(",")[0]
    .trim();
  const password = "Adminpass12345!";

  const unauthorized = await request.get("/auth/admin", { maxRedirects: 0 });
  expect(unauthorized.status()).toBe(303);
  expect(unauthorized.headers()["location"]).toBe("/user/login");

  const cookie = adminAuthCookie(adminEmail, password);
  const authHeaders = { Cookie: `${cookie.name}=${cookie.value}` };

  for (const path of [
    "/auth/admin/managed-settings",
    "/auth/admin/flags",
    "/auth/admin/trash",
  ]) {
    const response = await request.get(path, { headers: authHeaders });
    expect(response.status(), `${path} should stay reachable for authenticated admins`).toBe(200);
  }
});
