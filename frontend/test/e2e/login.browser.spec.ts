import { expect, test } from "@playwright/test";

test("valid login", async ({ page }) => {
  await page.goto("/home");
  await expect(page).toHaveURL("/");
  await page.fill("input[type='text']", "demo@example.com");
  await page.fill("input[type='password']", "demodemo");
  await page.click("button[type='submit']");
  await expect(page).toHaveURL("/home");
});

test("invalid login", async ({ page }) => {
  await page.goto("/home");
  await expect(page).toHaveURL("/");
  await page.fill("input[type='text']", "dummy@example.com");
  await page.fill("input[type='password']", "dummy");
  await page.click("button[type='submit']");
  await page.waitForTimeout(500);
  await expect(page.locator("div[class*='login-error']").first()).toHaveText("Invalid email or password");
  await expect(page).toHaveURL("/");
});

test("self-registration does not exist once setup has completed", async ({ page }) => {
  // Users are created by administrators; the login page must offer no
  // registration entry point at all (a user already exists in demo mode, so
  // the first-time setup card is not shown either).
  await page.goto("/");
  await expect(page.locator("#login-form")).toBeVisible();
  await expect(page.getByTestId("register-button")).toHaveCount(0);
  await expect(page.locator("#setup-form")).toHaveCount(0);
});
