import { expect, test } from "@playwright/test";

test("developer exercises fill, no-fill, and explanation in Decision Lab", async ({ page }) => {
  const suffix = `${Date.now()}`;
  await page.goto("/advertisers");
  await page.getByLabel("Advertiser name").fill(`Decision advertiser ${suffix}`);
  await page.getByRole("button", { name: "Create advertiser" }).click();
  await page.getByRole("link", { name: "Create campaign" }).click();
  await page.getByLabel("Campaign name").fill(`Decision campaign ${suffix}`);
  await page.getByLabel("Placement").selectOption("search_results");
  await page.getByLabel("Lifetime budget (minor units)").fill("1");
  await page.getByLabel("Country codes").fill("XZ");
  await page.getByRole("button", { name: "Create complete draft" }).click();
  await page.getByRole("button", { name: "Activate" }).click();
  await expect(page.getByText("ACTIVE", { exact: true })).toBeVisible();

  await page.getByRole("link", { name: "Decision Lab" }).click();
  await page.getByRole("button", { name: "Generate" }).click();
  await page.getByLabel("Placement").selectOption("search_results");
  await page.getByLabel("Country").fill("xz");
  await page.getByRole("button", { name: "Request decision" }).click();
  await expect(page.getByRole("heading", { name: "FILL" })).toBeVisible();
  await expect(page.getByText("XZ", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Explain decision" }).click();
  await expect(page.getByRole("heading", { name: "Diagnostic explanation" })).toBeVisible();
  await expect(page.getByText("eligible", { exact: true }).first()).toBeVisible();

  await page.getByLabel("Country").fill("XY");
  await page.getByRole("button", { name: "Request decision" }).click();
  await expect(page.getByRole("heading", { name: "NO_FILL" })).toBeVisible();
  await expect(page.getByText("No eligible campaign")).toBeVisible();
});
