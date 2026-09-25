import { expect, test } from "@playwright/test";

test("operator manages lifecycle and authoritative immediate spend", async ({ page }) => {
  const suffix = `${Date.now()}`;
  await page.goto("/advertisers");
  await page.getByLabel("Advertiser name").fill(`E2E advertiser ${suffix}`);
  await page.getByRole("button", { name: "Create advertiser" }).click();
  await expect(page).toHaveURL(/\/advertisers\/.+\/campaigns/);

  await page.getByRole("link", { name: "Create campaign" }).click();
  await page.getByLabel("Campaign name").fill(`E2E campaign ${suffix}`);
  await page.getByLabel("Placement").selectOption("home_feed");
  await page.getByLabel("Lifetime budget (minor units)").fill("1000");
  await page.getByLabel("Currency").selectOption("EUR");
  await page.getByLabel("Country codes").fill("DE, FR");
  await page.getByRole("button", { name: "Create complete draft" }).click();
  await expect(page).toHaveURL(/\/campaigns\/.+/);
  await expect(page.getByText("DRAFT", { exact: true })).toBeVisible();

  await page.getByLabel("Edit configured budget").fill("1500");
  await page.getByRole("button", { name: "Save budget" }).click();
  await expect(page.getByText("Configured budget").locator("..").getByText("1500 EUR minor units")).toBeVisible();

  await page.getByLabel("Edit country targeting").fill("GB, US");
  await page.getByRole("button", { name: "Save targeting" }).click();
  await page.getByRole("button", { name: "Activate" }).click();
  await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();

  await page.getByLabel("Consumption amount").fill("400");
  await page.getByRole("button", { name: "Commit spend" }).click();
  await expect(page.getByText("APPROVED", { exact: true })).toBeVisible();
  await expect(page.getByText("Committed spend").locator("..").getByText("400 EUR minor units", { exact: true })).toBeVisible();
  await expect(page.getByText("Remaining budget").locator("..").getByText("1100 EUR minor units", { exact: true })).toBeVisible();

  await page.getByLabel("Consumption amount").fill("1200");
  await page.getByRole("button", { name: "Commit spend" }).click();
  await expect(page.getByText("INSUFFICIENT_BUDGET", { exact: true })).toBeVisible();
  await expect(page.getByText("Committed spend").locator("..").getByText("400 EUR minor units", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Pause" }).click();
  await expect(page.getByText("PAUSED", { exact: true })).toBeVisible();
  await page.getByLabel("Edit placement").selectOption("search_results");
  await page.getByRole("button", { name: "Save placement" }).click();
  await expect(page.getByText("search_results", { exact: true })).toBeVisible();

  await page.getByRole("button", { name: "Resume" }).click();
  await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "End campaign" }).click();
  await expect(page.getByText("ENDED", { exact: true })).toBeVisible();

  await page.getByLabel("Consumption amount").fill("1");
  await page.getByRole("button", { name: "Commit spend" }).click();
  await expect(page.getByText("CAMPAIGN_NOT_ACTIVE", { exact: true })).toBeVisible();
  await expect(page.getByText(/terminal and cannot be edited/i)).toBeVisible();
  await expect(page.getByRole("button", { name: "Save name" })).toBeDisabled();
});
