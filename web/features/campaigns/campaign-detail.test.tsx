import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { CampaignDetail } from "./campaign-detail";

const refresh = vi.fn();
const updateName = vi.fn();

vi.mock("next/navigation", () => ({ useRouter: () => ({ refresh, push: vi.fn() }) }));
vi.mock("./actions", () => ({
  updateName: (...args: unknown[]) => updateName(...args),
  updateBudget: vi.fn(), updatePlacement: vi.fn(), updateTargeting: vi.fn(), transitionCampaign: vi.fn(),
}));

const campaign = {
  id: "00000000-0000-4000-8000-000000000001", advertiser_id: "00000000-0000-4000-8000-000000000002",
  name: "Lunch", state: "DRAFT" as const, placement_code: "home_feed",
  budget: { configured_amount_minor: "1000", currency: "EUR" as const }, targeting: { countries: ["DE"] },
  version: 3, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
};
const placements = [{ code: "home_feed", display_name: "Home feed" }];

describe("CampaignDetail", () => {
  beforeEach(() => { vi.clearAllMocks(); });

  it("shows stale-version guidance and does not silently retry", async () => {
    updateName.mockResolvedValue({ ok: false, stale: true, message: "Campaign changed since it was loaded" });
    render(<CampaignDetail campaign={campaign} placements={placements} />);
    fireEvent.change(screen.getByLabelText("Edit campaign name"), { target: { value: "Updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("changed elsewhere");
    expect(updateName).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "Reload current campaign" }));
    expect(refresh).toHaveBeenCalledOnce();
    expect(updateName).toHaveBeenCalledTimes(1);
  });

  it("keeps commands disabled until refreshed props contain the accepted version", async () => {
    updateName.mockResolvedValue({ ok: true, resourceVersion: 4 });
    const view = render(<CampaignDetail campaign={campaign} placements={placements} />);
    fireEvent.change(screen.getByLabelText("Edit campaign name"), { target: { value: "Updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Save name" }));

    await waitFor(() => expect(refresh).toHaveBeenCalledOnce());
    expect(screen.getByRole("button", { name: "Activate" })).toBeDisabled();

    view.rerender(<CampaignDetail campaign={{ ...campaign, name: "Updated", version: 4 }} placements={placements} />);
    await waitFor(() => expect(screen.getByRole("button", { name: "Activate" })).toBeEnabled());
  });

  it("renders terminal campaigns with all editors disabled", async () => {
    render(<CampaignDetail campaign={{ ...campaign, state: "ENDED" }} placements={placements} />);
    expect(screen.getByText(/terminal and cannot be edited/i)).toBeInTheDocument();
    await waitFor(() => expect(screen.getByRole("button", { name: "Save name" })).toBeDisabled());
    expect(screen.queryByRole("button", { name: "Activate" })).not.toBeInTheDocument();
  });
});
