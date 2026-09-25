import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { CampaignForm } from "./campaign-form";

vi.mock("next/navigation", () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock("./actions", () => ({ createCampaign: vi.fn() }));

describe("CampaignForm", () => {
  it("rejects an incomplete campaign before submission", async () => {
    render(<CampaignForm advertiserId="advertiser" placements={[{ code: "home_feed", display_name: "Home feed" }]} />);
    fireEvent.click(screen.getByRole("button", { name: "Create complete draft" }));
    expect(await screen.findAllByText("Invalid value")).toHaveLength(2);
  });
});
